package xpoa

import (
	"encoding/json"
	"errors"
)

var (
	MinerSelectErr   = errors.New("Node isn't a miner, calculate error.")
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
	InvalidQC        = errors.New("QC struct is invalid.")
	targetParamErr   = errors.New("Target paramters are invalid, please check them.")
	tooLowHeight     = errors.New("The height should be higher than 3.")
	aclErr           = errors.New("Xpoa needs valid acl account.")
	scheduleErr      = errors.New("minerScheduling overflow")
)

const (
	xpoaBucket           = "$xpoa"
	poaBucket            = "$poa"
	validateKeys         = "validates"
	contractGetValidates = "getValidates"
	contractEditValidate = "editValidates"

	fee = 1000

	MAXSLEEPTIME = 1000
	MAXMAPSIZE   = 1000
)

type xpoaConfig struct {
	Version int64 `json:"version,omitempty"`
	// 每个候选人每轮出块个数
	BlockNum int64 `json:"block_num"`
	// 单位为毫秒
	Period       int64        `json:"period"`
	InitProposer ProposerInfo `json:"init_proposer"`

	EnableBFT map[string]bool `json:"bft_config,omitempty"`
}

type ProposerInfo struct {
	Address []string `json:"address"`
}

// LoadValidatorsMultiInfo
// xpoa 格式为
// { "address": [$ADDR_STRING...] }
func loadValidatorsMultiInfo(res []byte) ([]string, error) {
	if res == nil {
		return nil, NotValidContract
	}
	// 读取最新的validators值
	contractInfo := ProposerInfo{}
	if err := json.Unmarshal(res, &contractInfo); err != nil {
		return nil, err
	}
	return contractInfo.Address, nil
}

func Find(a string, t []string) bool {
	for _, v := range t {
		if a != v {
			continue
		}
		return true
	}
	return false
}

func CalFault(input, sum int64) bool {
	// 根据3f+1, 计算最大恶意节点数
	f := (sum - 1) / 3
	if f < 0 {
		return false
	}
	if f == 0 {
		return input >= sum/2+1
	}
	return input >= (sum-f)/2+1
}

// 每个地址每一轮的总票数
type aksItem struct {
	Address string
	Weight  float64
}

type aksSlice []aksItem

func (a aksSlice) Len() int {
	return len(a)
}

func (a aksSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a aksSlice) Less(i, j int) bool {
	if a[j].Weight == a[i].Weight {
		return a[j].Address < a[i].Address
	}
	return a[j].Weight < a[i].Weight
}
