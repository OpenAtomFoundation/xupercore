package ledger

import (
	"fmt"

	lconf "github.com/OpenAtomFoundation/xupercore/global/bcs/ledger/xledger/config"
	"github.com/OpenAtomFoundation/xupercore/global/bcs/ledger/xledger/def"
	xconf "github.com/OpenAtomFoundation/xupercore/global/kernel/common/xconfig"
	xctx "github.com/OpenAtomFoundation/xupercore/global/kernel/common/xcontext"
	"github.com/OpenAtomFoundation/xupercore/global/lib/logs"
	"github.com/OpenAtomFoundation/xupercore/global/lib/timer"
)

// 账本运行上下文环境
type LedgerCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *xconf.EnvConf
	// 账本配置
	LedgerCfg *lconf.XLedgerConf
	// 链名
	BCName string
}

func NewLedgerCtx(envCfg *xconf.EnvConf, bcName string) (*LedgerCtx, error) {
	if envCfg == nil {
		return nil, fmt.Errorf("create ledger context failed because env conf is nil")
	}

	// 加载配置
	lcfg, err := lconf.LoadLedgerConf(envCfg.GenConfFilePath(envCfg.LedgerConf))
	if err != nil {
		return nil, fmt.Errorf("create ledger context failed because load config error.err:%v", err)
	}

	log, err := logs.NewLogger("", def.LedgerSubModName)
	if err != nil {
		return nil, fmt.Errorf("create ledger context failed because new logger error. err:%v", err)
	}

	ctx := new(LedgerCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.EnvCfg = envCfg
	ctx.LedgerCfg = lcfg
	ctx.BCName = bcName

	return ctx, nil
}
