package govern_token

import (
	"encoding/json"
	"fmt"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
	pb "github.com/xuperchain/xupercore/protos"
)

// Manager manages all gov releated data, providing read/write interface
type Manager struct {
	Ctx *GovCtx
}

// NewGovManager create instance of GovManager
func NewGovManager(ctx *GovCtx) (GovManager, error) {
	if ctx == nil || ctx.Ledger == nil || ctx.Contract == nil || ctx.BcName == "" {
		return nil, fmt.Errorf("acl ctx set error")
	}

	newGovGas, err := ctx.Ledger.GetNewGovGas()
	if err != nil {
		return nil, fmt.Errorf("get gov gas failed.err:%v", err)
	}

	predistribution, err := ctx.Ledger.GetGenesisPreDistribution()
	if err != nil {
		return nil, fmt.Errorf("get predistribution failed.err:%v", err)
	}

	t := NewKernContractMethod(ctx.BcName, newGovGas, predistribution)
	register := ctx.Contract.GetKernRegistry()
	register.RegisterKernMethod(utils.GovernTokenKernelContract, "Init", t.InitGovernTokens)
	register.RegisterKernMethod(utils.GovernTokenKernelContract, "Transfer", t.TransferGovernTokens)
	register.RegisterKernMethod(utils.GovernTokenKernelContract, "Lock", t.LockGovernTokens)
	register.RegisterKernMethod(utils.GovernTokenKernelContract, "UnLock", t.UnLockGovernTokens)
	register.RegisterKernMethod(utils.GovernTokenKernelContract, "Query", t.QueryAccountGovernTokens)
	register.RegisterKernMethod(utils.GovernTokenKernelContract, "TotalSupply", t.TotalSupply)

	mg := &Manager{
		Ctx: ctx,
	}

	return mg, nil
}

// GetGovTokenBalance get govern token balance of an account
func (mgr *Manager) GetGovTokenBalance(accountName string) (*pb.GovernTokenBalance, error) {
	accountBalanceBuf, err := mgr.GetObjectBySnapshot(utils.GetGovernTokenBucket(), []byte(utils.MakeAccountBalanceKey(accountName)))
	if err != nil {
		return nil, fmt.Errorf("query account balance failed.err:%v", err)
	}

	balance := utils.NewGovernTokenBalance()
	err = json.Unmarshal(accountBalanceBuf, balance)
	if err != nil {
		return nil, fmt.Errorf("no sender found")
	}

	balanceRes := &pb.GovernTokenBalance{
		TotalBalance: balance.TotalBalance.String(),
	}

	return balanceRes, nil
}

// DetermineGovTokenIfInitialized
func (mgr *Manager) DetermineGovTokenIfInitialized() (bool, error) {
	res, err := mgr.GetObjectBySnapshot(utils.GetGovernTokenBucket(), []byte(utils.GetDistributedKey()))
	if err != nil {
		return false, fmt.Errorf("query govern if initialized failed, err:%v", err)
	}

	if string(res) == "true" {
		return true, nil
	}

	return false, nil
}

func (mgr *Manager) GetObjectBySnapshot(bucket string, object []byte) ([]byte, error) {
	// 根据tip blockid 创建快照
	reader, err := mgr.Ctx.Ledger.GetTipXMSnapshotReader()
	if err != nil {
		return nil, err
	}

	return reader.Get(bucket, object)
}
