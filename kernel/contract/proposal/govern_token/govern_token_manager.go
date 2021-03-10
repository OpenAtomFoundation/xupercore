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

	t := NewKernContractMethod(ctx.BcName, newGovGas)
	register := ctx.Contract.GetKernRegistry()
	register.RegisterKernMethod(SubModName, "Init", t.InitGovernTokens)
	register.RegisterKernMethod(SubModName, "Transfer", t.TransferGovernTokens)
	register.RegisterKernMethod(SubModName, "Lock", t.LockGovernTokens)
	register.RegisterKernMethod(SubModName, "UnLock", t.UnLockGovernTokens)
	register.RegisterKernMethod(SubModName, "Rebase", t.RebaseGovernTokens)

	mg := &Manager{
		Ctx: ctx,
	}

	return mg, nil
}

// GetAccountACL get acl of an account
func (mgr *Manager) GetGovTokenBalance(accountName string) (*pb.Acl, error) {
	acl, err := mgr.GetObjectBySnapshot(utils.GetBalanceBucket(), []byte(accountName))
	if err != nil {
		return nil, fmt.Errorf("query account balance failed.err:%v", err)
	}

	aclBuf := &pb.Acl{}
	err = json.Unmarshal(acl, aclBuf)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal acl failed.acl:%s,err:%v", string(acl), err)
	}
	return aclBuf, nil
}

func (mgr *Manager) GetObjectBySnapshot(bucket string, object []byte) ([]byte, error) {
	// 根据tip blockid 创建快照
	reader, err := mgr.Ctx.Ledger.GetTipXMSnapshotReader()
	if err != nil {
		return nil, err
	}

	return reader.Get(bucket, object)
}
