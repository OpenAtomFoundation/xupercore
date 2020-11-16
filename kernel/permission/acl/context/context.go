package context

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
)

type LedgerCtx interface {
	//全局ctx Resource
	GetGenesisItem(item string) interface{}
	GetConfirmedAccountACL(accountName string) ([]byte, error)
	GetConfirmedMethodACL(contractName, methodName string) ([]byte, error)
}

type FakeContract interface {
	RegisterKernMethod(contract, method string, handle kernel.KernMethod)
}

type PermissionCtx struct {
	BcName   string
	BCtx     xcontext.BaseCtx
	Ledger   LedgerCtx
	Register FakeContract
}

func CreatePermissionCtx(bcName string, bCtx xcontext.BaseCtx, leger LedgerCtx, register FakeContract) PermissionCtx {
	return PermissionCtx{
		BcName:   bcName,
		BCtx:     bCtx,
		Ledger:   leger,
		Register: register,
	}
}
