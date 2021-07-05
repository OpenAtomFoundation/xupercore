package xuperos

import (
	"errors"
	"fmt"
	"sync"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
)

// ChainMgmtImpl 用于管理链操作
type ChainManagerImpl struct {
	// 链实例
	chains sync.Map
	engCtx *common.EngineCtx
	log    logs.Logger
}

func (m *ChainManagerImpl) Get(chainName string) (Chain, error) {
	c, ok := m.chains.Load(chainName)
	if !ok {
		return Chain{}, errors.New("target chainName doesn't exist")
	}
	if _, ok := c.(*Chain); !ok {
		return Chain{}, errors.New("transfer to Chain pointer error")
	}
	chainPtr := c.(*Chain)
	return *chainPtr, nil
}

func (m *ChainManagerImpl) Put(chainName string, chain *Chain) {
	m.chains.Store(chainName, chain)
}

func (m *ChainManagerImpl) GetChains() []string {
	var chains []string
	m.chains.Range(func(key, value interface{}) bool {
		cname, ok := key.(string)
		if !ok {
			return false
		}
		chains = append(chains, cname)
		return true
	})
	return chains
}

func (m *ChainManagerImpl) StartChains(wg *sync.WaitGroup) {
	m.chains.Range(func(k, v interface{}) bool {
		chainHD := v.(common.Chain)
		m.log.Trace("start chain " + k.(string))

		wg.Add(1)
		go func() {
			defer wg.Done()

			m.log.Trace("run chain " + k.(string))
			// 启动链
			chainHD.Start()
			m.log.Trace("chain " + k.(string) + " exit")
		}()

		return true
	})
}

func (m *ChainManagerImpl) StopChains(wg *sync.WaitGroup) {
	m.chains.Range(func(k, v interface{}) bool {
		chainHD := v.(common.Chain)

		m.log.Trace("stop chain " + k.(string))
		wg.Add(1)
		// 关闭链
		chainHD.Stop()
		wg.Done()
		m.log.Trace("chain " + k.(string) + " closed")

		return true
	})
}

func (m *ChainManagerImpl) LoadChain(chainName string) error {
	chain, err := LoadChain(m.engCtx, chainName)
	if err != nil {
		m.engCtx.XLog.Error("load chain failed", "error", err, "chain_name", chainName)
		return fmt.Errorf("load chain failed")
	}
	m.Put(chainName, chain)
	go chain.Start()
	return nil
}
