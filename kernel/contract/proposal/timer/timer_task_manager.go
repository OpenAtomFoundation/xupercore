package timer

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
	pb "github.com/xuperchain/xupercore/protos"
)

// Manager manages all timer releated data, providing read/write interface
type Manager struct {
	Ctx *TimerCtx
}

// NewTimerTaskManager create instance of TimerManager
func NewTimerTaskManager(ctx *TimerCtx) (TimerManager, error) {
	if ctx == nil || ctx.Ledger == nil || ctx.Contract == nil || ctx.BcName == "" {
		return nil, fmt.Errorf("timer ctx set error")
	}

	t := NewKernContractMethod(ctx.BcName)
	register := ctx.Contract.GetKernRegistry()
	register.RegisterKernMethod(SubModName, "Add", t.Add)
	register.RegisterKernMethod(SubModName, "Do", t.Do)

	mg := &Manager{
		Ctx: ctx,
	}

	return mg, nil
}

// GetTimerTasks get timer tasks
func (mgr *Manager) GetTimerTasks(blockHeight int64) (*pb.Acl, error) {
	blockHeightStr := strconv.FormatInt(blockHeight, 10)
	acl, err := mgr.GetObjectBySnapshot(utils.GetTimerBucket(), []byte(blockHeightStr))
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
