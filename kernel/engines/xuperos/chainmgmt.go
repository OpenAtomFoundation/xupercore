package xuperos

import (
	"fmt"
	"sync"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
)

// xuperos执行引擎，为公链场景订制区块链引擎
type ChainMgmtImpl struct {
	// 链实例
	chains sync.Map
	engCtx *common.EngineCtx
}

func (m *ChainMgmtImpl) Get(chainName string) (Chain, error) {
	return Chain{}, nil
}

func (m *ChainMgmtImpl) GetChains() []string {
	return nil
}

func (m *ChainMgmtImpl) Register(chainName string) error {
	chain, err := LoadChain(m.engCtx, chainName)
	if err != nil {
		m.engCtx.XLog.Error("load chain failed", "error", err, "chain_name", chainName)
		return fmt.Errorf("load chain failed")
	}
	m.chains.Store(chainName, chain)
	go chain.Start()
	return nil
}
