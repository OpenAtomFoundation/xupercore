package acl

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/permission/acl/bucket"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
)

type KernMethod struct {
	BcName                   string
	NewAccountResourceAmount int64
}

func NewKernContractMethod(bcName string, NewAccountResourceAmount int64) *KernMethod {
	t := &KernMethod{
		BcName:                   bcName,
		NewAccountResourceAmount: NewAccountResourceAmount,
	}
	return t
}

func (t *KernMethod) NewAccount(ctx contract.KContext) (*contract.Response, error) {

	// check fee
	if ctx.ResourceLimit().XFee < t.NewAccountResourceAmount {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewAccountResourceAmount)
	}

	// check ACK
	args := ctx.Args()
	aclJSON := args["acl"]
	acl, err := newACL(aclJSON)
	if err != nil {
		return nil, err
	}

	// check account name
	accountNumber := string(args["account_name"])
	if validErr := utils.ValidAccountNumber(accountNumber); validErr != nil {
		return nil, validErr
	}

	bcName := t.BcName
	if bcName == "" {
		return nil, fmt.Errorf("block name is empty")
	}
	accountName := utils.MakeAccountKey(bcName, accountNumber)

	// create account
	accounts := bucket.AccountBucket{DB: ctx}
	exist, err := accounts.IsExist(accountName)
	if err != nil {
		return nil, err
	}
	if exist {
		return nil, fmt.Errorf("account already exists: %s", accountName)
	}
	err = accounts.SetAccountACL(accountName, aclJSON)
	if err != nil {
		return nil, err
	}

	// add ak -> account reflection
	ak2Account := &bucket.AK2AccountBucket{DB: ctx}
	if err := ak2Account.UpdateForAccount(accountName, nil, acl.mustGetAKs()); err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: t.NewAccountResourceAmount,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    aclJSON,
	}, nil
}

func (t *KernMethod) SetAccountACL(ctx contract.KContext) (*contract.Response, error) {

	// check fee
	if ctx.ResourceLimit().XFee < t.NewAccountResourceAmount/1000 {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewAccountResourceAmount/1000)
	}

	// check new ACL
	args := ctx.Args()
	aclJSON := args["acl"]
	aclNew, err := newACL(aclJSON)
	if err != nil {
		return nil, err
	}

	// get old ACL
	accountName := string(args["account_name"])
	accounts := bucket.AccountBucket{DB: ctx}
	data, err := accounts.GetAccountACL(accountName)
	if err != nil {
		return nil, err
	}
	aclOld, err := newACL(data)
	if err != nil {
		return nil, fmt.Errorf("parse old ACL fail: %s", err)
	}

	// update ak -> account reflection
	ak2Account := &bucket.AK2AccountBucket{DB: ctx}
	err = ak2Account.UpdateForAccount(accountName, aclOld.mustGetAKs(), aclNew.mustGetAKs())
	if err != nil {
		return nil, err
	}

	// update account ACL
	err = accounts.SetAccountACL(accountName, aclJSON)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: t.NewAccountResourceAmount / 1000,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    aclJSON,
	}, nil
}

func (t *KernMethod) SetMethodACL(ctx contract.KContext) (*contract.Response, error) {

	// check fee
	if ctx.ResourceLimit().XFee < t.NewAccountResourceAmount/1000 {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewAccountResourceAmount/1000)
	}

	// check args
	args := ctx.Args()
	contractName := string(args["contract_name"])
	methodName := string(args["method_name"])
	if contractName == "" || methodName == "" {
		return nil, fmt.Errorf("set method acl failed, contract name is nil or method name is nil")
	}
	aclJSON := args["acl"]
	_, err := newACL(aclJSON)
	if err != nil {
		return nil, err
	}

	// update contract method ACL
	contracts := bucket.ContractBucket{DB: ctx}
	err = contracts.SetMethodACL(contractName, methodName, aclJSON)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: t.NewAccountResourceAmount / 1000,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    aclJSON,
	}, nil
}
