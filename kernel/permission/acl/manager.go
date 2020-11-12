package acl

import (
	"encoding/json"
	"errors"

	"github.com/xuperchain/xupercore/kernel/permission/acl/base"
	pctx "github.com/xuperchain/xupercore/kernel/permission/acl/context"
	"github.com/xuperchain/xupercore/kernel/permission/acl/pb"
)

const (
	StatusOK = 200
)

// Manager manages all ACL releated data, providing read/write interface for ACL table
type Manager struct {
	Ctx pctx.PermissionCtx
}

// NewACLManager create instance of ACLManager
func NewACLManager(ctx pctx.PermissionCtx) (base.PermissionImpl, error) {
	newAkResourceAmount, ok := ctx.Ledger.GetGenesisItem("NewAccountResourceAmount").(int64)
	if !ok {
		return nil, errors.New("get NewAccountResourceAmount fails")
	}
	t := NewKernContractMethod(ctx.BcName, newAkResourceAmount)
	ctx.Register.RegisterKernMethod("acl", "NewAccount", t.NewAccount)
	ctx.Register.RegisterKernMethod("acl", "SetAccountACL", t.SetAccountACL)
	ctx.Register.RegisterKernMethod("acl", "SetMethodACL", t.SetMethodACL)
	return &Manager{
		Ctx: ctx,
	}, nil
}

// GetAccountACL get acl of an account
func (mgr *Manager) GetAccountACL(ctx pctx.PermissionCtx, accountName string) (*pb.Acl, error) {
	//todo 提供新的方法 从ledger查询最后一笔已确认交易
	acl, err := ctx.Ledger.GetConfirmedAccountACL(accountName)
	if err != nil {
		return nil, err
	}
	aclBuf := &pb.Acl{}
	json.Unmarshal(acl, aclBuf)
	return aclBuf, nil
}

// GetContractMethodACL get acl of contract method
func (mgr *Manager) GetContractMethodACL(ctx pctx.PermissionCtx, contractName, methodName string) (*pb.Acl, error) {
	acl, err := ctx.Ledger.GetConfirmedMethodACL(contractName, methodName)
	if err != nil {
		return nil, err
	}
	aclBuf := &pb.Acl{}
	json.Unmarshal(acl, aclBuf)
	return aclBuf, nil
}

// GetAccountAddresses get the addresses belongs to contract account
func (mgr *Manager) GetAccountAddresses(ctx pctx.PermissionCtx, accountName string) ([]string, error) {
	acl, err := mgr.GetAccountACL(ctx, accountName)
	if err != nil {
		return nil, err
	}

	return mgr.getAddressesByACL(acl)
}

func (mgr *Manager) getAddressesByACL(acl *pb.Acl) ([]string, error) {
	addresses := make([]string, 0)

	switch acl.GetPm().GetRule() {
	case pb.PermissionRule_SIGN_THRESHOLD:
		for ak := range acl.GetAksWeight() {
			addresses = append(addresses, ak)
		}
	case pb.PermissionRule_SIGN_AKSET:
		for _, set := range acl.GetAkSets().GetSets() {
			aks := set.GetAks()
			addresses = append(addresses, aks...)
		}
	default:
		return nil, errors.New("Unknown permission rule")
	}

	return addresses, nil
}
