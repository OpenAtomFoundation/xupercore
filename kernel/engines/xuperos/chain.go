package xuperos

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
)

type XuperOSChain struct {
	// 链级上下文
	chainCtx def.ChainCtx
}

func LoadChain(dataDir string) (*XuperOSChain, error) {
	return nil, fmt.Errorf("the interface is not implemented")
}

func (t *XuperOSChain) Start() {

}

func (t *XuperOSChain) Stop() {

}

func (t *XuperOSChain) ProcessTx() {

}

func (t *XuperOSChain) ProcessBlock() {

}

func (t *XuperOSChain) PreExec() {

}

func (t *XuperOSChain) GetChainCtx() *def.ChainCtx {
	return nil
}
