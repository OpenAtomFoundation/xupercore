package base

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
	pctx "github.com/xuperchain/xupercore/kernel/permission/context"
	"github.com/xuperchain/xupercore/kernel/permission/pb"
)

type PermissionImpl interface {
	GetAccountACL(ctx pctx.PermissionCtx, accountName string) (*pb.Acl, error)
	GetAccountACLWithConfirmed(ctx pctx.PermissionCtx, accountName string) (*pb.Acl, bool, error)
	GetContractMethodACL(ctx pctx.PermissionCtx, contractName, methodName string) (*pb.Acl, error)
	GetContractMethodACLWithConfirmed(ctx pctx.PermissionCtx, contractName, methodName string) (*pb.Acl, bool, error)
	NewAccount(ctx kernel.KContext) (*contract.Response, error)
	SetAccountACL(ctx kernel.KContext) (*contract.Response, error)
	SetMethodACL(ctx kernel.KContext) (*contract.Response, error)
}
