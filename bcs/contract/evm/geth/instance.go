package geth

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/hyperledger/burrow/execution/errors"
	"github.com/hyperledger/burrow/execution/evm/abi"
	"github.com/hyperledger/burrow/execution/exec"

	xabi "github.com/xuperchain/xupercore/bcs/contract/evm/burrow/abi"
	gaddr "github.com/xuperchain/xupercore/bcs/contract/evm/geth/address"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
)

const (
	initializeMethod    = "initialize"
	evmParamJSONEncoded = "jsonEncoded"
)

type evmInstance struct {
	vm        *vm.EVM
	ctx       *bridge.Context
	cp        bridge.ContractCodeProvider
	code      []byte
	abi       []byte
	gasUsed   uint64
	fromCache bool
}

func (i *evmInstance) Exec() error {

	var (
		caller         gaddr.EVMAddress
		gas            = uint64(contract.MaxLimits.Cpu)
		value          = big.NewInt(0)
		input          []byte
		err            error
		needDecodeResp = false
	)

	// get caller
	if gaddr.IsContractAccount(i.ctx.Initiator) {
		caller, err = gaddr.NewEVMAddressFromContractAccount(i.ctx.Initiator)
	} else {
		caller, err = gaddr.NewEVMAddressFromXchainAK(i.ctx.Initiator)
	}
	if err != nil {
		return err
	}

	// set contract
	if i.fromCache {
		i.code, err = i.cp.GetContractCodeFromCache(i.ctx.ContractName)
		if err != nil {
			return err
		}
		i.abi, err = i.cp.GetContractAbiFromCache(i.ctx.ContractName)
		if err != nil {
			return err
		}
	} else {
		i.code, err = i.cp.GetContractCode(i.ctx.ContractName)
		if err != nil {
			return err
		}
		i.abi, err = i.cp.GetContractAbi(i.ctx.ContractName)
		if err != nil {
			return err
		}
	}

	// get contract value
	if i.ctx.TransferAmount != "" {
		var ok bool
		value, ok = new(big.Int).SetString(i.ctx.TransferAmount, 0)
		if !ok {
			return fmt.Errorf("get evm value error")
		}
	}

	// get callee
	callee, err := gaddr.NewEVMAddressFromContractName(i.ctx.ContractName)
	if err != nil {
		return err
	}

	jsonEncoded, ok := i.ctx.Args[evmParamJSONEncoded]
	if !ok || string(jsonEncoded) != "true" {
		input = i.ctx.Args[evmInput]
	} else {
		needDecodeResp = true
		if input, err = i.encodeInvokeInput(); err != nil {
			return err
		}
	}

	var leftOverGas uint64
	var ret []byte
	var contractAddr common.Address // for prototype return
	if i.ctx.Method == initializeMethod {
		ret, contractAddr, leftOverGas, err = i.vm.Create(caller, i.code, gas, value)
		fmt.Printf("leftOverGas: %+v\n", leftOverGas)
		fmt.Printf("contractAddr: %+v\n", contractAddr)
		fmt.Printf("ret: %+v\n", ret)
	} else {
		// Increment the nonce for the next transaction
		stateDB := i.vm.StateDB
		stateDB.SetNonce(caller.Address(), stateDB.GetNonce(caller.Address())+1)
		ret, leftOverGas, err = i.vm.Call(caller, callee.Address(), input, gas, value)
		fmt.Printf("leftOverGas: %+v\n", leftOverGas)
		fmt.Printf("ret: %+v\n", ret)
	}
	if err != nil {
		return err
	}

	// pack response
	i.gasUsed = uint64(contract.MaxLimits.Cpu) - leftOverGas
	fmt.Printf("gasUsed: %d\n", i.gasUsed)
	if needDecodeResp {
		// 执行结果根据 abi 解码，返回 json 格式的数组。
		ret, err = decodeRespWithAbiForEVM(string(i.abi), i.ctx.Method, ret)
		if err != nil {
			return err
		}
	}

	i.ctx.Output = &pb.Response{
		Status: 200,
		Body:   ret,
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
	// TODO
	return nil
}

func (i *evmInstance) deployContract() error {
	// TODO
	return nil
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
