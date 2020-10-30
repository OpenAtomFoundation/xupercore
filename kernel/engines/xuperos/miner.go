// 矿工
// 1.触发同步区块检查 2.共识或者出块权利时打包出块
package xuperos

import (
	"github.com/xuperchain/xupercore/kernel/engines"
)

type Miner interface {
	Start() error
	Stop()
}

type MinerImpl struct {
}

func NewMiner() (Miner, error) {

}

func (t *MinerImpl) Start() error {

}

func (t *MinerImpl) Stop() {

}
