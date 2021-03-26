package timer

import (
	"encoding/json"
	"fmt"

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
	taskIDBuf := args["task_id"]
	triggerBuf := args["trigger"]
	if blockHeightBuf == nil || taskIDBuf == nil || triggerBuf == nil {
		return nil, fmt.Errorf("add timer task failed, block_height, task_id or trigger is nil")
	}

	key := utils.MakeTimerBlockHeightTaskKey(string(blockHeightBuf), string(taskIDBuf))
	err := ctx.Put(utils.GetTimerBucket(), []byte(key), triggerBuf)
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
	_, err = ctx.Call("xkernel", trigger.Module, trigger.Method, timerTxArgs)
	if err != nil {
		return
	}

}
