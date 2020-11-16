package mock

import (
	"errors"
	"fmt"
	"time"

	"github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
)

var (
	blockSetItemErr = errors.New("item invalid")
)

type TimerFunc func(contractCtx kernel.KContext, height int64) error

func ContractRegister(f TimerFunc) TimerFunc {
	return f
}

type FakeP2p struct{}

func (p *FakeP2p) GetLocalAddress() string {
	return "local_p2p"
}

func (p *FakeP2p) GetCurrentPeerAddress() []string {
	return []string{"peer_p2p"}
}

type FakeBlock struct {
	proposer         string
	height           int64
	blockid          []byte
	consensusStorage []byte
	timestamp        int64
	nonce            int32
}

func NewBlock(height int) *FakeBlock {
	return &FakeBlock{
		proposer:         "xuper5",
		height:           int64(height),
		blockid:          []byte{byte(height)},
		consensusStorage: []byte{},
		timestamp:        time.Now().UnixNano(),
	}
}

func (b *FakeBlock) MakeBlockId() []byte {
	return b.blockid
}

func (b *FakeBlock) SetItem(param string, value interface{}) error {
	switch param {
	case "nonce":
		if s, ok := value.(int32); ok {
			b.nonce = s
			return nil
		}
	}
	return blockSetItemErr
}

func (b *FakeBlock) GetProposer() string {
	return b.proposer
}

func (b *FakeBlock) GetHeight() int64 {
	return b.height
}

func (b *FakeBlock) GetPreHash() []byte {
	return nil
}

func (b *FakeBlock) GetBlockid() []byte {
	return b.blockid
}

func (b *FakeBlock) GetPubkey() []byte {
	return nil
}
func (b *FakeBlock) GetSign() []byte {
	return nil
}

func (b *FakeBlock) GetConsensusStorage() []byte {
	return b.consensusStorage
}

func (b *FakeBlock) GetTimestamp() int64 {
	return b.timestamp
}

type FakeMeta struct {
	block *FakeBlock
}

func (m *FakeMeta) GetTrunkHeight() int64 {
	return m.block.height
}
func (m *FakeMeta) GetTipBlockid() []byte {
	return m.block.blockid
}

func GetGenesisConsensusConf() []byte {
	return []byte("{\"name\":\"fake\",\"config\":\"\"}")
}

type FakeReader struct {
	storage map[string]map[string][]byte
}

func (r *FakeReader) Get(bucket string, key []byte) ([]byte, error) {
	return r.storage[bucket][string(key)], nil
}

func (r *FakeReader) Select(bucket string, startKey []byte, endKey []byte) error {
	return nil
}

func CreateXModelCache() *FakeReader {
	consensus := map[string][]byte{}
	cache := map[string]map[string][]byte{}
	cache["consensus"] = consensus
	return &FakeReader{
		storage: cache,
	}
}

type FakeLedger struct {
	ledgerSlice   []*FakeBlock
	ledgerMap     map[string]*FakeBlock
	meta          *FakeMeta
	consensusConf []byte
	fakeCache     *FakeReader
}

func NewFakeLedger() *FakeLedger {
	l := &FakeLedger{
		ledgerSlice:   []*FakeBlock{},
		ledgerMap:     map[string]*FakeBlock{},
		meta:          nil,
		consensusConf: GetGenesisConsensusConf(),
		fakeCache:     CreateXModelCache(),
	}
	for i := 0; i < 3; i++ {
		l.put(NewBlock(i))
	}
	return l
}

func (l *FakeLedger) GetTipSnapShot() context.FakeXMReader {
	return l.fakeCache
}

func (l *FakeLedger) VerifyMerkle(context.BlockInterface) error {
	return nil
}

func (l *FakeLedger) GetGenesisConsensusConf() []byte {
	return l.consensusConf
}

func (l *FakeLedger) put(block *FakeBlock) error {
	l.ledgerSlice = append(l.ledgerSlice, block)
	id := fmt.Sprintf("%x", block.blockid)
	l.ledgerMap[id] = block
	l.meta = &FakeMeta{
		block: block,
	}
	return nil
}

func (l *FakeLedger) GetMeta() context.MetaInterface {
	if len(l.ledgerSlice) == 0 {
		return nil
	}
	return l.meta
}

func (l *FakeLedger) QueryBlock(blockId []byte) (context.BlockInterface, error) {
	id := fmt.Sprintf("%x", blockId)
	return l.ledgerMap[id], nil
}

func (l *FakeLedger) QueryBlockHeader(blockId []byte) (context.BlockInterface, error) {
	id := fmt.Sprintf("%x", blockId)
	return l.ledgerMap[id], nil
}

func (l *FakeLedger) QueryBlockByHeight(height int64) (context.BlockInterface, error) {
	return l.ledgerSlice[height], nil
}

func (l *FakeLedger) GetConsensusConf() []byte {
	return l.consensusConf
}

type FakeContext struct {
	cache map[string][]byte
}

func (c *FakeContext) Invoke(method string, args map[string][]byte) (*contract.Response, error) {
	v, ok := c.cache[method]
	if !ok {
		return nil, nil
	}
	return &contract.Response{
		Body: v,
	}, nil
}

func (c *FakeContext) Release() error {
	return nil
}

func (c *FakeContext) ResourceUsed() contract.Limits {
	return contract.Limits{}
}

type ContextConfig struct {
	XMCache      interface{}
	ContractName string
}

type FakeManager struct {
	storage map[string]map[string][]byte
}

func NewFakeManager() FakeManager {
	s := map[string]map[string][]byte{}
	cache := map[string][]byte{}
	s["updateConsensus"] = cache
	s["readConsensus"] = cache
	return FakeManager{
		storage: s,
	}
}

func (m *FakeManager) NewContext(cfg *contract.ContextConfig) (FakeContext, error) {
	v, ok := m.storage[cfg.ContractName]
	if !ok {
		return FakeContext{
			cache: map[string][]byte{},
		}, nil
	}
	return FakeContext{
		cache: v,
	}, nil
}

type FakeKContextImpl struct {
	desc []byte
}

func (kctx *FakeKContextImpl) Arg() []byte {
	return kctx.desc
}

func NewFakeKContextImpl(desc []byte) *FakeKContextImpl {
	return &FakeKContextImpl{
		desc: desc,
	}
}
