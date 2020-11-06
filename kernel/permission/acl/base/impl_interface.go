package base

import (
	pctx "github.com/xuperchain/xupercore/kernel/permission/acl/context"
	"github.com/xuperchain/xupercore/kernel/permission/acl/pb"
)

type PermissionImpl interface {
	GetAccountACL(ctx pctx.PermissionCtx, accountName string) (*pb.Acl, error)
	GetAccountACLWithConfirmed(ctx pctx.PermissionCtx, accountName string) (*pb.Acl, bool, error)
	GetContractMethodACL(ctx pctx.PermissionCtx, contractName, methodName string) (*pb.Acl, error)
	GetContractMethodACLWithConfirmed(ctx pctx.PermissionCtx, contractName, methodName string) (*pb.Acl, bool, error)
}
