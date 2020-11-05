// 矿工
// 1.触发同步区块检查
// 2.共识或者出块权利时打包出块
package xuperos

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// 封装矿工这个角色的所有行为
type miner struct {
	log logs.Logger
}

func NewMiner(ctx *def.ChainCtx) *miner {
	obj := &miner{
		log: ctx.XLog,
	}

	return obj
}

// 启动矿工
func (t *miner) start() {

}

// 停止矿工
func (t *miner) stop() {

}

// 实现矿工行为(同步区块或者生产区块)
func (t *miner) miner() {

}

// 生产区块
func (t *miner) produceBlock() {

}

// 检查同步区块
func (t *miner) checkSyncBlock() {

}

// 广播新区块
func (t *miner) broadcastBlock() {

}
