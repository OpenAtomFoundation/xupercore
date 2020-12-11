package mock

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	xuperp2p "github.com/xuperchain/xupercore/kernel/network/pb"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	Miner  = "default_miner"
	BcName = "xuper5"

	blockSetItemErr = errors.New("item invalid")
)

type FakeCryptoClient struct{}

func (cc *FakeCryptoClient) VerifyAddressUsingPublicKey(address string, pub *ecdsa.PublicKey) (bool, uint8) {
	return true, 0
}

func (cc *FakeCryptoClient) VerifyECDSA(k *ecdsa.PublicKey, signature, msg []byte) (bool, error) {
	return true, nil
}

func (cc *FakeCryptoClient) GetEcdsaPublicKeyFromJsonStr(keyStr string) (*ecdsa.PublicKey, error) {
	return nil, nil
}

func (cc *FakeCryptoClient) GetEcdsaPublicKeyFromFile(filename string) (*ecdsa.PublicKey, error) {
	return nil, nil
}

func (cc *FakeCryptoClient) GetEcdsaPrivateKeyFromFile(filename string) (*ecdsa.PrivateKey, error) {
	return nil, nil
}

func (cc *FakeCryptoClient) SignECDSA(k *ecdsa.PrivateKey, msg []byte) (signature []byte, err error) {
	return nil, nil
}

type FakeP2p struct{}

func (p *FakeP2p) GetLocalAccount() string {
	return Miner
}

func (p *FakeP2p) GetCurrentPeerAddress() []string {
	return []string{"peer_p2p"}
}

func (p *FakeP2p) NewSubscriber(xuperp2p.XuperMessage_MessageType, interface{}, ...p2p.SubscriberOption) p2p.Subscriber {
	return nil
}

func (p *FakeP2p) SendMessage(xcontext.BaseCtx, *xuperp2p.XuperMessage, ...p2p.OptionFunc) error {
	return nil
}

func (p *FakeP2p) Register(p2p.Subscriber) error {
	return nil
}

func (p *FakeP2p) UnRegister(p2p.Subscriber) error {
	return nil
}

type FakeBlock struct {
	proposer         string
	height           int64
	blockid          []byte
	consensusStorage []byte
	timestamp        int64
	nonce            uint64
}

func NewBlock(height int, storage []byte) *FakeBlock {
	b := &FakeBlock{
		proposer:         Miner,
		height:           int64(height),
		consensusStorage: storage,
		timestamp:        time.Now().UnixNano(),
	}
	b.blockid = b.MakeBlockId()
	return b
}

func (b *FakeBlock) MakeBlockId() []byte {
	copyBlock := &FakeBlock{
		proposer:         b.proposer,
		height:           b.height,
		consensusStorage: b.consensusStorage,
		timestamp:        b.timestamp,
		nonce:            b.nonce,
	}
	h := sha256.New()
	bb, _ := json.Marshal(copyBlock)
	h.Write([]byte(bb))
	bs := h.Sum(nil)
	return bs
}

func (b *FakeBlock) SetItem(param string, value interface{}) error {
	switch param {
	case "nonce":
		if s, ok := value.(uint64); ok {
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

type FakeLedger struct {
	ledgerSlice   []*FakeBlock
	ledgerMap     map[string]*FakeBlock
	consensusConf []byte
}

// NewFakeLedger 需提供一个共识文件，返回一个ledger，该账本含有三个区块
func NewFakeLedger(conf []byte) *FakeLedger {
	l := &FakeLedger{
		ledgerSlice:   []*FakeBlock{},
		ledgerMap:     map[string]*FakeBlock{},
		consensusConf: conf,
	}
	return l
}

func (l *FakeLedger) PutBlock(block *FakeBlock) error {
	l.ledgerSlice = append(l.ledgerSlice, block)
	id := fmt.Sprintf("%x", block.blockid)
	l.ledgerMap[id] = block
	return nil
}

func (l *FakeLedger) GetTipSnapShot() cctx.XMReader {
	return nil
}

func (l *FakeLedger) VerifyMerkle(cctx.BlockInterface) error {
	return nil
}

func (l *FakeLedger) QueryBlock(blockId []byte) (cctx.BlockInterface, error) {
	id := fmt.Sprintf("%x", blockId)
	return l.ledgerMap[id], nil
}

func (l *FakeLedger) QueryBlockHeader(blockId []byte) (cctx.BlockInterface, error) {
	id := fmt.Sprintf("%x", blockId)
	return l.ledgerMap[id], nil
}

func (l *FakeLedger) QueryBlockByHeight(height int64) (cctx.BlockInterface, error) {
	return l.ledgerSlice[height], nil
}

func (l *FakeLedger) GetConsensusConf() []byte {
	return l.consensusConf
}

func (l *FakeLedger) GetGenesisConsensusConf() []byte {
	return l.consensusConf
}

func NewFakeLogger() logs.Logger {
	confFile := utils.GetCurFileDir()
	confFile = filepath.Join(confFile, "config/log.yaml")
	logDir := utils.GetCurFileDir()
	logDir = filepath.Join(logDir, "logs")

	logs.InitLog(confFile, logDir)
	log, _ := logs.NewLogger("", "consensus_test")
	return log
}

// NewConsensusCtx 返回除ledger以外的所有所需的共识上下文
func NewConsensusCtx() cctx.ConsensusCtx {
	return cctx.ConsensusCtx{
		BcName:       BcName,
		P2p:          &FakeP2p{},
		CryptoClient: &FakeCryptoClient{},
		BCtx: xcontext.BaseCtx{
			XLog: NewFakeLogger(),
		},
	}
}
