package xuperos

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
)

type XuperOSChain struct {
	// 链级上下文
	chainCtx *def.ChainCtx
	// log
	log logs.Logger
	// 矿工
	miner *miner
	// 依赖代理组件
	relyAgent def.ChainRelyAgent
}

// 从本地存储加载链
func LoadChain(dataDir string) (*XuperOSChain, error) {
	// 初始化链环境上下文

	// 注册合约
	RegisterKernMethod()

	// 注册VAT

	// 实例化矿工

	return nil, fmt.Errorf("the interface is not implemented")
}

// 供单测时设置rely agent为mock agent，非并发安全
func (t *XuperOSChain) SetRelyAgent(agent def.ChainRelyAgent) error {
	if agent == nil {
		return fmt.Errorf("param error")
	}

	t.relyAgent = agent
	return nil
}

func (t *XuperOSChain) Start() {
	// 启动矿工
	t.miner.start()
}

func (t *XuperOSChain) Stop() {
	// 停止矿工
	t.miner.stop()

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
