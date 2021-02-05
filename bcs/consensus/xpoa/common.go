package xpoa

import (
	"encoding/json"
	"sync"
	"time"
)

const MAXMAPSIZE = 1000

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
// { "proposers": [{"Address":$STRING, "PeerAddr":$STRING}...] }
func loadValidatorsMultiInfo(res []byte, addrToNet *map[string]string, mutex *sync.Mutex) ([]string, error) {
	if res == nil {
		return nil, NotValidContract
	}
	// 读取最新的validators值
	contractInfo := ProposerInfos{}
	if err := json.Unmarshal(res, &contractInfo); err != nil {
		return nil, err
	}
	var validators []string
	for _, node := range contractInfo.Proposers {
		validators = append(validators, node.Address)
		if _, ok := (*addrToNet)[node.Address]; !ok {
			mutex.Lock()
			(*addrToNet)[node.Address] = node.Neturl
			mutex.Unlock()
		}
	}
	return validators, nil
}

type ProposerInfos struct {
	Proposers []NodeInfo `json:"proposers"`
}

type NodeInfo struct {
	Address string `json:"Address"`
	Neturl  string `json:"PeerAddr"`
}
