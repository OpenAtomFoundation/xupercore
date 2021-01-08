package tdpos

import (
	"encoding/json"
	"time"

	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
)

// 目前未定义pb结构
// TdposStorage tdpos占用block中consensusStorage json串的格式
type TdposStorage struct {
	CurTerm     int64                  `json:"curTerm,omitempty"`
	CurBlockNum int64                  `json:"curBlockNum,omitempty"`
	Justify     *chainedBft.QuorumCert `json:"Justify,omitempty"`
}

type ProposerInfo struct {
	Address string `json:"address"`
	Neturl  string `json:"neturl"`
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
	xconfig := &tdposConfig{}
	err := json.Unmarshal(input, xconfig)
	if err != nil {
		return nil, err
	}
	if xconfig.InitProposerNeturl != nil {
		if _, ok := (*xconfig.InitProposerNeturl)["1"]; !ok {
			return nil, InitProposerNeturlErr
		}
		if int64(len((*xconfig.InitProposerNeturl)["1"])) != xconfig.ProposerNum {
			return nil, ProposerNumErr
		}
		return xconfig, nil
	}
	if xconfig.NeedNetURL {
		return nil, NeedNetURLErr
	}
	return xconfig, nil
}

func cleanProduceMap(isProduce map[int64]bool, period int64, enableBFT bool) {
	if !enableBFT {
		return
	}
	// 删除已经落盘的所有key
	t := time.Now().UnixNano()
	key := t / period
	for k, _ := range isProduce {
		if k < key-3 {
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
