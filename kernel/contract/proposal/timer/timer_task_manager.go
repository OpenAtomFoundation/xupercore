package timer

import (
	"fmt"
	"strconv"

	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
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
	register.RegisterKernMethod(utils.TimerTaskKernelContract, "Add", t.Add)
	register.RegisterKernMethod(utils.TimerTaskKernelContract, "Do", t.Do)

	mg := &Manager{
		Ctx: ctx,
	}

	return mg, nil
}

// GetTimerTasks get timer tasks
func (mgr *Manager) GetTimerTasks(blockHeight int64) (uint64, error) {
	blockHeightStr := strconv.FormatInt(blockHeight, 10)
	_, err := mgr.GetObjectBySnapshot(utils.GetTimerBucket(), []byte(blockHeightStr))
	if err != nil {
		return 0, fmt.Errorf("query timer tasks failed.err:%v", err)
	}

	return 0, nil
}

func (mgr *Manager) GetObjectBySnapshot(bucket string, object []byte) ([]byte, error) {
	// 根据tip blockid 创建快照
	reader, err := mgr.Ctx.Ledger.GetTipXMSnapshotReader()
	if err != nil {
		return nil, err
	}

	return reader.Get(bucket, object)
}
