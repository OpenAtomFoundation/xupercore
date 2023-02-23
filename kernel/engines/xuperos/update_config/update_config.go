package update_config

import (
	"encoding/json"
	"fmt"

	statctx "github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/meta"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
	"github.com/xuperchain/xupercore/protos"
)

const (
	updateGasPriceMethod = "updateGasPrice"
	getGasPriceMethod    = "getGasPrice"
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
	gasPriceMap, ok := args["gasprice"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("gasPriceMap err: %v", err)
	}
	gasPriceByte, err := json.Marshal(&gasPriceMap)
	if err != nil {
		return nil, fmt.Errorf("gasPriceByte err: %v", err)
	}
	err = json.Unmarshal(gasPriceByte, &nextGasPrice)
	if err != nil {
		return nil, fmt.Errorf("unmarshal ctxArgs err: %v", err)
	}
	k.Context.XLog.Debug("args: %s", nextGasPrice)
	// 调用方法
	legAgent := k.Context.ChainCtx.Ledger
	sctx, err := statctx.NewStateCtx(k.Context.ChainCtx.EngCtx.EnvCfg, k.BcName, legAgent, k.Context.ChainCtx.Crypto)
	if err != nil {
		return nil, fmt.Errorf("new state ctx err: %v", err)
	}
	stateDB := k.Context.ChainCtx.Ledger.GetBaseDB()
	batch := k.Context.ChainCtx.State.NewBatch()
	meta, err := meta.NewMeta(sctx, stateDB)
	if err != nil {
		return nil, fmt.Errorf("new meta err: %v", err)
	}
	err = meta.UpdateGasPrice(&nextGasPrice, batch)
	if err != nil {
		return nil, fmt.Errorf("update gas price err: %v", err)
	}
	gasp := meta.GetGasPrice()
	fmt.Printf("gasp: %s", gasp)
	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (k *KernMethod) getGasPrice(contractCtx contract.KContext) (*contract.Response, error) {
	return nil, nil
}
