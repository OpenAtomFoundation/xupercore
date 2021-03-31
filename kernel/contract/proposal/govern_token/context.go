package govern_token

import (
	"fmt"

	xledger "github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

type LedgerRely interface {
	// 从创世块获取创建合约账户消耗gas
	GetNewGovGas() (int64, error)
	// 从创世块获取创建合约账户消耗gas
	GetGenesisPreDistribution() ([]xledger.Predistribution, error)
	// 获取状态机最新确认快照
	GetTipXMSnapshotReader() (ledger.XMSnapshotReader, error)
}

type GovCtx struct {
	// 基础上下文
	xcontext.BaseCtx
	BcName   string
	Ledger   LedgerRely
	Contract contract.Manager
}

func NewGovCtx(bcName string, leg LedgerRely, contract contract.Manager) (*GovCtx, error) {
	if bcName == "" || leg == nil || contract == nil {
		return nil, fmt.Errorf("new gov ctx failed because param error")
	}

	log, err := logs.NewLogger("", utils.GovernTokenKernelContract)
	if err != nil {
		return nil, fmt.Errorf("new gov ctx failed because new logger error. err:%v", err)
	}

	ctx := new(GovCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = bcName
	ctx.Ledger = leg
	ctx.Contract = contract

	return ctx, nil
}
