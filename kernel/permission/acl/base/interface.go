package base

import (
	pb "github.com/xuperchain/xupercore/protos"
)

type AclManager interface {
	GetAccountACL(accountName string) (*pb.Acl, error)
	GetContractMethodACL(contractName, methodName string) (*pb.Acl, error)
	GetAccountAddresses(accountName string) ([]string, error)
}
