package xtoken

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

type Context struct {
	// 基础上下文
	xcontext.BaseCtx

	BcName string

	Contract contract.Manager
	ChainCtx *common.ChainCtx
}

func NewXTokenCtx(cctx *common.ChainCtx) (*Context, error) {
	if cctx == nil {
		return nil, fmt.Errorf("new parachain ctx failed because param error")
	}

	log, err := logs.NewLogger("", XTokenContract)
	if err != nil {
		return nil, fmt.Errorf("new parachain ctx failed because new logger error. err:%v", err)
	}

	ctx := new(Context)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = cctx.BCName
	ctx.Contract = cctx.Contract
	ctx.ChainCtx = cctx
	// cctx.Ledger.GenesisBlock.GetConfig().

	return ctx, nil
}
