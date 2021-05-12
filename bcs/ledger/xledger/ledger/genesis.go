package ledger

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/xuperchain/xupercore/lib/cache"
	"github.com/xuperchain/xupercore/protos"
)

// awardCacheSize system award cache, in avoid of double computing
const awardCacheSize = 1000

// RootConfig genesis block configure
type RootConfig struct {
	Version   string `json:"version"`
	Crypto    string `json:"crypto"`
	Kvengine  string `json:"kvengine"`
	Consensus struct {
		Type  string `json:"type"`
		Miner string `json:"miner"`
	} `json:"consensus"`
	Predistribution []struct {
		Address string `json:"address"`
		Quota   string `json:"quota"`
	}
	// max block size in MB
	MaxBlockSize string `json:"maxblocksize"`
	Period       string `json:"period"`
	NoFee        bool   `json:"nofee"`
	Award        string `json:"award"`
	AwardDecay   struct {
		HeightGap int64   `json:"height_gap"`
		Ratio     float64 `json:"ratio"`
	} `json:"award_decay"`
	GasPrice struct {
		CpuRate  int64 `json:"cpu_rate"`
		MemRate  int64 `json:"mem_rate"`
		DiskRate int64 `json:"disk_rate"`
		XfeeRate int64 `json:"xfee_rate"`
	} `json:"gas_price"`
	Decimals          string                 `json:"decimals"`
	GenesisConsensus  map[string]interface{} `json:"genesis_consensus"`
	ReservedContracts []InvokeRequest        `json:"reserved_contracts"`
	ReservedWhitelist struct {
		Account string `json:"account"`
	} `json:"reserved_whitelist"`
	ForbiddenContract InvokeRequest `json:"forbidden_contract"`
	// NewAccountResourceAmount the amount of creating a new contract account
	NewAccountResourceAmount int64 `json:"new_account_resource_amount"`
	// IrreversibleSlideWindow
	IrreversibleSlideWindow string `json:"irreversibleslidewindow"`
	// GroupChainContract
	GroupChainContract InvokeRequest `json:"group_chain_contract"`
}

// GasPrice define gas rate for utxo
type GasPrice struct {
	CpuRate  int64 `json:"cpu_rate" mapstructure:"cpu_rate"`
	MemRate  int64 `json:"mem_rate" mapstructure:"mem_rate"`
	DiskRate int64 `json:"disk_rate" mapstructure:"disk_rate"`
	XfeeRate int64 `json:"xfee_rate" mapstructure:"xfee_rate"`
}

type Predistribution struct {
	Address string `json:"address"`
	Quota   string `json:"quota"`
}

// InvokeRequest define genesis reserved_contracts configure
type InvokeRequest struct {
	ModuleName   string            `json:"module_name" mapstructure:"module_name"`
	ContractName string            `json:"contract_name" mapstructure:"contract_name"`
	MethodName   string            `json:"method_name" mapstructure:"method_name"`
	Args         map[string]string `json:"args" mapstructure:"args"`
}

func InvokeRequestFromJSON2Pb(jsonRequest []InvokeRequest) ([]*protos.InvokeRequest, error) {
	requestsWithPb := []*protos.InvokeRequest{}
	for _, request := range jsonRequest {
		tmpReqWithPB := &protos.InvokeRequest{
			ModuleName:   request.ModuleName,
			ContractName: request.ContractName,
			MethodName:   request.MethodName,
			Args:         make(map[string][]byte),
		}
		for k, v := range request.Args {
			tmpReqWithPB.Args[k] = []byte(v)
		}
		requestsWithPb = append(requestsWithPb, tmpReqWithPB)
	}
	return requestsWithPb, nil
}

func (rc *RootConfig) GetCryptoType() string {
	if rc.Crypto != "" {
		return rc.Crypto
	}

	return "default"
}

// GetIrreversibleSlideWindow get irreversible slide window
func (rc *RootConfig) GetIrreversibleSlideWindow() int64 {
	irreversibleSlideWindow, _ := strconv.Atoi(rc.IrreversibleSlideWindow)
	return int64(irreversibleSlideWindow)
}

// GetMaxBlockSizeInByte get max block size in Byte
func (rc *RootConfig) GetMaxBlockSizeInByte() (n int64) {
	maxSizeMB, _ := strconv.Atoi(rc.MaxBlockSize)
	n = int64(maxSizeMB) << 20
	return
}

// GetNewAccountResourceAmount get the resource amount of new an account
func (rc *RootConfig) GetNewAccountResourceAmount() int64 {
	return rc.NewAccountResourceAmount
}

// GetGenesisConsensus get consensus config of genesis block
func (rc *RootConfig) GetGenesisConsensus() (map[string]interface{}, error) {
	if rc.GenesisConsensus == nil {
		consCfg := map[string]interface{}{}
		consCfg["name"] = rc.Consensus.Type
		consCfg["config"] = map[string]interface{}{
			"miner":  rc.Consensus.Miner,
			"period": rc.Period,
		}
		return consCfg, nil
	}
	return rc.GenesisConsensus, nil
}

// GetReservedContract get default contract config of genesis block
func (rc *RootConfig) GetReservedContract() ([]*protos.InvokeRequest, error) {
	return InvokeRequestFromJSON2Pb(rc.ReservedContracts)
}

func (rc *RootConfig) GetForbiddenContract() ([]*protos.InvokeRequest, error) {
	return InvokeRequestFromJSON2Pb([]InvokeRequest{rc.ForbiddenContract})
}

func (rc *RootConfig) GetGroupChainContract() ([]*protos.InvokeRequest, error) {
	return InvokeRequestFromJSON2Pb([]InvokeRequest{rc.GroupChainContract})
}

// GetReservedWhitelistAccount return reserved whitelist account
func (rc *RootConfig) GetReservedWhitelistAccount() string {
	return rc.ReservedWhitelist.Account
}

// GetPredistribution return predistribution
func (rc *RootConfig) GetPredistribution() []Predistribution {
	return PredistributionTranslator(rc.Predistribution)
}

func PredistributionTranslator(predistribution []struct {
	Address string `json:"address"`
	Quota   string `json:"quota"`
}) []Predistribution {
	var predistributionArray []Predistribution
	for _, pd := range predistribution {
		ps := Predistribution{
			Address: pd.Address,
			Quota:   pd.Quota,
		}
		predistributionArray = append(predistributionArray, ps)
	}
	return predistributionArray
}

// GenesisBlock genesis block data structure
type GenesisBlock struct {
	config     *RootConfig
	awardCache *cache.LRUCache
}

// NewGenesisBlock new a genesis block
func NewGenesisBlock(genesisCfg []byte) (*GenesisBlock, error) {
	if len(genesisCfg) < 1 {
		return nil, fmt.Errorf("genesis config is empty")
	}

	// 加载配置
	config := &RootConfig{}
	jsErr := json.Unmarshal(genesisCfg, config)
	if jsErr != nil {
		return nil, jsErr
	}
	if config.NoFee {
		config.Award = "0"
		config.NewAccountResourceAmount = 0
		// nofee场景下，不需要原生代币xuper
		// 但是治理代币，会从此配置中进行初始代币发行，故而保留config.Predistribution内容
		//config.Predistribution = []struct {
		//	Address string `json:"address"`
		//	Quota   string `json:"quota"`
		//}{}
		config.GasPrice.CpuRate = 0
		config.GasPrice.DiskRate = 0
		config.GasPrice.MemRate = 0
		config.GasPrice.XfeeRate = 0
	}

	gb := &GenesisBlock{
		awardCache: cache.NewLRUCache(awardCacheSize),
		config:     config,
	}

	return gb, nil
}

// GetConfig get config of genesis block
func (gb *GenesisBlock) GetConfig() *RootConfig {
	return gb.config
}

// CalcAward calc system award by block height
func (gb *GenesisBlock) CalcAward(blockHeight int64) *big.Int {
	award := big.NewInt(0)
	award.SetString(gb.config.Award, 10)
	if gb.config.AwardDecay.HeightGap == 0 { //无衰减策略
		return award
	}
	period := blockHeight / gb.config.AwardDecay.HeightGap
	if awardRemember, ok := gb.awardCache.Get(period); ok {
		return awardRemember.(*big.Int) //加个记忆，避免每次都重新算
	}
	var realAward = float64(award.Int64())
	for i := int64(0); i < period; i++ { //等比衰减
		realAward = realAward * gb.config.AwardDecay.Ratio
	}
	N := int64(math.Round(realAward)) //四舍五入
	award.SetInt64(N)
	gb.awardCache.Add(period, award)
	return award
}

// GetGasPrice get gas rate for different resource(cpu, mem, disk and xfee)
func (rc *RootConfig) GetGasPrice() *protos.GasPrice {
	gasPrice := &protos.GasPrice{
		CpuRate:  rc.GasPrice.CpuRate,
		MemRate:  rc.GasPrice.MemRate,
		DiskRate: rc.GasPrice.DiskRate,
		XfeeRate: rc.GasPrice.XfeeRate,
	}
	return gasPrice
}
