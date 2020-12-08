package context

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/ledger"
)

type LedgerRely interface {
	// 从创世块获取创建合约账户消耗gas
	GetNewAccountGas() (int64, error)
	// 获取状态机最新确认快照
	GetTipXMSnapshotReader() (ledger.XMSnapshotReader, error)
}

type ContractRely interface {
	RegisterKernMethod(contract, method string, handle kernel.KernMethod)
}

type AclCtx struct {
	// 基础上下文
	xcontext.BaseCtx
	BcName   string
	Ledger   LedgerRely
	Register ContractRely
}
