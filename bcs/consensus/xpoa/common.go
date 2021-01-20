package xpoa

import (
	"encoding/json"
	"sync"
	"time"

	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
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

// XpoaStorage xpoa占用block中consensusStorage json串的格式
type XpoaStorage struct {
	Justify *chainedBft.QuorumCert `json:"Justify,omitempty"`
}

func cleanProduceMap(isProduce map[int64]bool, period int64) {
	// 删除已经落盘的所有key
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
	Neturl  string `json:"peerAddr"`
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
