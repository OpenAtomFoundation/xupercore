package context

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

const (
	SubModName = "acl"
)

type LedgerRely interface {
	// 从创世块获取创建合约账户消耗gas
	GetNewAccountGas() (int64, error)
	// 获取状态机最新确认快照
	GetTipXMSnapshotReader() (ledger.XMSnapshotReader, error)
}

type AclCtx struct {
	// 基础上下文
	xcontext.BaseCtx
	BcName   string
	Ledger   LedgerRely
	Contract contract.Manager
}

func NewAclCtx(bcName string, leg LedgerRely, contract contract.Manager) (*AclCtx, error) {
	if bcName == "" || leg == nil || contract == nil {
		return nil, fmt.Errorf("new acl ctx failed because param error")
	}

	log, err := logs.NewLogger("", SubModName)
	if err != nil {
		return nil, fmt.Errorf("new acl ctx failed because new logger error. err:%v", err)
	}

	ctx := new(AclCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = bcName
	ctx.Ledger = leg
	ctx.Contract = contract

	return ctx, nil
}
