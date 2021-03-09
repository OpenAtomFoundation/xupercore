package mock

import (
	"errors"
	"fmt"
	"time"

	"github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	blockSetItemErr = errors.New("item invalid")
)

type FakeP2p struct{}

func (p *FakeP2p) GetLocalAccount() string {
	return "local_p2p"
}

func (p *FakeP2p) GetCurrentPeerAddress() []string {
	return []string{"peer_p2p"}
}

type FakeBlock struct {
	Proposer         string
	Height           int64
	Blockid          []byte
	ConsensusStorage []byte
	Timestamp        int64
	Nonce            int32
	PublicKey        string
	Sign             []byte
}

func NewBlock(height int) *FakeBlock {
	return &FakeBlock{
		Height:           int64(height),
		Blockid:          []byte{byte(height)},
		ConsensusStorage: []byte{},
		Timestamp:        time.Now().UnixNano(),
	}
}

func (b *FakeBlock) MakeBlockId() ([]byte, error) {
	return b.Blockid, nil
}

func (b *FakeBlock) SetItem(param string, value interface{}) error {
	switch param {
	case "nonce":
		if s, ok := value.(int32); ok {
			b.Nonce = s
			return nil
		}
	}
	return blockSetItemErr
}

func (b *FakeBlock) GetProposer() []byte {
	return []byte(b.Proposer)
}

func (b *FakeBlock) GetHeight() int64 {
	return b.Height
}

func (b *FakeBlock) GetPreHash() []byte {
	return nil
}

func (b *FakeBlock) GetBlockid() []byte {
	return b.Blockid
}

func (b *FakeBlock) GetPublicKey() string {
	return b.PublicKey
}
func (b *FakeBlock) GetSign() []byte {
	return b.Sign
}

func (b *FakeBlock) GetConsensusStorage() ([]byte, error) {
	return b.ConsensusStorage, nil
}

func (b *FakeBlock) GetTimestamp() int64 {
	return b.Timestamp
}

type FakeMeta struct {
	block *FakeBlock
}

func (m *FakeMeta) GetTrunkHeight() int64 {
	return m.block.Height
}
func (m *FakeMeta) GetTipBlockid() []byte {
	return m.block.Blockid
}

func GetGenesisConsensusConf() []byte {
	return []byte("{\"name\":\"fake\",\"config\":\"\"}")
}

type FakeLedger struct {
	ledgerSlice   []*FakeBlock
	ledgerMap     map[string]*FakeBlock
	consensusConf []byte
	sandbox       *FakeSandBox
}

type FakeSandBox struct {
	storage map[string]map[string][]byte
}

func (s *FakeSandBox) Get(bucket string, key []byte) ([]byte, error) {
	if _, ok := s.storage[bucket]; !ok {
		return nil, nil
	}
	return s.storage[bucket][utils.F(key)], nil
}

func (s *FakeSandBox) SetContext(bucket string, key, value []byte) {
	if _, ok := s.storage[bucket]; ok {
		s.storage[bucket][utils.F(key)] = value
		return
	}
	addition := make(map[string][]byte)
	addition[utils.F(key)] = value
	s.storage[bucket] = addition
}

func NewFakeLedger(conf []byte) *FakeLedger {
	a := &FakeSandBox{
		storage: make(map[string]map[string][]byte),
	}
	l := &FakeLedger{
		ledgerSlice:   []*FakeBlock{},
		ledgerMap:     map[string]*FakeBlock{},
		consensusConf: conf,
		sandbox:       a,
	}
	for i := 0; i < 3; i++ {
		l.Put(NewBlock(i))
	}
	return l
}

func (l *FakeLedger) VerifyMerkle(context.BlockInterface) error {
	return nil
}

func (l *FakeLedger) GetGenesisConsensusConf() []byte {
	return l.consensusConf
}

func (l *FakeLedger) Put(block *FakeBlock) error {
	l.ledgerSlice = append(l.ledgerSlice, block)
	id := fmt.Sprintf("%x", block.Blockid)
	l.ledgerMap[id] = block
	return nil
}

func (l *FakeLedger) QueryBlock(blockId []byte) (ledger.BlockHandle, error) {
	id := fmt.Sprintf("%x", blockId)
	return l.ledgerMap[id], nil
}

func (l *FakeLedger) QueryBlockByHeight(height int64) (ledger.BlockHandle, error) {
	return l.ledgerSlice[height], nil
}

func (l *FakeLedger) GetConsensusConf() ([]byte, error) {
	return l.consensusConf, nil
}

func (l *FakeLedger) GetTipBlock() ledger.BlockHandle {
	if len(l.ledgerSlice) == 0 {
		return nil
	}
	return l.ledgerSlice[len(l.ledgerSlice)-1]
}

func (l *FakeLedger) GetTipXMSnapshotReader() (ledger.XMSnapshotReader, error) {
	return l.sandbox, nil
}

func (l *FakeLedger) CreateSnapshot(blkId []byte) (ledger.XMReader, error) {
	return nil, nil
}

func (l *FakeLedger) GetTipSnapshot() (ledger.XMReader, error) {
	return nil, nil
}

type FakeKContext struct {
	args map[string][]byte
	m    map[string]map[string][]byte
}

func NewFakeKContext(args map[string][]byte, m map[string]map[string][]byte) *FakeKContext {
	return &FakeKContext{
		args: args,
		m:    m,
	}
}

func (c *FakeKContext) Args() map[string][]byte {
	return c.args
}

func (c *FakeKContext) Initiator() string {
	return ""
}

func (c *FakeKContext) AuthRequire() []string {
	return nil
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

func (c *FakeKContext) Get(bucket string, key []byte) ([]byte, error) {
	if _, ok := c.m[bucket]; !ok {
		return nil, nil
	}
	return c.m[bucket][utils.F(key)], nil
}

func (c *FakeKContext) Select(bucket string, startKey []byte, endKey []byte) (contract.Iterator, error) {
	return nil, nil
}

func (c *FakeKContext) Put(bucket string, key, value []byte) error {
	if _, ok := c.m[bucket]; !ok {
		a := make(map[string][]byte)
		a[utils.F(key)] = value
		c.m[bucket] = a
	}
	c.m[bucket][utils.F(key)] = value
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

type FakeManager struct {
	R *FakeRegistry
}

func (m *FakeManager) NewContext(cfg *contract.ContextConfig) (contract.Context, error) {
	return nil, nil
}

func (m *FakeManager) NewStateSandbox(cfg *contract.SandboxConfig) (contract.StateSandbox, error) {
	return nil, nil
}

func (m *FakeManager) GetKernRegistry() contract.KernRegistry {
	return m.R
}

type FakeRegistry struct {
	M map[string]contract.KernMethod
}

func (r *FakeRegistry) RegisterKernMethod(contract, method string, handler contract.KernMethod) {
	r.M[method] = handler
}

func (r *FakeRegistry) GetKernMethod(contract, method string) (contract.KernMethod, error) {
	return nil, nil
}
