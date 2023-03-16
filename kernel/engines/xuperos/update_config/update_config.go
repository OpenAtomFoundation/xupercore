package update_config

import (
	"encoding/json"
	"fmt"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
	"github.com/xuperchain/xupercore/protos"
)

const (
	updateGasPriceMethod = "updateGasPrice"
	updateMaxBlockSize   = "updateMaxBlockSize"
)

type KernMethod struct {
	BcName  string
	Context *UpdateConfigCtx
}

func NewKernMethod(ctx *UpdateConfigCtx) *KernMethod {
	t := &KernMethod{
		BcName:  ctx.BcName,
		Context: ctx,
	}
	return t
}

func (k *KernMethod) updateGasPrice(contractCtx contract.KContext) (*contract.Response, error) {
	ctxArgs := contractCtx.Args()
	args := make(map[string]interface{})
	err := json.Unmarshal(ctxArgs["args"], &args)
	if err != nil {
		return nil, fmt.Errorf("unmarshal ctxArgs err: %v", err)
	}
	var nextGasPrice protos.GasPrice
	gasPriceMap, ok := args["gas_price"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("gasPriceMap err, gasPriceMap %v", args["gasprice"])
	}
	gasPriceByte, err := json.Marshal(&gasPriceMap)
	if err != nil {
		return nil, fmt.Errorf("gasPriceByte err: %v", err)
	}
	err = json.Unmarshal(gasPriceByte, &nextGasPrice)
	if err != nil {
		return nil, fmt.Errorf("unmarshal gasPriceByte err: %v", err)
	}
	k.Context.XLog.Debug("args: %s", nextGasPrice)
	// 调用方法
	batch := k.Context.ChainCtx.State.NewBatch()
	err = k.Context.ChainCtx.State.UpdateGasPrice(k.Context.OldGasPrice, &nextGasPrice, batch)
	if err != nil {
		return nil, fmt.Errorf("update gas price err: %v", err)
	}
	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (k *KernMethod) updateMaxBlockSize(contractCtx contract.KContext) (*contract.Response, error) {
	ctxArgs := contractCtx.Args()
	args := make(map[string]int64)
	err := json.Unmarshal(ctxArgs["args"], &args)
	if err != nil {
		return nil, fmt.Errorf("unmarshal ctxArgs err: %v", err)
	}
	batch := k.Context.ChainCtx.State.NewBatch()
	err = k.Context.ChainCtx.State.UpdateMaxBlockSize(k.Context.OldMaxBlockSize, args["maxBlockSize"], batch)
	if err != nil {
		return nil, fmt.Errorf("update max block size err: %v", err)
	}
	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}
