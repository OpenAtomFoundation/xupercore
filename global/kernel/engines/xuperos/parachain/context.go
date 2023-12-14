package parachain

import (
	"fmt"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/common/xcontext"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"
	"github.com/OpenAtomFoundation/xupercore/global/lib/logs"
	"github.com/OpenAtomFoundation/xupercore/global/lib/timer"
)

const (
	ParaChainKernelContract = "$parachain"
)

const (
	ParaChainStatusStart = 0
	ParaChainStatusStop  = 1
)

// Deprecated
// use Ctx instead
type ParaChainCtx = Ctx

type Ctx struct {
	// 基础上下文
	xcontext.BaseCtx
	BcName   string
	Contract contract.Manager
	ChainCtx *common.ChainCtx
}

func NewParaChainCtx(bcName string, cCtx *common.ChainCtx) (*Ctx, error) {
	if bcName == "" || cCtx == nil {
		return nil, fmt.Errorf("new parachain ctx failed because param error")
	}

	log, err := logs.NewLogger("", ParaChainKernelContract)
	if err != nil {
		return nil, fmt.Errorf("new parachain ctx failed because new logger error. err:%v", err)
	}

	ctx := new(Ctx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = bcName
	ctx.Contract = cCtx.Contract
	ctx.ChainCtx = cCtx

	return ctx, nil
}
