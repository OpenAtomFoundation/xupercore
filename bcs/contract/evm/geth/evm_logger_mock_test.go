package geth

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type mockEVMLogger struct {
}

func (m mockEVMLogger) CaptureTxStart(gasLimit uint64) {
	fmt.Printf("CaptureTxStart(), gasLimit: %d\n", gasLimit)
}

func (m mockEVMLogger) CaptureTxEnd(restGas uint64) {
	fmt.Printf("CaptureTxEnd(), restGas: %d\n", restGas)
}

func (m mockEVMLogger) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	fmt.Printf("CaptureStart(), env: %+v\n"+
		"\t%+v -> %+v, create: %+v\n"+
		"\tinput: %+v\n"+
		"\tgas: %+v, value: %+v\n",
		env, from, to, create, input, gas, value)
}

func (m mockEVMLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	fmt.Printf("CaptureEnd(), output: %+v\n"+
		"\tgasUsed: %+v\n"+
		"\ttime: %+v\n"+
		"\terr: %+v\n",
		output, gasUsed, t, err)
}

func (m mockEVMLogger) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	fmt.Printf("CaptureEnter(), typ: %+v\n"+
		"\t%+v -> %+v\n"+
		"\tinput: %+v\n"+
		"\tgas: %+v, value: %+v\n",
		typ, from, to, input, gas, value)
}

func (m mockEVMLogger) CaptureExit(output []byte, gasUsed uint64, err error) {
	fmt.Printf("CaptureExit()\n"+
		"\toutput: %+v\n"+
		"\tgasUsed: %+v\n"+
		"\terr: %+v\n",
		output, gasUsed, err)
}

func (m mockEVMLogger) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	fmt.Printf("CaptureState(), pc: %+v, op: %+v, gas: %+v, cost: %+v\n"+
		"\trData: %+v\n"+
		"\tdepth: %+v, err: %+v\n",
		pc, op, gas, cost, rData, depth, err)
	stack := scope.Stack.Data()
	fmt.Printf("\tscopt.Stack: len(%d)\n", len(stack))
	for idx, data := range stack {
		fmt.Printf("\t\t%d: %+v\n", idx, data)
	}
}

func (m mockEVMLogger) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	fmt.Printf("CaptureFault(), pc: %+v, op: %+v, gas: %+v, cost: %+v, depth: %+v\n"+
		"\terr: %+v\n",
		pc, op, gas, cost, depth, err)
	stack := scope.Stack.Data()
	fmt.Printf("\tscopt.Stack: len(%d)\n", len(stack))
	for idx, data := range stack {
		fmt.Printf("\t\t%d: %+v\n", idx, data)
	}
}
