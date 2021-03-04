package xpoa

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	MinerSelectErr   = errors.New("Node isn't a miner, calculate error.")
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
	InvalidQC        = errors.New("QC struct is invalid.")
	targetParamErr   = errors.New("Target paramters are invalid, please check them.")
	tooLowHeight     = errors.New("The height should be higher than 3.")
)

const (
	contractBucket       = "$xpoa"
	validateKeys         = "validates"
	contractGetValidates = "getValidates"
	contractEditValidate = "editValidates"

	fee = 1000

	statusOK         = 200
	statusBadRequest = 400
	statusErr        = 500

	MAXSLEEPTIME = 1000
	MAXMAPSIZE   = 1000
)

type xpoaConfig struct {
	Version int64 `json:"version,omitempty"`
	// 每个候选人每轮出块个数
	BlockNum int64 `json:"block_num"`
	// 单位为毫秒
	Period       int64          `json:"period"`
	InitProposer []ProposerInfo `json:"init_proposer"`

	EnableBFT map[string]bool `json:"bft_config,omitempty"`
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

type ProposerInfo struct {
	Address string `json:"address"`
	Neturl  string `json:"neturl"`
}

// LoadValidatorsMultiInfo
// xpoa 格式为
// { "validators": [$ADDR_STRING...] }
func loadValidatorsMultiInfo(res []byte) ([]string, error) {
	if res == nil {
		return nil, NotValidContract
	}
	// 读取最新的validators值
	contractInfo := ValidatorsInfo{}
	if err := json.Unmarshal(res, &contractInfo); err != nil {
		return nil, err
	}
	return contractInfo.Validators, nil
}
