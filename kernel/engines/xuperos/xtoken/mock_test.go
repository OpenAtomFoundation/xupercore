package xtoken

import (
	"math/big"
	"sort"
	"strings"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

type FakeKContext struct {
	args      map[string][]byte
	data      map[string]map[string][]byte
	initiator string
}

func NewFakeKContext(args map[string][]byte, data map[string]map[string][]byte) *FakeKContext {
	return &FakeKContext{
		args:      args,
		data:      data,
		initiator: "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
	}
}

func (c *FakeKContext) EmitAsyncTask(event string, args interface{}) error {
	return nil
}

func (c *FakeKContext) Args() map[string][]byte {
	return c.args
}

func (c *FakeKContext) Initiator() string {
	return c.initiator
}

func (c *FakeKContext) Caller() string {
	return ""
}

func (c *FakeKContext) AuthRequire() []string {
	return []string{"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY", "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"}
}

func (c *FakeKContext) GetAccountAddresses(accountName string) ([]string, error) {
	return nil, nil
}

func (c *FakeKContext) VerifyContractPermission(initiator string, authRequire []string, contractName string, methodName string) (bool, error) {
	return true, nil
}

func (c *FakeKContext) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	return nil
}

func (c *FakeKContext) RWSet() *contract.RWSet {
	return nil
}

func (c *FakeKContext) AddEvent(events ...*protos.ContractEvent) {}

func (c *FakeKContext) Flush() error {
	return nil
}

func (c *FakeKContext) Get(bucket string, key []byte) ([]byte, error) {
	if _, ok := c.data[bucket]; !ok {
		return nil, nil
	}
	return c.data[bucket][utils.F(key)], nil
}

func (c *FakeKContext) Select(bucket string, startKey []byte, endKey []byte) (contract.Iterator, error) {
	iter := newFakeIterator(c.data, bucket, startKey, endKey)
	return iter, nil
}

func (c *FakeKContext) Put(bucket string, key, value []byte) error {
	if _, ok := c.data[bucket]; !ok {
		a := make(map[string][]byte)
		a[utils.F(key)] = value
		c.data[bucket] = a
	}
	c.data[bucket][utils.F(key)] = value
	return nil
}

func (c *FakeKContext) Del(bucket string, key []byte) error {
	return nil
}

func (c *FakeKContext) AddResourceUsed(delta contract.Limits) {}

func (c *FakeKContext) ResourceLimit() contract.Limits {
	return contract.Limits{
		Cpu:    0,
		Memory: 0,
		Disk:   0,
		XFee:   0,
	}
}

func (c *FakeKContext) Call(module, contract, method string, args map[string][]byte) (*contract.Response, error) {
	return nil, nil
}

func (c *FakeKContext) UTXORWSet() *contract.UTXORWSet {
	return &contract.UTXORWSet{
		Rset: []*protos.TxInput{},
		WSet: []*protos.TxOutput{},
	}
}
func (c *FakeKContext) Transfer(from string, to string, amount *big.Int) error {
	return nil
}
func (c *FakeKContext) QueryBlock(blockid []byte) (*xldgpb.InternalBlock, error) {
	return &xldgpb.InternalBlock{}, nil
}
func (c *FakeKContext) QueryTransaction(txid []byte) (*pb.Transaction, error) {
	return &pb.Transaction{}, nil
}

type FakeIterator struct {
	start, end []byte
	allData    map[string][]byte
	sortedKey  []string
	index      int
}

func newFakeIterator(m map[string]map[string][]byte, bucket string, start, end []byte) *FakeIterator {
	value := m[bucket]
	keys := make([]string, 0, len(value))

	for k := range value {
		if strings.HasPrefix(k, utils.F(start)) {
			keys = append(keys, k)
		}
	}
	if len(keys) > 0 {
		sort.Strings(keys)
	}
	return &FakeIterator{
		start:     start,
		end:       end,
		sortedKey: keys,
		allData:   value,
		index:     -1,
	}
}

func (i *FakeIterator) Key() []byte {
	return []byte(i.sortedKey[i.index])
}

func (i *FakeIterator) Value() []byte {
	key := (i.sortedKey[i.index])
	return i.allData[key]
}

func (i *FakeIterator) Next() bool {
	i.index++
	return i.index < len(i.sortedKey)
}

func (i *FakeIterator) Error() error {
	return nil
}

func (i *FakeIterator) Close() {
}
