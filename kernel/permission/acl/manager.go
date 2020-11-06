package acl

import (
	"encoding/json"
	"errors"

	"github.com/xuperchain/xupercore/kernel/permission/acl/base"
	pctx "github.com/xuperchain/xupercore/kernel/permission/acl/context"
	"github.com/xuperchain/xupercore/kernel/permission/acl/pb"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
)

const (
	StatusOK = 200
)

// Manager manages all ACL releated data, providing read/write interface for ACL table
type Manager struct {
	Ctx pctx.PermissionCtx
}

// NewACLManager create instance of ACLManager
func NewACLManager(ctx pctx.PermissionCtx) base.PermissionImpl {
	t := NewKernContractMethod(ctx.BcName)
	ctx.Register.RegisterKernMethod("acl", "NewAccount", t.NewAccount)
	ctx.Register.RegisterKernMethod("acl", "SetAccountACL", t.SetAccountACL)
	ctx.Register.RegisterKernMethod("acl", "SetMethodACL", t.SetMethodACL)
	return &Manager{
		Ctx: ctx,
	}
}

// GetAccountACL get acl of an account
func (mgr *Manager) GetAccountACL(ctx pctx.PermissionCtx, accountName string) (*pb.Acl, error) {
	acl, confirmed, err := mgr.GetAccountACLWithConfirmed(ctx, accountName)
	if err != nil {
		return nil, err
	}
	if acl != nil && !confirmed {
		return nil, errors.New("acl in unconfirmed")
	}
	return acl, nil
}

// GetContractMethodACL get acl of contract method
func (mgr *Manager) GetContractMethodACL(ctx pctx.PermissionCtx, contractName, methodName string) (*pb.Acl, error) {
	acl, confirmed, err := mgr.GetContractMethodACLWithConfirmed(ctx, contractName, methodName)
	if err != nil {
		return nil, err
	}
	if acl != nil && !confirmed {
		return nil, errors.New("acl in unconfirmed")
	}
	return acl, nil
}

// GetAccountACLWithConfirmed implements reading ACL of an account with confirmed state
func (mgr *Manager) GetAccountACLWithConfirmed(ctx pctx.PermissionCtx, accountName string) (*pb.Acl, bool, error) {
	data, err := ctx.XModel.Get(utils.GetAccountBucket(), []byte(accountName))
	if err != nil || data == nil {
		return nil, false, err
	}
	exists, err := ctx.Ledger.HasTransaction(data)
	if err != nil {
		return nil, false, err
	}

	// 反序列化
	acl := &pb.Acl{}
	json.Unmarshal(data, acl)

	return acl, exists, nil
}

// GetContractMethodACLWithConfirmed implements reading ACL of a contract method with confirmed state
func (mgr *Manager) GetContractMethodACLWithConfirmed(ctx pctx.PermissionCtx, contractName, methodName string) (*pb.Acl, bool, error) {
	key := utils.MakeContractMethodKey(contractName, methodName)
	data, err := ctx.XModel.Get(utils.GetContractBucket(), []byte(key))
	if err != nil || data == nil {
		return nil, false, err
	}
	exists, err := ctx.Ledger.HasTransaction(data)
	if err != nil {
		return nil, false, err
	}

	// 反序列化
	acl := &pb.Acl{}
	json.Unmarshal(data, acl)

	return acl, exists, nil
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
