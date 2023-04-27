package chain_config

import (
	"encoding/json"
	"testing"
)

func TestUpdateGasPrice(t *testing.T) {

	args := make(map[string]interface{})

	gasprice := map[string]interface{}{
		"cpu_rate":  1,
		"mem_rate":  1,
		"disk_rate": 2,
		"xfee_rate": 2,
	}
	args["gas_price"] = gasprice
	argsBytes, err := json.Marshal(&args)
	if err != nil {
		t.Fatal("Failed to marshal gasprice map to JSON:", err)
		return
	}
	ctxArgs := make(map[string][]byte)
	ctxArgs["args"] = argsBytes
	ctx := NewFakeKContext(ctxArgs, map[string]map[string][]byte{})
	cc := &ChainConfigCtx{}
	mgr := NewKernMethod(cc)
	_, err = mgr.updateGasPrice(ctx)
	if err != nil {
		t.Fatal("update gasprice failed", err)
		return
	}
}

func TestUpdateMaxBlockSize(t *testing.T) {
	args := make(map[string]interface{})
	maxBlockSize := 12222222
	args["maxBlockSize"] = maxBlockSize

	argsBytes, err := json.Marshal(&args)
	if err != nil {
		t.Fatal("json marshal error", err)
	}
	ctxArgs := make(map[string][]byte)
	ctxArgs["args"] = argsBytes
	ctx := NewFakeKContext(ctxArgs, map[string]map[string][]byte{})

	cc := &ChainConfigCtx{}
	mgr := NewKernMethod(cc)
	_, err = mgr.updateMaxBlockSize(ctx)
	if err != nil {
		t.Fatal("update max block size failed", err)
		return
	}
}
