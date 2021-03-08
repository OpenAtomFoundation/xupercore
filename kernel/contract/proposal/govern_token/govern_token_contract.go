package govern_token

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
)

type KernMethod struct {
	BcName               string
	NewGovResourceAmount int64
}

func NewKernContractMethod(bcName string, NewGovResourceAmount int64) *KernMethod {
	t := &KernMethod{
		BcName:               bcName,
		NewGovResourceAmount: NewGovResourceAmount,
	}
	return t
}

func (t *KernMethod) InitGovernTokens(ctx contract.KContext) (*contract.Response, error) {
	if ctx.ResourceLimit().XFee < t.NewGovResourceAmount {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewGovResourceAmount)
	}

	initiator := ctx.Initiator()
	fmt.Println(initiator)

	args := ctx.Args()
	totalSupplyBuf := args["total_supply"]
	if totalSupplyBuf == nil {
		return nil, fmt.Errorf("init gov tokens failed, total_supply is nil or amount is nil")
	}

	key := utils.GetTotalSupplyBucket()
	err := ctx.Put(utils.MakeTotalSupplyKey(), []byte(key), totalSupplyBuf)
	if err != nil {
		return nil, err
	}

	key = utils.MakeAccountBalanceKey(initiator)
	err = ctx.Put(utils.GetBalanceBucket(), []byte(key), totalSupplyBuf)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: t.NewGovResourceAmount,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) TransferGovernTokens(ctx contract.KContext) (*contract.Response, error) {
	if ctx.ResourceLimit().XFee < t.NewGovResourceAmount/1000 {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewGovResourceAmount/1000)
	}
	args := ctx.Args()
	receiverBuf := args["receiver"]
	amountBuf := args["amount"]
	if receiverBuf == nil || amountBuf == nil {
		return nil, fmt.Errorf("transfer gov tokens failed, receiver is nil or amount is nil")
	}

	delta := contract.Limits{
		XFee: t.NewGovResourceAmount / 1000,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) RebaseGovernTokens(ctx contract.KContext) (*contract.Response, error) {
	if ctx.ResourceLimit().XFee < t.NewGovResourceAmount/1000 {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewGovResourceAmount/1000)
	}
	//args := ctx.Args()
	//receiverBuf := args["receiver"]
	//amountBuf := args["amount"]
	//if receiverBuf == nil || amountBuf == nil {
	//	return nil, fmt.Errorf("transfer gov tokens failed, receiver is nil or amount is nil")
	//}

	delta := contract.Limits{
		XFee: t.NewGovResourceAmount / 1000,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) LockGovernTokens(ctx contract.KContext) (*contract.Response, error) {
	if ctx.ResourceLimit().XFee < t.NewGovResourceAmount/1000 {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewGovResourceAmount/1000)
	}

	initiator := ctx.Initiator()
	fmt.Println(initiator)

	args := ctx.Args()
	amountBuf := args["amount"]
	lockHeightBuf := args["lock_height"]
	if amountBuf == nil || lockHeightBuf == nil {
		return nil, fmt.Errorf("lock gov tokens failed, amount is nil or lock_height name is nil")
	}

	//amount := string(amountBuf)
	//lockHeight := string(lockHeightBuf)

	delta := contract.Limits{
		XFee: t.NewGovResourceAmount / 1000,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) UnLockGovernTokens(ctx contract.KContext) (*contract.Response, error) {
	if ctx.ResourceLimit().XFee < t.NewGovResourceAmount/1000 {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewGovResourceAmount/1000)
	}

	initiator := ctx.Initiator()
	fmt.Println(initiator)
	args := ctx.Args()
	amountBuf := args["amount"]
	if amountBuf == nil {
		return nil, fmt.Errorf("unlock gov tokens failed, amount is nil")
	}

	//amount := string(amountBuf)

	delta := contract.Limits{
		XFee: t.NewGovResourceAmount / 1000,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}
