package evm

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/engine"
	"github.com/hyperledger/burrow/execution/errors"
	"github.com/hyperledger/burrow/execution/evm"
	"github.com/hyperledger/burrow/execution/evm/abi"
	"github.com/hyperledger/burrow/execution/exec"

	xabi "github.com/xuperchain/xupercore/bcs/contract/evm/abi"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
	xchainpb "github.com/xuperchain/xupercore/protos"
)

const (
	initializeMethod    = "initialize"
	evmParamJSONEncoded = "jsonEncoded"
	evmInput            = "input"
)

type evmCreator struct {
	vm *evm.EVM
}

func newEvmCreator(_ *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	opt := evm.Options{}
	vm := evm.New(opt)
	return &evmCreator{
		vm: vm,
	}, nil
}

// CreateInstance instances an evm virtual machine instance which can run a single contract call
func (e *evmCreator) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	state := newStateManager(ctx)
	blockState := newBlockStateManager(ctx)
	return &evmInstance{
		vm:         e.vm,
		ctx:        ctx,
		state:      state,
		blockState: blockState,
		cp:         cp,
		fromCache:  ctx.ReadFromCache,
	}, nil
}

func (e *evmCreator) RemoveCache(_ string) {
}

type evmInstance struct {
	vm         *evm.EVM
	ctx        *bridge.Context
	state      *stateManager
	blockState *blockStateManager
	cp         bridge.ContractCodeProvider
	code       []byte
	abi        []byte
	gasUsed    uint64
	fromCache  bool
}

func (i *evmInstance) Exec() error {
	var err error

	// 获取合约的 code。
	if i.fromCache {
		i.code, err = i.cp.GetContractCodeFromCache(i.ctx.ContractName)
		if err != nil {
			return err
		}
	} else {
		i.code, err = i.cp.GetContractCode(i.ctx.ContractName)
		if err != nil {
			return err
		}
	}

	// 部署合约或者调用合约时参数未使用 abi 编码时需要获取到合约的 abi。执行结果也需要使用 abi 解析。
	if i.fromCache {
		i.abi, err = i.cp.GetContractAbiFromCache(i.ctx.ContractName)
		if err != nil {
			return err
		}
	} else {
		i.abi, err = i.cp.GetContractAbi(i.ctx.ContractName)
		if err != nil {
			return err
		}
	}

	if i.ctx.Method == initializeMethod {
		return i.deployContract()
	}

	var caller crypto.Address
	if IsContractAccount(i.state.ctx.Initiator) {
		caller, err = ContractAccountToEVMAddress(i.state.ctx.Initiator)
	} else {
		caller, err = XchainToEVMAddress(i.state.ctx.Initiator)
	}
	if err != nil {
		return err
	}

	callee, err := ContractNameToEVMAddress(i.ctx.ContractName)
	if err != nil {
		return err
	}

	gas := uint64(contract.MaxLimits.Cpu)

	// 如果客户端已经将参数进行了 abi 编码，那么此处不需要再进行编码，而且返回的结果也不需要 abi 解码。否则此处需要将参数 abi 编码同时将结果 abi 解码。
	needDecodeResp := false
	input := []byte{}
	jsonEncoded, ok := i.ctx.Args[evmParamJSONEncoded]
	if !ok || string(jsonEncoded) != "true" {
		input = i.ctx.Args[evmInput]
	} else {
		needDecodeResp = true
		if input, err = i.encodeInvokeInput(); err != nil {
			return err
		}
	}

	value := big.NewInt(0)
	ok = false
	if i.ctx.TransferAmount != "" {
		value, ok = new(big.Int).SetString(i.ctx.TransferAmount, 0)
		if !ok {
			return fmt.Errorf("get evm value error")
		}
	}
	params := engine.CallParams{
		CallType: exec.CallTypeCode,
		Caller:   caller,
		Callee:   callee,
		Input:    input,
		Value:    value,
		Gas:      &gas,
	}
	out, err := i.vm.Execute(i.state, i.blockState, i, params, i.code)
	if err != nil {
		return err
	}

	if needDecodeResp {
		// 执行结果根据 abi 解码，返回 json 格式的数组。
		out, err = decodeRespWithAbiForEVM(string(i.abi), i.ctx.Method, out)
		if err != nil {
			return err
		}
	}

	i.gasUsed = uint64(contract.MaxLimits.Cpu) - *params.Gas

	i.ctx.Output = &pb.Response{
		Status: 200,
		Body:   out,
	}
	return nil
}

func (i *evmInstance) ResourceUsed() contract.Limits {
	return contract.Limits{
		Cpu: int64(i.gasUsed),
	}
}

func (i *evmInstance) Release() {
}

func (i *evmInstance) Abort(msg string) {
}

func (i *evmInstance) Call(call *exec.CallEvent, exception *errors.Exception) error {
	return nil
}

func (i *evmInstance) Log(log *exec.LogEvent) error {
	contractName, _, err := DetermineEVMAddress(log.Address)
	if err != nil {
		return err
	}

	contractAbiByte, err := i.cp.GetContractAbi(contractName)
	if err != nil {
		return err
	}
	event, err := unpackEventFromAbi(contractAbiByte, contractName, log)
	if err != nil {
		return err
	}
	i.ctx.Events = append(i.ctx.Events, event)
	i.ctx.State.AddEvent(event)
	return nil
}

func unpackEventFromAbi(abiByte []byte, contractName string, log *exec.LogEvent) (*xchainpb.ContractEvent, error) {
	var eventID abi.EventID
	copy(eventID[:], log.GetTopic(0).Bytes())
	spec, err := abi.ReadSpec(abiByte)
	if err != nil {
		return nil, err
	}
	eventSpec, ok := spec.EventsByID[eventID]
	if !ok {
		return nil, fmt.Errorf("The Event By ID Not Found ")
	}
	vals := abi.GetPackingTypes(eventSpec.Inputs)
	if err := abi.UnpackEvent(eventSpec, log.Topics, log.Data, vals...); err != nil {
		return nil, err
	}
	event := &xchainpb.ContractEvent{
		Contract: contractName,
	}
	var uint8type = reflect.TypeOf((*[]uint8)(nil))
	event.Name = eventSpec.Name
	for i := 0; i < len(vals); i++ {
		t := reflect.TypeOf(vals[i])
		if t == uint8type {
			s := fmt.Sprintf("%x", vals[i])
			vals[i] = s[1:]
		}
	}
	data, err := json.Marshal(vals)
	if err != nil {
		return nil, err
	}
	event.Body = data
	return event, nil
}

func (i *evmInstance) deployContract() error {
	var caller crypto.Address
	var err error
	if IsContractAccount(i.state.ctx.Initiator) {
		caller, err = ContractAccountToEVMAddress(i.state.ctx.Initiator)
	} else {
		caller, err = XchainToEVMAddress(i.state.ctx.Initiator)
	}
	if err != nil {
		return err
	}

	callee, err := ContractNameToEVMAddress(i.ctx.ContractName)
	if err != nil {
		return err
	}

	gas := uint64(contract.MaxLimits.Cpu)

	input := []byte{}
	jsonEncoded, ok := i.ctx.Args[evmParamJSONEncoded]
	if !ok || string(jsonEncoded) != "true" {
		// 客户端传来的参数是已经 abi 编码的。
		input = i.code
	} else {
		// 客户端未将参数编码。
		if input, err = i.encodeDeployInput(); err != nil {
			return err
		}
	}

	params := engine.CallParams{
		CallType: exec.CallTypeCode,
		Origin:   caller,
		Caller:   caller,
		Callee:   callee,
		Input:    input,
		Value:    big.NewInt(0),
		Gas:      &gas,
	}
	contractCode, err := i.vm.Execute(i.state, i.blockState, i, params, input)
	if err != nil {
		return err
	}

	key := evmCodeKey(i.ctx.ContractName)
	err = i.ctx.State.Put("contract", key, contractCode)
	if err != nil {
		return err
	}

	i.gasUsed = uint64(contract.MaxLimits.Cpu) - *params.Gas

	i.ctx.Output = &pb.Response{
		Status: 200,
	}
	return nil
}

func evmCodeKey(contractName string) []byte {
	return []byte(contractName + "." + "code")
}

func evmAbiKey(contractName string) []byte {
	return []byte(contractName + "." + "abi")
}

func init() {
	bridge.Register(bridge.TypeEvm, "evm", newEvmCreator)
}

// func encodeArgsWithAbiForEVM(abiData []byte, funcName string, args []byte) ([]byte, error) {
// 	packedBytes, _, err := abi.EncodeFunctionCall(string(abiData), funcName, nil, args)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return packedBytes, nil
// }

func decodeRespWithAbiForEVM(abiData, funcName string, resp []byte) ([]byte, error) {
	Variables, err := abi.DecodeFunctionReturn(abiData, funcName, resp)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]string, 0, len(Variables))
	for _, v := range Variables {
		result = append(result, map[string]string{
			v.Name: v.Value,
		})
	}

	out, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (i *evmInstance) encodeDeployInput() ([]byte, error) {
	// 客户端如果未将参数进行 abi 编码，那么通过 input 获取的是参数 json 序列化的结果。
	argsBytes, ok := i.ctx.Args[evmInput]
	if !ok {
		return nil, fmt.Errorf("missing emvInput")
	}

	// map 的类型与客户端一致，如果 cli 或者 SDK 对此结构有改动，需要同时修改。
	args := make(map[string]interface{})
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		return nil, err
	}

	enc, err := xabi.New(i.abi)
	if err != nil {
		return nil, err
	}

	input, err := enc.Encode("", args)
	if err != nil {
		return nil, err
	}

	evmCode := string(i.code) + hex.EncodeToString(input)
	codeBuf, err := hex.DecodeString(evmCode)
	if err != nil {
		return nil, err
	}

	return codeBuf, nil
}

func (i *evmInstance) encodeInvokeInput() ([]byte, error) {
	argsBytes, ok := i.ctx.Args[evmInput]
	if !ok {
		return nil, nil
	}

	args := make(map[string]interface{})
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		return nil, err
	}

	enc, err := xabi.New(i.abi)
	if err != nil {
		return nil, err
	}

	input, err := enc.Encode(i.ctx.Method, args)
	if err != nil {
		return nil, err
	}

	return input, nil
}
