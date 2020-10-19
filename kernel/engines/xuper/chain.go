package xuper

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines"
)

type XuperChain struct {
	// 链级上下文
	sysCtx engines.ChainCtx
	// 交易处理器
	txProc TxProcessor
	// 矿工
	miner Miner
}

func CreateChain() (*XuperChain, error) {

}

func LoadChain(path string) (*XuperChain, error) {

}

func (t *XuperChain) SubmitTx() {

}

func (t *XuperChain) Start() {

}

func (t *XuperChain) Stop() {

}
