// 统一管理系统引擎和链运行上下文
package def

import (
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	envconf "github.com/xuperchain/xupercore/kernel/engines/config"
	engconf "github.com/xuperchain/xupercore/kernel/engines/xuperos/config"
)

// 引擎运行上下文环境
type EngineCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *envconf.EnvConf
	// 引擎配置
	EngCfg *engconf.EngineConf
	// 网络组件句柄
	Net XNetwork
}

// 链级别上下文，维护链级别上下文，每条平行链各有一个
type ChainCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 账本
	Ledger XLedger
	// 共识
	Consensus XConsensus
	// 合约
	Contract XContract
	// 权限
	Permission XPermission
	// 加密
	Crypto XCrypto
}
