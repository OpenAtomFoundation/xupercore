// 统一管理系统引擎和链运行上下文
package commom

import (
	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	engconf "github.com/xuperchain/xupercore/kernel/engines/xuperos/config"
)

// 引擎运行上下文环境
type EngineCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *xconf.EnvConf
	// 引擎配置
	EngCfg *engconf.EngineConf
	// 网络组件句柄
	Net XNetwork
}

// 链级别上下文，维护链级别上下文，每条平行链各有一个
type ChainCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 引擎上下文
	EngCtx *EngineCtx
	// 链名
	BCName string
	// 账本
	Ledger XLedger
	// 状态机
	State XState
	// 合约
	Contract XContract
	// 共识
	Consensus XConsensus
	// 加密
	Crypto XCrypto
	// 权限
	Acl XAcl
	// 结点账户信息
	Address *Address
}
