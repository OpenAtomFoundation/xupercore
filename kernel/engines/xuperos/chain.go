package xuperos

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

type XuperOSChain struct {
	// 链级上下文
	chainCtx *def.ChainCtx
	// log
	log logs.Logger
	// 矿工
	miner *miner
}

// 从本地存储加载链
func LoadChain(dataDir string) (*XuperOSChain, error) {
	// 初始化链环境上下文

	// 注册合约

	// 注册VAT

	// 实例化矿工

	return nil, fmt.Errorf("the interface is not implemented")
}

func (t *XuperOSChain) Start() {
	// 启动矿工
}

func (t *XuperOSChain) Stop() {
	// 停止矿工

	// 释放资源
}

// 交易和区块结构由账本定义
func (t *XuperOSChain) ProcTx() {
	// (账本)验证交易

	//（账本）提交交易

}

// 处理新区块
func (t *XuperOSChain) ProcBlocks() {

}

// 交易预执行
func (t *XuperOSChain) PreExec() {

}

func (t *XuperOSChain) GetChainCtx() *def.ChainCtx {
	return t.chainCtx
}
