package burrow

import (
	evmaddr "github.com/xuperchain/xupercore/bcs/contract/evm/burrow/address"
	"math/big"
	"time"

	"github.com/hyperledger/burrow/acm"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/permission"

	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)

type stateManager struct {
	ctx *bridge.Context
}

func newStateManager(ctx *bridge.Context) *stateManager {
	return &stateManager{
		ctx: ctx,
	}
}

// Get an account by its address return nil if it does not exist (which should not be an error)
func (s *stateManager) GetAccount(address crypto.Address) (*acm.Account, error) {
	addr, addrType, err := evmaddr.DetermineEVMAddress(address)
	if err != nil {
		return nil, nil
	}

	var evmCode []byte
	if addrType == evmaddr.ContractNameType {
		v, err := s.ctx.State.Get("contract", evmCodeKey(addr))
		if err != nil {
			return nil, nil
		}
		evmCode = v
	}

	var balance *big.Int
	return &acm.Account{
		Address:     address,
		Balance:     balance,
		EVMCode:     evmCode,
		Permissions: permission.AllAccountPermissions,
	}, nil
}

// Retrieve a 32-byte value stored at key for the account at address, return Zero256 if key does not exist but
// error if address does not
func (s *stateManager) GetStorage(address crypto.Address, key binary.Word256) ([]byte, error) {
	//log.Debug("get storage for evm", "contract", s.ctx.ContractName, "address", address.String(), "key", key.String())
	contractName, err := evmaddr.DetermineContractNameFromEVM(address)
	if err != nil {
		return nil, nil
	}
	v, err := s.ctx.State.Get(contractName, key.Bytes())
	if err != nil {
		return binary.Zero256.Bytes(), nil
	}
	return v, nil
}

// Updates the fields of updatedAccount by address, creating the account
// if it does not exist
func (s *stateManager) UpdateAccount(updatedAccount *acm.Account) error {
	return nil
}

// Remove the account at address
func (s *stateManager) RemoveAccount(address crypto.Address) error {
	return nil
}

// Store a 32-byte value at key for the account at address, setting to Zero256 removes the key
func (s *stateManager) SetStorage(address crypto.Address, key binary.Word256, value []byte) error {
	//log.Debug("set storage for evm", "contract", s.ctx.ContractName, "address", address.String(), "key", key.String())
	contractName, err := evmaddr.DetermineContractNameFromEVM(address)
	if err != nil {
		return err
	}
	return s.ctx.State.Put(contractName, key.Bytes(), value)
}

// Transfer native token
func (s *stateManager) Transfer(from, to crypto.Address, amount *big.Int) error {
	fromAddr, addrType, err := evmaddr.DetermineEVMAddress(from)
	if err != nil {
		return err
	}

	// return directly when from is xchain address or contract account
	// only transfer from a contract name works
	if addrType == evmaddr.ContractAccountType || addrType == evmaddr.XchainAddrType {
		return nil
	}

	toAddr, addrType, err := evmaddr.DetermineEVMAddress(to)
	if err != nil {
		return err
	}

	if addrType == evmaddr.ContractAccountType {
		// 构造完整的合约账户
		toAddr = "XC" + toAddr + "@" + s.ctx.ChainName
	}

	return s.ctx.State.Transfer(fromAddr, toAddr, amount)
}

type blockStateManager struct {
	ctx *bridge.Context
}

func newBlockStateManager(ctx *bridge.Context) *blockStateManager {
	return &blockStateManager{
		ctx: ctx,
	}
}

// LastBlockHeight
func (s *blockStateManager) LastBlockHeight() uint64 {
	// TODO
	return 0
}

// LastBlockTime
func (s *blockStateManager) LastBlockTime() time.Time {
	// TODO
	return time.Time{}
}

// LastBlockHeight
func (s *blockStateManager) BlockHash(height uint64) ([]byte, error) {
	// TODO
	return nil, nil
}
