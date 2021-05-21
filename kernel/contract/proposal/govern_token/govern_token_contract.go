package govern_token

import (
	"encoding/json"
	"fmt"
	"math/big"

	xledger "github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
)

type KernMethod struct {
	BcName               string
	NewGovResourceAmount int64
	Predistribution      []xledger.Predistribution
}

func NewKernContractMethod(bcName string, NewGovResourceAmount int64, Predistribution []xledger.Predistribution) *KernMethod {
	t := &KernMethod{
		BcName:               bcName,
		NewGovResourceAmount: NewGovResourceAmount,
		Predistribution:      Predistribution,
	}
	return t
}

func (t *KernMethod) InitGovernTokens(ctx contract.KContext) (*contract.Response, error) {
	// 判断是否已经初始化
	res, err := ctx.Get(utils.GetGovernTokenBucket(), []byte(utils.GetDistributedKey()))
	if err == nil && string(res) == "true" {
		return &contract.Response{
			Status:  utils.StatusOK,
			Message: "success",
			Body:    []byte("Govern tokens has been initialized"),
		}, fmt.Errorf("Govern tokens has been initialized.")
	}

	totalSupply := big.NewInt(0)
	for _, ps := range t.Predistribution {
		amount := big.NewInt(0)
		amount.SetString(ps.Quota, 10)
		if amount.Cmp(big.NewInt(0)) < 0 {
			return nil, fmt.Errorf("init gov tokens failed, parse genesis account error, negative amount")
		}

		balance := utils.NewGovernTokenBalance()
		balance.TotalBalance = amount

		balanceBuf, err := json.Marshal(balance)
		if err != nil {
			return nil, err
		}

		// 设置初始账户的govern token余额
		key := utils.MakeAccountBalanceKey(ps.Address)
		err = ctx.Put(utils.GetGovernTokenBucket(), []byte(key), balanceBuf)
		if err != nil {
			return nil, err
		}

		// 更新余额
		totalSupply = totalSupply.Add(totalSupply, amount)
	}

	// 保存总额
	key := utils.MakeTotalSupplyKey()
	err = ctx.Put(utils.GetGovernTokenBucket(), []byte(key), []byte(totalSupply.String()))
	if err != nil {
		return nil, err
	}

	// 设置已经初始化的标志
	err = ctx.Put(utils.GetGovernTokenBucket(), []byte(utils.GetDistributedKey()), []byte("true"))
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
	sender := ctx.Initiator()
	receiverBuf := args["to"]
	amountBuf := args["amount"]
	if receiverBuf == nil || amountBuf == nil {
		return nil, fmt.Errorf("transfer gov tokens failed, to is nil or amount is nil")
	}

	amount := big.NewInt(0)
	_, isAmount := amount.SetString(string(amountBuf), 10)
	if !isAmount || amount.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("transfer gov tokens failed, parse amount error")
	}

	// 查询sender余额
	senderBalance, err := t.balanceOf(ctx, sender)
	if err != nil {
		return nil, fmt.Errorf("transfer gov tokens failed, query sender balance error")
	}

	// 比较并更新sender余额
	for _, v := range senderBalance.LockedBalance {
		tmpAmount := big.NewInt(0)
		tmpAmount.Sub(senderBalance.TotalBalance, v)
		if tmpAmount.Cmp(amount) < 0 {
			return nil, fmt.Errorf("transfer gov tokens failed, sender's insufficient balance")
		}
	}
	senderBalance.TotalBalance.Sub(senderBalance.TotalBalance, amount)

	// 设置receiver余额
	receiverBalance := utils.NewGovernTokenBalance()
	receiverBalance.TotalBalance.Set(amount)

	// 查询receiver余额并更新
	receiverKey := utils.MakeAccountBalanceKey(string(receiverBuf))
	receiverBalanceBuf, err := ctx.Get(utils.GetGovernTokenBucket(), []byte(receiverKey))
	if err == nil {
		receiverBalanceOld := &utils.GovernTokenBalance{}
		json.Unmarshal(receiverBalanceBuf, receiverBalanceOld)
		receiverBalance.TotalBalance.Add(receiverBalance.TotalBalance, receiverBalanceOld.TotalBalance)
	}

	// 更新sender余额
	senderBalanceBuf, _ := json.Marshal(senderBalance)
	senderKey := utils.MakeAccountBalanceKey(sender)
	err = ctx.Put(utils.GetGovernTokenBucket(), []byte(senderKey), senderBalanceBuf)
	if err != nil {
		return nil, fmt.Errorf("transfer gov tokens failed, update sender's balance")
	}

	// 更新receiver余额
	receiverBalanceBuf, _ = json.Marshal(receiverBalance)
	err = ctx.Put(utils.GetGovernTokenBucket(), []byte(receiverKey), receiverBalanceBuf)
	if err != nil {
		return nil, fmt.Errorf("transfer gov tokens failed, update receriver's balance")
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

func (t *KernMethod) LockGovernTokens(ctx contract.KContext) (*contract.Response, error) {
	// 调用权限校验
	if ctx.Caller() != utils.ProposalKernelContract && ctx.Caller() != utils.TDPOSKernelContract && ctx.Caller() != utils.XPOSKernelContract {
		return nil, fmt.Errorf("caller %s no authority to LockGovernTokens", ctx.Caller())
	}
	args := ctx.Args()
	accountBuf := args["from"]
	amountBuf := args["amount"]
	lockTypeBuf := args["lock_type"]
	if accountBuf == nil || amountBuf == nil || lockTypeBuf == nil {
		return nil, fmt.Errorf("lock gov tokens failed, account, amount or lock_type is nil")
	}

	// 校验场景
	lockType := string(lockTypeBuf)
	if lockType != utils.GovernTokenTypeOrdinary && lockType != utils.GovernTokenTypeTDPOS {
		return nil, fmt.Errorf("lock gov tokens failed, lock_type invalid: %s", lockType)
	}

	// 查询account余额
	accountBalance, err := t.balanceOf(ctx, string(accountBuf))
	if err != nil {
		return nil, fmt.Errorf("lock gov tokens failed, query account balance error")
	}
	amountLock := big.NewInt(0)
	amountLock.SetString(string(amountBuf), 10)
	// 比较account available balance amount
	availableBalance := big.NewInt(0)
	availableBalance.Sub(accountBalance.TotalBalance, accountBalance.LockedBalance[lockType])
	if availableBalance.Cmp(amountLock) == -1 {
		return nil, fmt.Errorf("lock gov tokens failed, account available balance insufficient")
	}

	accountBalance.LockedBalance[lockType].Add(accountBalance.LockedBalance[lockType], amountLock)

	// 更新account余额
	accountBalanceBuf, _ := json.Marshal(accountBalance)
	accountKey := utils.MakeAccountBalanceKey(string(accountBuf))
	err = ctx.Put(utils.GetGovernTokenBucket(), []byte(accountKey), accountBalanceBuf)
	if err != nil {
		return nil, fmt.Errorf("transfer gov tokens failed, update sender's balance")
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

func (t *KernMethod) UnLockGovernTokens(ctx contract.KContext) (*contract.Response, error) {
	// 调用权限校验
	if ctx.Caller() != utils.ProposalKernelContract && ctx.Caller() != utils.TDPOSKernelContract && ctx.Caller() != utils.XPOSKernelContract {
		return nil, fmt.Errorf("caller %s no authority to UnLockGovernTokens", ctx.Caller())
	}
	args := ctx.Args()
	accountBuf := args["from"]
	amountBuf := args["amount"]
	lockTypeBuf := args["lock_type"]
	if accountBuf == nil || amountBuf == nil || lockTypeBuf == nil {
		return nil, fmt.Errorf("lock gov tokens failed, account, amount or lock_type is nil")
	}

	// 查询account余额
	accountBalance, err := t.balanceOf(ctx, string(accountBuf))
	if err != nil {
		return nil, fmt.Errorf("unlock gov tokens failed, query account balance error")
	}
	amountLock := big.NewInt(0)
	amountLock.SetString(string(amountBuf), 10)
	// 解锁account balance amount
	lockType := string(lockTypeBuf)
	if lockType != utils.GovernTokenTypeOrdinary && lockType != utils.GovernTokenTypeTDPOS {
		return nil, fmt.Errorf("unlock gov tokens failed, lock_type invalid: %s", lockType)
	}
	accountBalance.LockedBalance[lockType] = accountBalance.LockedBalance[lockType].Sub(accountBalance.LockedBalance[lockType], amountLock)

	// 更新account余额
	accountBalanceBuf, _ := json.Marshal(accountBalance)
	accountKey := utils.MakeAccountBalanceKey(string(accountBuf))
	err = ctx.Put(utils.GetGovernTokenBucket(), []byte(accountKey), accountBalanceBuf)
	if err != nil {
		return nil, fmt.Errorf("transfer gov tokens failed, update sender's balance")
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

func (t *KernMethod) QueryAccountGovernTokens(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	accountBuf := args["account"]

	// 查询account余额
	balance, err := t.balanceOf(ctx, string(accountBuf))
	if err != nil {
		return nil, fmt.Errorf("query account gov tokens balance failed, query account balance error:%s", err.Error())
	}

	balanceResBuf, err := json.Marshal(balance)
	if err != nil {
		return nil, fmt.Errorf("query account gov tokens balance failed, error:%s", err.Error())
	}

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    balanceResBuf,
	}, nil
}

func (t *KernMethod) TotalSupply(ctx contract.KContext) (*contract.Response, error) {
	key := utils.MakeTotalSupplyKey()
	totalSupplyBuf, err := ctx.Get(utils.GetGovernTokenBucket(), []byte(key))
	if err != nil {
		return nil, err
	}

	totalSupply := big.NewInt(0)
	totalSupply.SetString(string(totalSupplyBuf), 10)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    []byte(totalSupply.String()),
	}, nil

}

func (t *KernMethod) balanceOf(ctx contract.KContext, account string) (*utils.GovernTokenBalance, error) {
	accountKey := utils.MakeAccountBalanceKey(account)
	accountBalanceBuf, err := ctx.Get(utils.GetGovernTokenBucket(), []byte(accountKey))
	if err != nil {
		return utils.NewGovernTokenBalance(), fmt.Errorf("no sender found")
	}
	balance := utils.NewGovernTokenBalance()
	err = json.Unmarshal(accountBalanceBuf, balance)
	if err != nil {
		return utils.NewGovernTokenBalance(), fmt.Errorf("no sender found")
	}

	return balance, nil
}
