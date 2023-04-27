package chain_config

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

type ChainConfigCtx struct {
	xcontext.BaseCtx
	BcName          string
	Contract        contract.Manager
	ChainCtx        *common.ChainCtx
	OldGasPrice     *protos.GasPrice
	OldMaxBlockSize int64
}

func NewChainConfigCtx(chainCtx *common.ChainCtx) (*ChainConfigCtx, error) {
	if chainCtx.BCName == "" || chainCtx.Contract == nil {
		return nil, NewChainConfigCtxErr
	}

	log, err := logs.NewLogger("", utils.ChainConfigKernelContract)
	if err != nil {
		return nil, fmt.Errorf("new updateConfig ctx faild because new logger error. err: %v", err)
	}
	meta := chainCtx.State.GetMeta()
	ctx := new(ChainConfigCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = chainCtx.BCName
	ctx.Contract = chainCtx.Contract
	ctx.ChainCtx = chainCtx
	ctx.OldGasPrice = meta.GetGasPrice()
	ctx.OldMaxBlockSize = meta.GetMaxBlockSize()
	return ctx, nil
}
