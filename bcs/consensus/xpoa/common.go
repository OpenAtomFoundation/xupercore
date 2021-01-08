package xpoa

import (
	"encoding/json"
	"strings"
	"time"

	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
)

type xpoaConfig struct {
	InitProposer []ProposerInfo `json:"init_proposer"`
	BlockNum     int64          `json:"block_num"`
	// 单位为毫秒
	Period  int64 `json:"period"`
	Version int64 `json:"version"`
}

// XpoaStorage xpoa占用block中consensusStorage json串的格式
type XpoaStorage struct {
	Justify *chainedBft.QuorumCert `json:"Justify,omitempty"`
}

type ProposerInfo struct {
	Address string `json:"address"`
	Neturl  string `json:"neturl"`
}

func cleanProduceMap(isProduce map[int64]bool, period int64) {
	// 删除已经落盘的所有key
	t := time.Now().UnixNano()
	key := t / period
	for k, _ := range isProduce {
		if k < key-3 {
			delete(isProduce, k)
		}
	}
}

func loadValidatorsMultiInfo(res []byte, addrToNet *map[string]string) ([]string, error) {
	if res == nil {
		return nil, NotValidContract
	}
	// 读取最新的validators值
	contractInfo := ProposerInfo{}
	if err := json.Unmarshal(res, &contractInfo); err != nil {
		return nil, err
	}
	validators := strings.Split(contractInfo.Address, ";") // validators由分号隔开
	if len(validators) == 0 {
		return nil, EmptyValidors
	}
	neturls := strings.Split(contractInfo.Neturl, ";") // neturls由分号隔开
	if len(neturls) != len(validators) {
		return nil, EmptyValidors
	}
	for i, v := range validators {
		(*addrToNet)[v] = neturls[i]
	}
	return validators, nil
}
