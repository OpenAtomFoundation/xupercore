/* package geth
This package will be extracted (and build) as a plugin to avoid license issue.
Notice: you'd better not to depend on the package in xupercore repo which is outside this package
*/
package geth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	gaddr "github.com/xuperchain/xupercore/bcs/contract/evm/geth/address"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)


const (
	evmInput = "input"
)

func init() {
	bridge.Register(bridge.TypeEvm, "geth", newEvmCreator)
}

type evmCreator struct {
	blockCtx vm.BlockContext
	chainCfg *params.ChainConfig
	vmConfig vm.Config
}

func newEvmCreator(_ *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	return &evmCreator{
		blockCtx: vm.BlockContext{
			CanTransfer: canTransfer,
			Transfer: transfer,
		},
		chainCfg: new(params.ChainConfig),
		vmConfig: vm.Config{
			Debug: true,
		},
	}, nil
}

// CreateInstance instances an evm virtual machine instance which can run a single contract call
func (e *evmCreator) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	origin, err := gaddr.NewEVMAddressFromXchainAccount(ctx.Initiator)
	if err != nil {
		return nil, err
	}
	txCtx := vm.TxContext{
		Origin: origin.Address(),
		GasPrice: big.NewInt(1),
	}
	evm := vm.NewEVM(e.blockCtx, txCtx, newStateDB(ctx, cp), e.chainCfg, e.vmConfig)
	return &evmInstance{
		vm:        evm,
		ctx:       ctx,
		cp:        cp,
		fromCache: ctx.ReadFromCache,
	}, nil
}

func (e *evmCreator) RemoveCache(_ string) {
}

// methods for vm.BlockContext
func transfer(db vm.StateDB, from common.Address, to common.Address, amount *big.Int) {
	if amount == nil || amount.Sign() <= 0 {
		return
	}
	db.SubBalance(from, amount)
	db.AddBalance(to, amount)
}

func canTransfer(db vm.StateDB, from common.Address, amount *big.Int) bool {
	_, addrType, err := gaddr.DetermineEVMAddress(from)
	if err != nil {
		return false
	}

	// return directly when from is xchain address or contract account
	// only transfer from a contract name works
	if addrType == gaddr.ContractNameType || addrType == gaddr.XchainAddrType {
		return false
	}

	return db.GetBalance(from).Cmp(amount) >= 0
}
