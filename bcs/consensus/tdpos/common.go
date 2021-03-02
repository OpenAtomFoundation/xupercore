package tdpos

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"
)

const (
	MAXSLEEPTIME        = 1000
	MAXMAPSIZE          = 1000
	MAXHISPROPOSERSSIZE = 100

	contractNominateCandidate = "nominateCandidate"
	contractRevokeCandidata   = "revokeNominate"
	contractVote              = "voteCandidate"
	contractRevokeVote        = "revokeVote"

	contractBucket  = "$tdpos"
	nominateKey     = "nominate"
	voteKeyPrefix   = "vote_"
	revokeKeyPrefix = "revoke_"
	urlmapKey       = "urlmap"

	StatusOK  = 200
	StatusErr = 500

	NOMINATETYPE = "nominate"
	VOTETYPE     = "vote"
)

var (
	InitProposerNeturlErr         = errors.New("Init proposer neturl is invalid.")
	ProposerNumErr                = errors.New("Proposer num isn't equal to proposer neturl.")
	NeedNetURLErr                 = errors.New("Init proposer neturl must be mentioned.")
	invalidProposerErr            = errors.New("Invalid proposer.")
	invalidTermErr                = errors.New("Invalid term.")
	proposeBlockMoreThanConfigErr = errors.New("Propose block more than config num error.")
	timeoutBlockErr               = errors.New("New block is out of date.")

	MinerSelectErr   = errors.New("Node isn't a miner, calculate error.")
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
	InvalidQC        = errors.New("QC struct is invalid.")

	proposerNotEnoughErr = errors.New("Term publish proposer num less than config.")
	heightTooLow         = errors.New("The target height is lower than 4.")

	tooLowHeight      = errors.New("TipHeight < 3, use init parameters.")
	nominateAddrErr   = errors.New("Addr in nominate candidate tx can not be empty.")
	nominateUrlErr    = errors.New("NetUrl in nominate candidate tx can not be empty.")
	emptyVoteAddrErr  = errors.New("Addr in vote candidate tx can not be empty.")
	voteNominateErr   = errors.New("Addr in vote candidate hasn't been nominated.")
	amountErr         = errors.New("Amount in contract can not be empty.")
	authErr           = errors.New("candidate has not been authenticated")
	repeatNominateErr = errors.New("The candidate had been nominate.")
	emptyNominateKey  = errors.New("No valid candidate key when revoke.")
	notFoundErr       = errors.New("Value not found, please check your input parameters.")
)

// tdpos 共识机制的配置
type tdposConfig struct {
	Version int64 `json:"version,omitempty"`
	// 每轮选出的候选人个数
	ProposerNum int64 `json:"proposer_num"`
	// 出块间隔
	Period int64 `json:"period"`
	// 更换候选人时间间隔
	AlternateInterval int64 `json:"alternate_interval"`
	// 更换轮时间间隔
	TermInterval int64 `json:"term_interval"`
	// 每轮每个候选人最多出多少块
	BlockNum int64 `json:"block_num"`
	// 投票单价
	VoteUnitPrice *big.Int `json:"vote_unit_price"`
	// 初始时间
	InitTimestamp int64 `json:"timestamp"`
	// 系统指定的前两轮的候选人名单
	InitProposer       map[string][]string `json:"init_proposer"`
	InitProposerNeturl map[string][]string `json:"init_proposer_neturl"`
	// json支持两种格式的解析形式
	NeedNetURL bool            `json:"need_neturl"`
	EnableBFT  map[string]bool `json:"bft_config,omitempty"`
}

func (tp *tdposConsensus) needSync() bool {
	tipBlock := tp.election.ledger.GetTipBlock()
	if tipBlock.GetHeight() == 0 {
		return true
	}
	if string(tipBlock.GetProposer()) == string(tp.election.address) {
		return false
	}
	return true
}

// unmarshalTdposConfig 解析xpoaconfig
func unmarshalTdposConfig(input []byte) (*tdposConfig, error) {
	// 由于创世块中的配置全部使用的string，内部使用时做下转换
	// 转换配置结构到内部结构
	xconfig, err := buildConfigs(input)
	if err != nil {
		return nil, err
	}

	if xconfig.InitProposerNeturl != nil {
		if _, ok := (xconfig.InitProposerNeturl)["1"]; !ok {
			return nil, InitProposerNeturlErr
		}
		if int64(len((xconfig.InitProposerNeturl)["1"])) != xconfig.ProposerNum {
			return nil, ProposerNumErr
		}
		return xconfig, nil
	}
	if xconfig.NeedNetURL {
		return nil, NeedNetURLErr
	}
	return xconfig, nil
}

func buildConfigs(input []byte) (*tdposConfig, error) {
	// 先转为interface{}
	consCfg := make(map[string]interface{})
	err := json.Unmarshal(input, &consCfg)
	if err != nil {
		return nil, err
	}

	// assemble consensus config
	tdposCfg := &tdposConfig{}

	// int64统一转换
	int64Map := map[string]int64{
		"version":            0,
		"proposer_num":       0,
		"period":             0,
		"alternate_interval": 0,
		"term_interval":      0,
		"block_num":          0,
		"timestamp":          0,
	}
	for k, _ := range int64Map {
		if _, ok := consCfg[k]; !ok {
			if k == "version" {
				continue
			}
			return nil, fmt.Errorf("marshal consensus config failed key %s unset", k)
		}

		value, err := strconv.ParseInt(consCfg[k].(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("marshal consensus config failed key %s set error", k)
		}
		int64Map[k] = value
	}
	tdposCfg.Version = int64Map["version"]
	tdposCfg.ProposerNum = int64Map["proposer_num"]
	tdposCfg.Period = int64Map["period"]
	tdposCfg.AlternateInterval = int64Map["alternate_interval"]
	tdposCfg.TermInterval = int64Map["term_interval"]
	tdposCfg.BlockNum = int64Map["block_num"]
	tdposCfg.InitTimestamp = int64Map["timestamp"]

	// 转换其他特殊结构
	voteUnitPrice := big.NewInt(0)
	if _, ok := voteUnitPrice.SetString(consCfg["vote_unit_price"].(string), 10); !ok {
		return nil, fmt.Errorf("vote_unit_price set error")
	}
	tdposCfg.VoteUnitPrice = voteUnitPrice

	type tempStruct struct {
		InitProposer       map[string][]string `json:"init_proposer"`
		InitProposerNeturl map[string][]string `json:"init_proposer_neturl"`
		EnableBFT          map[string]bool     `json:"bft_config,omitempty"`
		NeedNetURL         bool                `json:"need_neturl"`
	}
	var temp tempStruct
	err = json.Unmarshal(input, &temp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal to temp struct failed.err:%v", err)
	}
	tdposCfg.InitProposer = temp.InitProposer
	tdposCfg.InitProposerNeturl = temp.InitProposerNeturl
	tdposCfg.EnableBFT = temp.EnableBFT
	tdposCfg.NeedNetURL = temp.NeedNetURL

	return tdposCfg, nil
}

func cleanProduceMap(isProduce map[int64]bool, period int64) {
	// 删除已经落盘的所有key
	if len(isProduce) <= MAXMAPSIZE {
		return
	}
	t := time.Now().UnixNano()
	key := t / period
	for k, _ := range isProduce {
		if k < key-MAXMAPSIZE {
			delete(isProduce, k)
		}
	}
}

// 每个地址每一轮的总票数
type termBallots struct {
	Address string
	Ballots int64
}

type termBallotsSlice []*termBallots

func (tv termBallotsSlice) Len() int {
	return len(tv)
}

func (tv termBallotsSlice) Swap(i, j int) {
	tv[i], tv[j] = tv[j], tv[i]
}

func (tv termBallotsSlice) Less(i, j int) bool {
	if tv[j].Ballots == tv[i].Ballots {
		return tv[j].Address < tv[i].Address
	}
	return tv[j].Ballots < tv[i].Ballots
}

type historyProposer struct {
	height    int64
	proposers []string
}

type historyProposers []historyProposer

func (s historyProposers) Len() int {
	return len(s)
}

func (s historyProposers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s historyProposers) Less(i, j int) bool {
	return s[i].height < s[j].height
}
