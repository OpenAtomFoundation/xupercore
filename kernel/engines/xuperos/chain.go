package xuperos

import (
	"github.com/xuperchain/xupercore/kernel/engines"
)

type XuperOSChain struct {
	// 链级上下文
	sysCtx engines.ChainCtx
	// 交易处理器
	txProc TxProcessor
	// 矿工
	miner Miner
}

func CreateChain() (*XuperOSChain, error) {

}

func LoadChain(path string) (*XuperOSChain, error) {

}

func (t *XuperOSChain) SubmitTx() {

}

func (t *XuperOSChain) Start() {

}

func (t *XuperOSChain) Stop() {

}
