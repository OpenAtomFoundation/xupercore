package context

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
)

type LedgerCtx interface {
	HasTransaction(txId []byte) (bool, error)
}

type FakeContract interface {
	RegisterKernMethod(contract, method string, handle kernel.KernMethod)
}

type XModelCtx interface {
	Get(bucket string, key []byte) ([]byte, error)
}

type PermissionCtx struct {
	BcName   string
	BCtx     xcontext.BaseCtx
	Ledger   LedgerCtx
	Register FakeContract
	XModel   XModelCtx
}

func CreatePermissionCtx(bcName string, bCtx xcontext.BaseCtx, leger LedgerCtx, register FakeContract) PermissionCtx {
	return PermissionCtx{
		BcName:   bcName,
		BCtx:     bCtx,
		Ledger:   leger,
		Register: register,
	}
}
