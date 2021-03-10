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
	proposalIDBuf := args["proposal_id"]
	triggerBuf := args["trigger"]
	if blockHeightBuf == nil || proposalIDBuf == nil || triggerBuf == nil {
		return nil, fmt.Errorf("add timer task failed, block_height, proposal_id or trigger is nil")
	}

	key := utils.MakeTimerBlockHeightTaskKey(string(blockHeightBuf), string(proposalIDBuf))
	err := ctx.Put(utils.GetTimerBucket(), []byte(key), triggerBuf)
	if err != nil {
		return nil, err
	}

	fmt.Println("timerAndTrigger,", args["trigger"])

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

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil

	// 根据高度遍历所有提案，判断是否达到投票要求并执行
	prefix := utils.MakeTimerBlockHeightPrefix(string(blockHeightBuf))
	startKey, endKey := utils.PrefixRange([]byte(prefix))
	iter, err := ctx.Select(utils.GetTimerBucket(), startKey, endKey)
	defer iter.Close()
	if err != nil {
		return nil, fmt.Errorf("do timer tasks failed, generate proposals iterator error")
	}
	for iter.Next() {
		// 判断投票是否够，调用proposal合约
		proposalID := string(iter.Key())[len(prefix):]
		queryProposalArgs := make(map[string][]byte)
		queryProposalArgs["proposal_id"] = []byte(proposalID)
		_, err = ctx.Call("xkernel", "$proposal", "IsVoteOK", queryProposalArgs)
		if err != nil {
			continue
		}

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
	switch trigger.Module {
	case "$consensus":
		// 跨合约调用，进行共识升级
		updateConsensusArgs := make(map[string][]byte)
		triggerArgsBytes, err := json.Marshal(trigger.Args)
		if err != nil {
			return
		}
		updateConsensusArgs["args"] = triggerArgsBytes
		_, err = ctx.Call("xkernel", "$consensus", "updateConsensus", updateConsensusArgs)
		if err != nil {
			return
		}
	default:
		return
	}
}
