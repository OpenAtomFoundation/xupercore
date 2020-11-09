package mock

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xuperchain/xupercore/kernel/consensus/context"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

type TimerFunc func(contractCtx cctx.FakeKContext, height int64) error

func ContractRegister(f TimerFunc) TimerFunc {
	return f
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

func (b *FakeBlock) GetProposer() string {
	return b.proposer
}

func (b *FakeBlock) GetHeight() int64 {
	return b.height
}

func (b *FakeBlock) GetBlockid() []byte {
	return b.blockid
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
	args := context.ConsensusConfig{
		ConsensusName: "fake",
		BeginBlockid:  []byte{byte(0)},
		Timestamp:     time.Now().UnixNano(),
		BaseComponent: context.Empty,
	}
	result, err := json.Marshal(args)
	if err != nil {
		return nil
	}
	return result
}

type FakeLedger struct {
	ledgerSlice   []*FakeBlock
	ledgerMap     map[string]*FakeBlock
	meta          *FakeMeta
	consensusConf []byte
}

func NewFakeLedger() *FakeLedger {
	l := &FakeLedger{
		ledgerSlice:   []*FakeBlock{},
		ledgerMap:     map[string]*FakeBlock{},
		meta:          nil,
		consensusConf: GetGenesisConsensusConf(),
	}
	for i := 0; i < 3; i++ {
		l.put(NewBlock(i))
	}
	return l
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

func (l *FakeLedger) QueryBlock(blockId []byte) context.BlockInterface {
	id := fmt.Sprintf("%x", blockId)
	return l.ledgerMap[id]
}

func (l *FakeLedger) QueryBlockByHeight(height int64) context.BlockInterface {
	return l.ledgerSlice[height]
}

func (l *FakeLedger) Truncate() error {
	return nil
}

func (l *FakeLedger) GetConsensusConf() []byte {
	return l.consensusConf
}

func (l *FakeLedger) GetGenesisBlock() context.BlockInterface {
	if len(l.ledgerSlice) == 0 {
		return nil
	}
	return l.ledgerSlice[0]
}
