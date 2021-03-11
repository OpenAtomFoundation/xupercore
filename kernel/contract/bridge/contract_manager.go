package bridge

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xuperchain/crypto/core/hash"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/protos"

	"github.com/golang/protobuf/proto"
)

type contractManager struct {
	xbridge      *XBridge
	codeProvider ContractCodeProvider
}

// DeployContract deploy contract and initialize contract
func (c *contractManager) DeployContract(kctx contract.KContext) (*contract.Response, contract.Limits, error) {
	args := kctx.Args()
	state := kctx
	name := args["contract_name"]
	if name == nil {
		return nil, contract.Limits{}, errors.New("bad contract name")
	}
	contractName := string(name)
	_, err := c.codeProvider.GetContractCodeDesc(contractName)
	if err == nil {
		return nil, contract.Limits{}, fmt.Errorf("contract %s already exists", contractName)
	}

	code := args["contract_code"]
	if code == nil {
		return nil, contract.Limits{}, errors.New("missing contract code")
	}
	initArgsBuf := args["init_args"]
	if initArgsBuf == nil {
		return nil, contract.Limits{}, errors.New("missing args field in args")
	}
	var initArgs map[string][]byte
	err = json.Unmarshal(initArgsBuf, &initArgs)
	if err != nil {
		return nil, contract.Limits{}, err
	}

	descbuf := args["contract_desc"]
	var desc protos.WasmCodeDesc
	err = proto.Unmarshal(descbuf, &desc)
	if err != nil {
		return nil, contract.Limits{}, err
	}
	desc.Digest = hash.DoubleSha256(code)
	descbuf, _ = proto.Marshal(&desc)

	state.Put("contract", ContractCodeDescKey(contractName), descbuf)
	state.Put("contract", contractCodeKey(contractName), code)

	if desc.ContractType == string(TypeEvm) {
		abiBuf := args["contract_abi"]
		state.Put("contract", contractAbiKey(contractName), abiBuf)
	}

	contractType, err := getContractType(&desc)
	if err != nil {
		return nil, contract.Limits{}, err
	}
	creator := c.xbridge.getCreator(contractType)
	if creator == nil {
		return nil, contract.Limits{}, fmt.Errorf("contract type %s not found", contractType)
	}
	cp := newCodeProvider(state)
	instance, err := creator.CreateInstance(&Context{
		State:          state,
		ContractName:   contractName,
		Method:         "initialize",
		ResourceLimits: kctx.ResourceLimit(),
	}, cp)
	if err != nil {
		creator.RemoveCache(contractName)
		// log.Error("create contract instance error when deploy contract", "error", err, "contract", contractName)
		return nil, contract.Limits{}, err
	}
	instance.Release()

	initConfig := contract.ContextConfig{
		ResourceLimits:        kctx.ResourceLimit(),
		State:                 kctx,
		Initiator:             kctx.Initiator(),
		AuthRequire:           kctx.AuthRequire(),
		ContractName:          contractName,
		CanInitialize:         true,
		ContractCodeFromCache: true,
	}
	initConfig.ContractName = contractName
	initConfig.CanInitialize = true
	initConfig.ContractCodeFromCache = true
	initConfig.State = kctx
	out, resourceUsed, err := c.initContract(contractType, &initConfig, initArgs)
	if err != nil {
		if _, ok := err.(*ContractError); !ok {
			creator.RemoveCache(contractName)
		}
		// log.Error("call contract initialize method error", "error", err, "contract", contractName)
		return nil, contract.Limits{}, err
	}
	return out, resourceUsed, nil
}

func (v *contractManager) initContract(tp ContractType, contextConfig *contract.ContextConfig, args map[string][]byte) (*contract.Response, contract.Limits, error) {
	ctx, err := v.xbridge.NewContext(contextConfig)
	if err != nil {
		return nil, contract.Limits{}, err
	}
	out, err := ctx.Invoke("initialize", args)
	if err != nil {
		return nil, contract.Limits{}, err
	}
	return out, ctx.ResourceUsed(), nil
}

// UpgradeContract deploy contract and initialize contract
func (c *contractManager) UpgradeContract(kctx contract.KContext) (*contract.Response, contract.Limits, error) {
	args := kctx.Args()
	if !c.xbridge.config.EnableUpgrade {
		return nil, contract.Limits{}, errors.New("contract upgrade disabled")
	}

	name := args["contract_name"]
	if name == nil {
		return nil, contract.Limits{}, errors.New("bad contract name")
	}
	contractName := string(name)
	desc, err := c.codeProvider.GetContractCodeDesc(contractName)
	if err != nil {
		return nil, contract.Limits{}, fmt.Errorf("contract %s not exists", contractName)
	}

	code := args["contract_code"]
	if code == nil {
		return nil, contract.Limits{}, errors.New("missing contract code")
	}
	desc.Digest = hash.DoubleSha256(code)
	descbuf, _ := proto.Marshal(desc)

	store := kctx
	store.Put("contract", ContractCodeDescKey(contractName), descbuf)
	store.Put("contract", contractCodeKey(contractName), code)

	cp := newCodeProvider(store)

	contractType, err := getContractType(desc)
	if err != nil {
		return nil, contract.Limits{}, err
	}
	creator := c.xbridge.getCreator(contractType)
	if creator == nil {
		return nil, contract.Limits{}, fmt.Errorf("contract type %s not found", contractType)
	}
	instance, err := creator.CreateInstance(&Context{
		ContractName:   contractName,
		ResourceLimits: contract.MaxLimits,
	}, cp)
	if err != nil {
		// log.Error("create contract instance error when upgrade contract", "error", err, "contract", contractName)
		return nil, contract.Limits{}, err
	}
	instance.Release()

	return &contract.Response{
			Status: 200,
			Body:   []byte("upgrade success"),
		}, contract.Limits{
			Disk: modelCacheDiskUsed(store),
		}, nil
}

func modelCacheDiskUsed(store contract.KContext) int64 {
	size := int64(0)
	wset := store.RWSet().WSet
	for _, w := range wset {
		size += int64(len(w.GetKey()))
		size += int64(len(w.GetValue()))
	}
	return size
}

func ContractCodeDescKey(contractName string) []byte {
	return []byte(contractName + "." + "desc")
}

func contractCodeKey(contractName string) []byte {
	return []byte(contractName + "." + "code")
}

func contractAbiKey(contractName string) []byte {
	return []byte(contractName + "." + "abi")
}

func getContractType(desc *protos.WasmCodeDesc) (ContractType, error) {
	switch desc.ContractType {
	case "", "wasm":
		return TypeWasm, nil
	case "native":
		return TypeNative, nil
	case "evm":
		return TypeEvm, nil
	case "xkernel":
		return TypeKernel, nil
	default:
		return "", fmt.Errorf("unknown contract type:%s", desc.ContractType)
	}
}
