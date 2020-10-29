// 面向接口编程
package def

import (
	"github.com/xuperchain/xupercore/kernel/engines"
)

type Chain interface {
	Init()
	Start()
	Stop()
	ProcessTx()
	ProcessBlock()
	PreExec()
	GetChainCtx() *ChainCtx
}

// 定义xuperos引擎对外暴露接口
// 依赖接口而不是依赖具体实现
type Engine interface {
	engines.BCEngine
	Get(string) Chain
	Set(string, Chain)
	GetChains() []string
	GetEngineCtx() *EngineCtx
	CreateChain(string, []byte) (Chain, error)
	RegisterChain(string) error
	UnloadChain(string) error
}

// 定义该引擎对各组件依赖接口约束

// 定义引擎对网络组件依赖接口约束
type XNetwork interface {
}

// 定义引擎对账本组件依赖接口约束
type XLedger interface {
}

// 定义引擎对共识组件依赖接口约束
type XConsensus interface {
}

// 定义引擎对合约组件依赖接口约束
type XContract interface {
}

// 定引擎义对权限组件依赖接口约束
type XPermission interface {
}

// 定义引擎对加密组件依赖接口约束
type XCrypto interface {
}
