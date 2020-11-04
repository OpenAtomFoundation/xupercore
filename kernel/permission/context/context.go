package context

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
)

type LedgerCtx interface {
	HasTransaction(txId []byte) (bool, error)
}

type PermissionCtx struct {
	BcName string
	BCtx   xcontext.BaseCtx
	Ledger LedgerCtx
	KCtx   kernel.KContext
}

func CreatePermissionCtx(bcName string, bCtx xcontext.BaseCtx, leger LedgerCtx, kCtx kernel.KContext) PermissionCtx {
	return PermissionCtx{
		BcName: bcName,
		BCtx:   bCtx,
		Ledger: leger,
		KCtx:   kCtx,
	}
}
