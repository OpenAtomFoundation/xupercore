package timer

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
)

type KernMethod struct {
	BcName string
}

func NewKernContractMethod(bcName string) *KernMethod {
	t := &KernMethod{
		BcName: bcName,
	}
	return t
}

func (t *KernMethod) Add(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	blockHeightBuf := args["block_height"]
	triggerBuf := args["trigger"]
	if blockHeightBuf == nil || triggerBuf == nil {
		return nil, fmt.Errorf("add timer task failed, block_height, task_id or trigger is nil")
	}

	taskID, err := t.getNextTaskID(ctx)
	if err != nil {
		return nil, fmt.Errorf("add timer task failed, get task_id err")
	}

	// 更新taskID
	err = ctx.Put(utils.GetTimerBucket(), utils.GetTaskIDKey(), []byte(taskID))
	if err != nil {
		return nil, err
	}

	key := utils.MakeTimerBlockHeightTaskKey(string(blockHeightBuf), taskID)
	err = ctx.Put(utils.GetTimerBucket(), []byte(key), triggerBuf)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: 0,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) Do(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	blockHeightBuf := args["block_height"]
	if blockHeightBuf == nil {
		return nil, fmt.Errorf("do timer tasks failed, blockHeightBuf is nil")
	}
	// 根据高度遍历所有提案，判断是否达到投票要求并执行
	startKey := utils.MakeTimerBlockHeightPrefix(string(blockHeightBuf))
	prefix := utils.MakeTimerBlockHeightPrefixSeparator(string(blockHeightBuf))
	endKey := utils.PrefixRange([]byte(prefix))
	iter, err := ctx.Select(utils.GetTimerBucket(), []byte(startKey), endKey)
	if err != nil {
		return nil, fmt.Errorf("do timer tasks failed, generate proposals iterator error")
	}
	defer iter.Close()
	for iter.Next() {
		// 触发交易
		triggerBuf := iter.Value()
		t.Trigger(ctx, triggerBuf)
	}

	delta := contract.Limits{
		XFee: 0,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) Trigger(ctx contract.KContext, triggerBuf []byte) {
	var trigger utils.TriggerDesc
	err := json.Unmarshal(triggerBuf, &trigger)
	if err != nil {
		return
	}
	timerTxArgs := make(map[string][]byte)
	triggerArgsBytes, err := json.Marshal(trigger.Args)
	if err != nil {
		return
	}
	timerTxArgs["args"] = triggerArgsBytes

	// 回调proposal trigger
	_, err = ctx.Call(trigger.Module, trigger.Contract, trigger.Method, timerTxArgs)
	if err != nil {
		return
	}

}

func (t *KernMethod) getNextTaskID(ctx contract.KContext) (string, error) {
	latestTaskID, err := ctx.Get(utils.GetTimerBucket(), utils.GetTaskIDKey())
	if err != nil {
		// 没找到，从1开始
		return big.NewInt(1).String(), nil
	} else {
		// 找到了，自增1
		taskID := big.NewInt(0)
		taskID.SetString(string(latestTaskID), 10)
		return taskID.Add(taskID, big.NewInt(1)).String(), nil
	}
}
