package update_config

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/protos"
)

type LedgerRely interface {
	GetTipXMSnapshotReader() (ledger.XMSnapshotReader, error)
}

type UpdateConfigCtx struct {
	xcontext.BaseCtx
	BcName          string
	Contract        contract.Manager
	ChainCtx        *common.ChainCtx
	OldGasPrice     *protos.GasPrice
	OldMaxBlockSize int64
}

func NewUpdateConfigCtx(chainCtx *common.ChainCtx) (*UpdateConfigCtx, error) {
	if chainCtx.BCName == "" || chainCtx.Contract == nil {
		return nil, NewUpdateConfigCtxErr
	}

	log, err := logs.NewLogger("", utils.UpdateConfigKernelContract)
	if err != nil {
		return nil, fmt.Errorf("new updateConfig ctx faild because new logger error. err: %v", err)
	}
	meta := chainCtx.State.GetMeta()
	ctx := new(UpdateConfigCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = chainCtx.BCName
	ctx.Contract = chainCtx.Contract
	ctx.ChainCtx = chainCtx
	ctx.OldGasPrice = meta.GetGasPrice()
	ctx.OldMaxBlockSize = meta.GetMaxBlockSize()
	return ctx, nil
}
