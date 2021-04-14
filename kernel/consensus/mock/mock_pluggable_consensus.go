package mock

import (
	"errors"
	"fmt"
	"time"

	"github.com/xuperchain/xupercore/bcs/network/p2pv2"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/kernel/mock"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
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
	PreHash          []byte
}

func NewBlock(height int) *FakeBlock {
	return &FakeBlock{
		Height:           int64(height),
		Blockid:          []byte{byte(height)},
		ConsensusStorage: []byte{},
		Timestamp:        time.Now().UnixNano(),
		Proposer:         "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN",
		PreHash:          []byte{byte(height - 1)},
	}
}

func (b *FakeBlock) MakeBlockId() ([]byte, error) {
	return b.Blockid, nil
}

func (b *FakeBlock) SetTimestamp(t int64) {
	b.Timestamp = t
}

func (b *FakeBlock) SetProposer(m string) {
	b.Proposer = m
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
	return b.PreHash
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
	fakeReader    FakeXMReader
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
	l.fakeReader = NewFakeXMReader()
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
	if height < 0 {
		return nil, blockSetItemErr
	}
	if int(height) > len(l.ledgerSlice)-1 {
		return nil, blockSetItemErr
	}
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
	return &l.fakeReader, nil
}

func (l *FakeLedger) SetSnapshot(bucket string, key []byte, value []byte) {
	l.fakeReader[string(key)] = FReaderItem{
		Bucket: bucket,
		Key:    key,
		Value:  value,
	}
}

func (l *FakeLedger) GetTipSnapshot() (ledger.XMReader, error) {
	return nil, nil
}

func (l *FakeLedger) SetConsensusStorage(height int, s []byte) {
	if len(l.ledgerSlice)-1 < height {
		return
	}
	l.ledgerSlice[height].ConsensusStorage = s
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
	return "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"
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

func (c *FakeKContext) Call(module, contract, method string, args map[string][]byte) (*contract.Response, error) {
	return nil, nil
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

func (r *FakeRegistry) RegisterShortcut(oldmethod, contract, method string) {
}

type FReaderItem struct {
	Bucket string
	Key    []byte
	Value  []byte
}

type FakeXMReader map[string]FReaderItem

func NewFakeXMReader() FakeXMReader {
	a := make(map[string]FReaderItem)
	return a
}

func (r FakeXMReader) Get(bucket string, key []byte) (*ledger.VersionedData, error) {
	item, ok := r[string(key)]
	if !ok {
		return nil, nil
	}
	return &ledger.VersionedData{
		PureData: &ledger.PureData{
			Bucket: item.Bucket,
			Key:    item.Key,
			Value:  item.Value,
		},
	}, nil
}

func (r *FakeXMReader) Select(bucket string, startKey []byte, endKey []byte) (ledger.XMIterator, error) {
	return nil, nil
}

func NewXContent() *xctx.BaseCtx {
	return &xctx.BaseCtx{}
}

func NewP2P(node string) (p2p.Server, *nctx.NetCtx, error) {
	// 创建p2p
	var Npath string
	switch node {
	case "node":
		Npath = "node"
	case "nodeA":
		Npath = "node1"
	case "nodeB":
		Npath = "node2"
	case "nodeC":
		Npath = "node3"
	}
	ecfg, _ := mock.NewEnvConfForTest("p2pv2/" + Npath + "/conf/env.yaml")
	ctx, _ := nctx.NewNetCtx(ecfg)
	p2pNode := p2pv2.NewP2PServerV2()
	return p2pNode, ctx, nil
}
