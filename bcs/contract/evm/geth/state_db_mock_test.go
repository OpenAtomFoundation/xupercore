package geth

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	gaddr "github.com/xuperchain/xupercore/bcs/contract/evm/geth/address"
)

type mockStateDB struct {
	Addr2Nonce    map[common.Address]uint64
	Addr2CodeHash map[common.Address]common.Hash
	Balances      map[common.Address]*big.Int
	Codes         map[common.Address][]byte
	UtxoExt       map[common.Address]map[common.Hash]common.Hash
	Version       int
}

func newMockStateDB() *mockStateDB {
	return &mockStateDB{
		Addr2Nonce:    map[common.Address]uint64{},
		Addr2CodeHash: map[common.Address]common.Hash{},
		Balances: map[common.Address]*big.Int{
			common.HexToAddress("3131313231313131313131313131313131313131"): big.NewInt(0),
		},
		UtxoExt: map[common.Address]map[common.Hash]common.Hash{},
		Codes:   map[common.Address][]byte{},
	}
}

func (m mockStateDB) CreateAccount(address common.Address) {
	m.Balances[address] = big.NewInt(0)
}

func (m mockStateDB) SubBalance(address common.Address, amount *big.Int) {
	fmt.Printf("SubBalance( %+v, %v)\n", address, amount)
	if amount == nil {
		return
	}
	balance := m.Balances[address]
	if balance == nil || balance.Cmp(amount) < 0 {
		return
	}
	balance.Sub(balance, amount)
}

func (m mockStateDB) AddBalance(address common.Address, amount *big.Int) {
	fmt.Printf("AddBalance( %+v, %v)\n", address, amount)
	if amount == nil {
		return
	}
	balance := m.Balances[address]
	if balance == nil {
		balance = big.NewInt(0)
	}
	m.Balances[address].Add(balance, amount)
}

func (m mockStateDB) GetBalance(address common.Address) *big.Int {
	fmt.Printf("GetBalance(address: %s)\n", address)
	balance := m.Balances[address]
	if balance == nil {
		return big.NewInt(0)
	}
	return balance
}

func (m mockStateDB) GetNonce(address common.Address) uint64 {
	return m.Addr2Nonce[address]
}

func (m mockStateDB) SetNonce(address common.Address, u uint64) {
	m.Addr2Nonce[address] = u
}

func (m mockStateDB) GetCodeHash(address common.Address) common.Hash {
	return m.Addr2CodeHash[address]
}

func (m mockStateDB) GetCode(address common.Address) []byte {
	contractName, err := gaddr.EVMAddressToContractName(address)
	fmt.Printf("GetCode(%v->%v)\n", address.String(), contractName)
	if err == nil && (contractName == "storage") {
		code, err := binCode(contractName)
		if err == nil {
			return code
		}
	}
	return m.Codes[address]
}

func (m mockStateDB) SetCode(address common.Address, bytes []byte) {
	m.Codes[address] = bytes
}

func (m mockStateDB) GetCodeSize(address common.Address) int {
	panic("implement me")
}

func (m mockStateDB) AddRefund(u uint64) {
	panic("implement me")
}

func (m mockStateDB) SubRefund(u uint64) {
	panic("implement me")
}

func (m mockStateDB) GetRefund() uint64 {
	panic("implement me")
}

func (m mockStateDB) GetCommittedState(address common.Address, hash common.Hash) common.Hash {
	panic("implement me")
}

func (m mockStateDB) GetState(contract common.Address, key common.Hash) common.Hash {
	fmt.Printf("GetState(%s.%s)\n", contract.String(), key.String())
	contractUtxoExt, ok := m.UtxoExt[contract]
	if !ok {
		return common.Hash{}
	}
	return contractUtxoExt[key]
}

func (m mockStateDB) SetState(contract common.Address, key common.Hash, value common.Hash) {
	fmt.Printf("SetState(%s.%s)=%s\n", contract.String(), key.String(), value.String())
	_, ok := m.UtxoExt[contract]
	if !ok {
		m.UtxoExt[contract] = map[common.Hash]common.Hash{
			key: value,
		}
	} else {
		m.UtxoExt[contract][key] = value
	}
}

func (m mockStateDB) Suicide(address common.Address) bool {
	panic("implement me")
}

func (m mockStateDB) HasSuicided(address common.Address) bool {
	panic("implement me")
}

func (m mockStateDB) Exist(address common.Address) bool {
	_, exist := m.Balances[address]
	return exist
}

func (m mockStateDB) Empty(address common.Address) bool {
	panic("implement me")
}

func (m mockStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	panic("implement me")
}

func (m mockStateDB) AddressInAccessList(addr common.Address) bool {
	panic("implement me")
}

func (m mockStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	panic("implement me")
}

func (m mockStateDB) AddAddressToAccessList(addr common.Address) {
	panic("implement me")
}

func (m mockStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	panic("implement me")
}

func (m mockStateDB) RevertToSnapshot(i int) {
	// TODO
	m.Version = i
}

func (m mockStateDB) Snapshot() int {
	// TODO
	version := m.Version
	m.Version++
	return version
}

func (m mockStateDB) AddLog(log *types.Log) {
	panic("implement me")
}

func (m mockStateDB) AddPreimage(hash common.Hash, bytes []byte) {
	panic("implement me")
}

func (m mockStateDB) ForEachStorage(address common.Address, f func(common.Hash, common.Hash) bool) error {
	panic("implement me")
}
