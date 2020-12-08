package base

import (
	actx "github.com/xuperchain/xupercore/kernel/permission/acl/context"
	pb "github.com/xuperchain/xupercore/protos"
)

type AclManager interface {
	GetAccountACL(accountName string) (*pb.Acl, error)
	GetContractMethodACL(contractName, methodName string) (*pb.Acl, error)
}
