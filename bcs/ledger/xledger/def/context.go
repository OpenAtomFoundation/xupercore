// 统一管理系统引擎和链运行上下文
package def

import (
	lconf "github.com/xuperchain/xupercore/bcs/ledger/xledger/config"
	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
)

// 引擎运行上下文环境
type LedgerCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *xconf.EnvConf
	// 账本配置
	LedgerCfg *lconf.XLedgerConf
}
