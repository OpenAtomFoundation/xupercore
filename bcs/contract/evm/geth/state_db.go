package geth

import (
	"fmt"
	"math/big"
	"reflect"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/execution/evm/abi"

	gaddr "github.com/xuperchain/xupercore/bcs/contract/evm/geth/address"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	xchainpb "github.com/xuperchain/xupercore/protos"
)

type stateDB struct {
	ctx             *bridge.Context
	cp              bridge.ContractCodeProvider
	transferFrom    string
	balanceCache    map[common.Address]*big.Int
	addr2Nonce      map[common.Address]uint64
	snapshotVersion int
}

func newStateDB(ctx *bridge.Context, cp bridge.ContractCodeProvider) *stateDB {
	return &stateDB{
		ctx:          ctx,
		cp:           cp,
		balanceCache: map[common.Address]*big.Int{},
		addr2Nonce:   map[common.Address]uint64{},
	}
}

func (s stateDB) CreateAccount(address common.Address) {
}

func (s stateDB) SubBalance(address common.Address, b *big.Int) {
	fromAddr, addrType, err := gaddr.DetermineEVMAddress(address)
	if err != nil {
		return
	}
	if addrType == gaddr.ContractAccountType || addrType == gaddr.XchainAddrType {
		return
	}
	s.transferFrom = fromAddr
}

func (s stateDB) AddBalance(address common.Address, amount *big.Int) {
	if s.transferFrom == "" {
		return
	}

	toAddr, err := gaddr.EVMAddress(address).XchainAccount(s.ctx.ChainName)
	if err != nil {
		return
	}

	// canTransfer is checked before invoking
	_ = s.ctx.State.Transfer(s.transferFrom, toAddr, amount)
	s.transferFrom = ""
}

func (s stateDB) GetBalance(address common.Address) *big.Int {
	return s.balanceCache[address]
}

func (s stateDB) GetNonce(address common.Address) uint64 {
	return s.addr2Nonce[address]
}

func (s stateDB) SetNonce(address common.Address, u uint64) {
	s.addr2Nonce[address] = u
}

func (s stateDB) GetCodeHash(address common.Address) common.Hash {
	code := s.GetCode(address)
	ch := codeAndHash{code: code}
	return ch.Hash()
}

func (s stateDB) GetCode(_ common.Address) []byte {
	if s.ctx.ReadFromCache {
		code, _ := s.cp.GetContractCodeFromCache(s.ctx.ContractName)
		return code
	}
	code, _ := s.cp.GetContractCode(s.ctx.ContractName)
	return code
}

func (s stateDB) SetCode(_ common.Address, bytes []byte) {
	key := evmCodeKey(s.ctx.ContractName)
	_ = s.ctx.State.Put("contract", key, bytes)
}

func evmCodeKey(contractName string) []byte {
	return []byte(contractName + "." + "code")
}

func (s stateDB) GetCodeSize(address common.Address) int {
	code := s.GetCode(address)
	return len(code)
}

func (s stateDB) AddRefund(u uint64) {
}

func (s stateDB) SubRefund(u uint64) {
}

func (s stateDB) GetRefund() uint64 {
	// not used
	return 0
}

func (s stateDB) GetCommittedState(_ common.Address, key common.Hash) common.Hash {
	value, err := s.ctx.State.Get(s.ctx.ContractName, key[:])
	if err != nil {
		return common.Hash{}
	}
	return common.BytesToHash(value)
}

func (s stateDB) GetState(_ common.Address, key common.Hash) common.Hash {
	value, err := s.ctx.State.Get(s.ctx.ContractName, key[:])
	if err != nil {
		return common.Hash{}
	}
	return common.BytesToHash(value)
}

func (s stateDB) SetState(_ common.Address, key common.Hash, value common.Hash) {
	_ = s.ctx.State.Put(s.ctx.ContractName, key[:], value[:])
}

func (s stateDB) Suicide(address common.Address) bool {
	// not used
	return false
}

func (s stateDB) HasSuicided(address common.Address) bool {
	// true for skip AddRefund
	return true
}

func (s stateDB) Exist(address common.Address) bool {
	// for CreateAccount
	// return true due to no need to create yet
	return true
}

func (s stateDB) Empty(address common.Address) bool {
	_, _, err := gaddr.DetermineEVMAddress(address)
	return err != nil
}

func (s stateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
}

func (s stateDB) AddressInAccessList(addr common.Address) bool {
	return true
}

func (s stateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return true, true
}

func (s stateDB) AddAddressToAccessList(addr common.Address) {
}

func (s stateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
}

func (s stateDB) RevertToSnapshot(i int) {
	s.snapshotVersion = i
}

func (s stateDB) Snapshot() int {
	version := s.snapshotVersion
	s.snapshotVersion++
	return version
}

func (s stateDB) AddLog(log *types.Log) {
	contractName, _, err := gaddr.DetermineEVMAddress(log.Address)
	if err != nil {
		return
	}

	contractAbiByte, err := s.cp.GetContractAbi(contractName)
	if err != nil {
		return
	}
	event, err := unpackEventFromAbi(contractAbiByte, contractName, log)
	if err != nil {
		return
	}
	s.ctx.Events = append(s.ctx.Events, event)
	s.ctx.State.AddEvent(event)
	return
}

func unpackEventFromAbi(abiByte []byte, contractName string, log *types.Log) (*xchainpb.ContractEvent, error) {
	var eventID abi.EventID
	copy(eventID[:], log.Topics[0].Bytes())
	spec, err := abi.ReadSpec(abiByte)
	if err != nil {
		return nil, err
	}
	eventSpec, ok := spec.EventsByID[eventID]
	if !ok {
		return nil, fmt.Errorf("The Event By ID Not Found ")
	}
	values := abi.GetPackingTypes(eventSpec.Inputs)
	topics := make([]binary.Word256, 0, len(log.Topics))
	for _, topic := range topics {
		topics = append(topics, topic)
	}
	if err := abi.UnpackEvent(eventSpec, topics, log.Data, values...); err != nil {
		return nil, err
	}
	event := &xchainpb.ContractEvent{
		Contract: contractName,
	}
	var uint8type = reflect.TypeOf((*[]uint8)(nil))
	event.Name = eventSpec.Name
	for i := 0; i < len(values); i++ {
		t := reflect.TypeOf(values[i])
		if t == uint8type {
			s := fmt.Sprintf("%x", values[i])
			values[i] = s[1:]
		}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	event.Body = data
	return event, nil
}

func (s stateDB) AddPreimage(hash common.Hash, bytes []byte) {
}

func (s stateDB) ForEachStorage(address common.Address, f func(common.Hash, common.Hash) bool) error {
	return nil
}
