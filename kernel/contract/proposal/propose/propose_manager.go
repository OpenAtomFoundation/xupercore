package propose

import (
	"encoding/json"
	"fmt"

	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
	pb "github.com/xuperchain/xupercore/protos"
)

// Manager manages all timer releated data, providing read/write interface
type Manager struct {
	Ctx *ProposeCtx
}

// NewTimerManager create instance of ProposeManager
func NewProposeManager(ctx *ProposeCtx) (ProposeManager, error) {
	if ctx == nil || ctx.Ledger == nil || ctx.Contract == nil || ctx.BcName == "" {
		return nil, fmt.Errorf("propose ctx set error")
	}

	t := NewKernContractMethod(ctx.BcName)
	register := ctx.Contract.GetKernRegistry()
	register.RegisterKernMethod(SubModName, "Propose", t.Propose)
	register.RegisterKernMethod(SubModName, "Vote", t.Vote)
	register.RegisterKernMethod(SubModName, "Thaw", t.Thaw)
	register.RegisterKernMethod(SubModName, "IsVoteOk", t.IsVoteOK)

	mg := &Manager{
		Ctx: ctx,
	}

	return mg, nil
}

// GetAccountACL get acl of an account
func (mgr *Manager) GetProposalByID(proposalID string) (*pb.Acl, error) {
	acl, err := mgr.GetObjectBySnapshot(utils.GetTimerBucket(), []byte(proposalID))
	if err != nil {
		return nil, fmt.Errorf("query timer tasks failed.err:%v", err)
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
