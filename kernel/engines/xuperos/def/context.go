// 统一管理系统引擎和链运行上下文
package def

import (
	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract"
	engconf "github.com/xuperchain/xupercore/kernel/engines/xuperos/config"
	"github.com/xuperchain/xupercore/kernel/network"
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
	Net network.Network
}

// 链级别上下文，维护链级别上下文，每条平行链各有一个
type ChainCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 链名
	BCName string
	// 网络
	Net network.Network
	// 账本
	Ledger XLedger
	// 共识
	Consensus XConsensus
	// 合约
	Contract contract.Manager
	// 权限
	Acl XAcl
	// 加密
	Crypto XCrypto
	// 结点账户信息
	AddrInfo *NodeAddrInfo
}
