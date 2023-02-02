package geth

import (
	"fmt"
	"github.com/xuperchain/xupercore/bcs/contract/evm/geth/address"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/hyperledger/burrow/execution/errors"
	"github.com/hyperledger/burrow/execution/exec"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)

func Test_evmInstance_Exec(t *testing.T) {
	origin, err := address.NewEVMAddressFromContractAccount("XC1111111111111111@xuper")
	if err != nil {
		t.Fatalf("generate origin err")
	}
	type fields struct {
		vm        *vm.EVM
		ctx       *bridge.Context
		cp        bridge.ContractCodeProvider
		code      []byte
		abi       []byte
		gasUsed   uint64
		fromCache bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "deploy",
			fields: fields{
				vm: vm.NewEVM(newBlockContext(),
					vm.TxContext{
						Origin:   origin.Address(),
						GasPrice: big.NewInt(1),
					},
					newMockStateDB(),
					new(params.ChainConfig),
					vm.Config{
						Debug:  true,
						Tracer: &mockEVMLogger{},
					}),
				ctx: &bridge.Context{
					Initiator: "XC1111111111111111@xuper",
					Method:    initializeMethod,
				},
				cp: mockContractCodeProvider{},
			},
		},
		{
			name: "getOwner",
			fields: fields{
				vm: vm.NewEVM(newBlockContext(),
					vm.TxContext{
						Origin:   origin.Address(),
						GasPrice: big.NewInt(1),
					},
					newMockStateDB(),
					new(params.ChainConfig),
					vm.Config{
						Debug:  true,
						Tracer: &mockEVMLogger{},
					}),
				ctx: &bridge.Context{
					Initiator:    "XC1111111111111111@xuper",
					Method:       "getOwner",
					ContractName: "counter",
				},
				cp: mockContractCodeProvider{},
			},
		},
		{
			name: "non Exist",
			fields: fields{
				vm: vm.NewEVM(newBlockContext(),
					vm.TxContext{
						Origin:   origin.Address(),
						GasPrice: big.NewInt(1),
					},
					newMockStateDB(),
					new(params.ChainConfig),
					vm.Config{
						Debug:  true,
						Tracer: &mockEVMLogger{},
					}),
				ctx: &bridge.Context{
					Initiator:    "XC1111111111111111@xuper",
					Method:       "nonExistMethod",
					ContractName: "nonExistContract",
				},
				cp: mockContractCodeProvider{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &evmInstance{
				vm:        tt.fields.vm,
				ctx:       tt.fields.ctx,
				cp:        tt.fields.cp,
				code:      tt.fields.code,
				abi:       tt.fields.abi,
				gasUsed:   tt.fields.gasUsed,
				fromCache: tt.fields.fromCache,
			}
			if err := i.Exec(); (err != nil) != tt.wantErr {
				t.Errorf("evmInstance.Exec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_evmInstance_ResourceUsed(t *testing.T) {
	type fields struct {
		vm        *vm.EVM
		ctx       *bridge.Context
		cp        bridge.ContractCodeProvider
		code      []byte
		abi       []byte
		gasUsed   uint64
		fromCache bool
	}
	tests := []struct {
		name   string
		fields fields
		want   contract.Limits
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &evmInstance{
				vm:        tt.fields.vm,
				ctx:       tt.fields.ctx,
				cp:        tt.fields.cp,
				code:      tt.fields.code,
				abi:       tt.fields.abi,
				gasUsed:   tt.fields.gasUsed,
				fromCache: tt.fields.fromCache,
			}
			if got := i.ResourceUsed(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("evmInstance.ResourceUsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_evmInstance_Release(t *testing.T) {
	type fields struct {
		vm        *vm.EVM
		ctx       *bridge.Context
		cp        bridge.ContractCodeProvider
		code      []byte
		abi       []byte
		gasUsed   uint64
		fromCache bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &evmInstance{
				vm:        tt.fields.vm,
				ctx:       tt.fields.ctx,
				cp:        tt.fields.cp,
				code:      tt.fields.code,
				abi:       tt.fields.abi,
				gasUsed:   tt.fields.gasUsed,
				fromCache: tt.fields.fromCache,
			}
			i.Release()
		})
	}
}

func Test_evmInstance_Abort(t *testing.T) {
	type fields struct {
		vm        *vm.EVM
		ctx       *bridge.Context
		cp        bridge.ContractCodeProvider
		code      []byte
		abi       []byte
		gasUsed   uint64
		fromCache bool
	}
	type args struct {
		msg string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &evmInstance{
				vm:        tt.fields.vm,
				ctx:       tt.fields.ctx,
				cp:        tt.fields.cp,
				code:      tt.fields.code,
				abi:       tt.fields.abi,
				gasUsed:   tt.fields.gasUsed,
				fromCache: tt.fields.fromCache,
			}
			i.Abort(tt.args.msg)
		})
	}
}

func Test_evmInstance_Call(t *testing.T) {
	type fields struct {
		vm        *vm.EVM
		ctx       *bridge.Context
		cp        bridge.ContractCodeProvider
		code      []byte
		abi       []byte
		gasUsed   uint64
		fromCache bool
	}
	type args struct {
		call      *exec.CallEvent
		exception *errors.Exception
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &evmInstance{
				vm:        tt.fields.vm,
				ctx:       tt.fields.ctx,
				cp:        tt.fields.cp,
				code:      tt.fields.code,
				abi:       tt.fields.abi,
				gasUsed:   tt.fields.gasUsed,
				fromCache: tt.fields.fromCache,
			}
			if err := i.Call(tt.args.call, tt.args.exception); (err != nil) != tt.wantErr {
				t.Errorf("evmInstance.Call() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_evmInstance_Log(t *testing.T) {
	type fields struct {
		vm        *vm.EVM
		ctx       *bridge.Context
		cp        bridge.ContractCodeProvider
		code      []byte
		abi       []byte
		gasUsed   uint64
		fromCache bool
	}
	type args struct {
		log *exec.LogEvent
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &evmInstance{
				vm:        tt.fields.vm,
				ctx:       tt.fields.ctx,
				cp:        tt.fields.cp,
				code:      tt.fields.code,
				abi:       tt.fields.abi,
				gasUsed:   tt.fields.gasUsed,
				fromCache: tt.fields.fromCache,
			}
			if err := i.Log(tt.args.log); (err != nil) != tt.wantErr {
				t.Errorf("evmInstance.Log() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_evmInstance_deployContract(t *testing.T) {
	type fields struct {
		vm        *vm.EVM
		ctx       *bridge.Context
		cp        bridge.ContractCodeProvider
		code      []byte
		abi       []byte
		gasUsed   uint64
		fromCache bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &evmInstance{
				vm:        tt.fields.vm,
				ctx:       tt.fields.ctx,
				cp:        tt.fields.cp,
				code:      tt.fields.code,
				abi:       tt.fields.abi,
				gasUsed:   tt.fields.gasUsed,
				fromCache: tt.fields.fromCache,
			}
			if err := i.deployContract(); (err != nil) != tt.wantErr {
				t.Errorf("evmInstance.deployContract() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_evmInstance_encodeDeployInput(t *testing.T) {
	type fields struct {
		vm        *vm.EVM
		ctx       *bridge.Context
		cp        bridge.ContractCodeProvider
		code      []byte
		abi       []byte
		gasUsed   uint64
		fromCache bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &evmInstance{
				vm:        tt.fields.vm,
				ctx:       tt.fields.ctx,
				cp:        tt.fields.cp,
				code:      tt.fields.code,
				abi:       tt.fields.abi,
				gasUsed:   tt.fields.gasUsed,
				fromCache: tt.fields.fromCache,
			}
			got, err := i.encodeDeployInput()
			if (err != nil) != tt.wantErr {
				t.Errorf("evmInstance.encodeDeployInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("evmInstance.encodeDeployInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_evmInstance_encodeInvokeInput(t *testing.T) {
	type fields struct {
		vm        *vm.EVM
		ctx       *bridge.Context
		cp        bridge.ContractCodeProvider
		code      []byte
		abi       []byte
		gasUsed   uint64
		fromCache bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &evmInstance{
				vm:        tt.fields.vm,
				ctx:       tt.fields.ctx,
				cp:        tt.fields.cp,
				code:      tt.fields.code,
				abi:       tt.fields.abi,
				gasUsed:   tt.fields.gasUsed,
				fromCache: tt.fields.fromCache,
			}
			got, err := i.encodeInvokeInput()
			if (err != nil) != tt.wantErr {
				t.Errorf("evmInstance.encodeInvokeInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("evmInstance.encodeInvokeInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeRespWithAbiForEVM(t *testing.T) {
	type args struct {
		abiData  string
		funcName string
		resp     []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeRespWithAbiForEVM(tt.args.abiData, tt.args.funcName, tt.args.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeRespWithAbiForEVM() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeRespWithAbiForEVM() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newBlockContext() vm.BlockContext {
	return vm.BlockContext{
		CanTransfer: mockCanTransfer,
		Transfer:    mockTransfer,
	}
}

func mockCanTransfer(db vm.StateDB, address common.Address, b *big.Int) bool {
	fmt.Printf("CanTransfer(address: %s, needValue: %+v)\n", address, b)
	return db.GetBalance(address).Cmp(b) >= 0
}

func mockTransfer(db vm.StateDB, from common.Address, to common.Address, amount *big.Int) {
	fmt.Printf("Transfer( %+v -> %+v with %v)\n", from, to, amount)
	if amount == nil || amount.Sign() <= 0 {
		return
	}
	db.SubBalance(from, amount)
	db.AddBalance(to, amount)
}
