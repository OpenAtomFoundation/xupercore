package xpoa

import (
	"time"

	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
)

type xpoaConfig struct {
	InitProposer []chainedBft.ProposerInfo `json:"init_proposer"`
	BlockNum     int64                     `json:"block_num"`
	// 单位为毫秒
	Period    int64            `json:"period"`
	Version   int64            `json:"version"`
	EnableBFT *map[string]bool `json:"bft_config,omitempty"`
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
		if k < key-3 {
			delete(isProduce, k)
		}
	}
}
