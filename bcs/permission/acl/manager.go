package acl

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xuperchain/xupercore/bcs/permission/acl/utils"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/permission"
	"github.com/xuperchain/xupercore/kernel/permission/base"
	pctx "github.com/xuperchain/xupercore/kernel/permission/context"
	"github.com/xuperchain/xupercore/kernel/permission/pb"
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
	return &Manager{
		Ctx: ctx,
	}
}

func init() {
	permission.Register("acl", NewACLManager)
	t := &Manager{}
	kernel.RegisterKernMethod("$acl", "NewAccount", t.NewAccount)
	kernel.RegisterKernMethod("$acl", "SetAccountACL", t.SetAccountACL)
	kernel.RegisterKernMethod("$acl", "SetMethodACL", t.SetMethodACL)
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
	data, err := ctx.KCtx.GetObject(utils.GetAccountBucket(), []byte(accountName))
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
	data, err := ctx.KCtx.GetObject(utils.GetContractBucket(), []byte(key))
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

func (mgr *Manager) NewAccount(ctx kernel.KContext) (*contract.Response, error) {
	args := ctx.Args()
	// json -> pb.Acl
	accountName := args["account_name"]
	aclJSON := args["acl"]
	aclBuf := &pb.Acl{}
	json.Unmarshal(aclJSON, aclBuf)

	if accountName == nil {
		return nil, fmt.Errorf("Invoke NewAccount failed, warn: account name is empty")
	}
	accountStr := string(accountName)
	if validErr := utils.ValidRawAccount(accountStr); validErr != nil {
		return nil, validErr
	}

	bcname := mgr.Ctx.BcName
	if bcname == "" {
		return nil, fmt.Errorf("block name is empty")
	}
	accountStr = utils.MakeAccountKey(bcname, accountStr)

	if validErr := validACL(aclBuf); validErr != nil {
		return nil, validErr
	}

	oldAccount, err := ctx.GetObject(utils.GetAccountBucket(), []byte(accountStr))
	if err != nil {
		return nil, err
	}
	if oldAccount != nil {
		return nil, fmt.Errorf("account already exists: %s", accountName)
	}
	err = ctx.PutObject(utils.GetAccountBucket(), []byte(accountStr), aclJSON)
	if err != nil {
		return nil, err
	}

	// add ak -> account reflection
	err = UpdateAK2AccountReflection(ctx, nil, aclJSON, accountStr)
	if err != nil {
		return nil, err
	}

	//todo ctx.AddResourceUsed(ctx.NewAccountResourceAmount)

	return &contract.Response{
		Status:  StatusOK,
		Message: "success",
		Body:    aclJSON,
	}, nil
}

func (mgr *Manager) SetAccountACL(ctx kernel.KContext) (*contract.Response, error) {
	args := ctx.Args()
	// json -> pb.Acl
	accountName := args["account_name"]
	aclJSON := args["acl"]
	aclBuf := &pb.Acl{}
	json.Unmarshal(aclJSON, aclBuf)
	if validErr := validACL(aclBuf); validErr != nil {
		return nil, validErr
	}

	data, err := ctx.GetObject(utils.GetAccountBucket(), accountName)
	if err != nil {
		return nil, err
	}
	// delete ak -> account reflection
	// add ak -> account reflection
	aclOldJSON := data
	err = UpdateAK2AccountReflection(ctx, aclOldJSON, aclJSON, string(accountName))
	if err != nil {
		return nil, err
	}

	err = ctx.PutObject(utils.GetAccountBucket(), accountName, aclJSON)
	if err != nil {
		return nil, err
	}

	//todo ctx.AddResourceUsed(ctx.NewAccountResourceAmount / 1000)

	return &contract.Response{
		Status:  StatusOK,
		Message: "success",
		Body:    aclJSON,
	}, nil
}

func (mgr *Manager) SetMethodACL(ctx kernel.KContext) (*contract.Response, error) {
	args := ctx.Args()
	contractNameBuf := args["contract_name"]
	methodNameBuf := args["method_name"]
	if contractNameBuf == nil || methodNameBuf == nil {
		return nil, fmt.Errorf("set method acl failed, contract name is nil or method name is nil")
	}

	// json -> pb.Acl
	contractName := string(contractNameBuf)
	methodName := string(methodNameBuf)
	aclJSON := args["acl"]
	aclBuf := &pb.Acl{}
	json.Unmarshal(aclJSON, aclBuf)

	if validErr := validACL(aclBuf); validErr != nil {
		return nil, validErr
	}
	key := utils.MakeContractMethodKey(contractName, methodName)
	err := ctx.PutObject(utils.GetContractBucket(), []byte(key), aclJSON)
	if err != nil {
		return nil, err
	}

	//todo ctx.AddXFeeUsed(ctx.NewAccountResourceAmount / 1000)
	return &contract.Response{
		Status:  StatusOK,
		Message: "success",
		Body:    aclJSON,
	}, nil
}

func validACL(acl *pb.Acl) error {
	// param absence check
	if acl == nil {
		return fmt.Errorf("valid acl failed, arg of acl is nil")
	}

	// permission model check
	if permissionModel := acl.GetPm(); permissionModel != nil {
		permissionRule := permissionModel.GetRule()
		akSets := acl.GetAkSets()
		aksWeight := acl.GetAksWeight()
		if akSets == nil && aksWeight == nil {
			return fmt.Errorf("invoke NewAccount failed, permission model is not valid")
		}
		// aks limitation check
		if permissionRule == pb.PermissionRule_SIGN_THRESHOLD {
			if aksWeight == nil || len(aksWeight) > utils.GetAkLimit() {
				return fmt.Errorf("valid acl failed, aksWeight is nil or size of aksWeight is very big")
			}
		} else if permissionRule == pb.PermissionRule_SIGN_AKSET {
			if akSets != nil {
				sets := akSets.GetSets()
				if sets == nil || len(sets) > utils.GetAkLimit() {
					return fmt.Errorf("valid acl failed, Sets is nil or size of Sets is very big")
				}
			} else {
				return fmt.Errorf("valid acl failed, akSets is nil")
			}
		} else {
			return fmt.Errorf("valid acl failed, permission model is not found")
		}
	} else {
		return fmt.Errorf("valid acl failed, lack of argument of permission model")
	}

	return nil
}
