// 面向接口编程
package def

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/engines"
	"github.com/xuperchain/xupercore/kernel/network"
)

type Chain interface {
	SetRelyAgent(ChainRelyAgent) error
	Start()
	Stop()
	ProcTx()
	ProcBlocks()
	PreExec()
	GetChainCtx() *ChainCtx
}

// 定义该引擎对各组件依赖接口约束
type ChainRelyAgent interface {
	CreateContractManager() (contract.Manager, error)
}

// 定义xuperos引擎对外暴露接口
// 依赖接口而不是依赖具体实现
type Engine interface {
	engines.BCEngine
	SetRelyAgent(EngineRelyAgent) error
	Get(string) Chain
	Set(string, Chain)
	GetChains() []string
	GetEngineCtx() *EngineCtx
	CreateChain(string, []byte) (Chain, error)
	RegisterChain(string) error
	UnloadChain(string) error
}

// 定义该引擎对各组件依赖接口约束
type EngineRelyAgent interface {
	CreateNetwork() (network.Network, error)
	CreateLedger() (XLedger, error)
}

// 定义引擎对账本组件依赖接口约束
type XLedger interface {
}

// 定义引擎对共识组件依赖接口约束
type XConsensus interface {
}

// 定引擎义对权限组件依赖接口约束
type XAcl interface {
}

// 定义引擎对加密组件依赖接口约束
type XCrypto interface {
}
