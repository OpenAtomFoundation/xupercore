// 明确定义该模块需要的上下文信息，方便代码阅读和理解
package context

import (
	"context"
	"crypto/ecdsa"

	"github.com/xuperchain/xupercore/kernel/common/xaddress"
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"

	"github.com/xuperchain/xupercore/kernel/network/p2p"
	xuperp2p "github.com/xuperchain/xupercore/protos"
)

type ConsensusComponent int

const (
	Empty      ConsensusComponent = iota //无
	ChainedBFT                           // 使用了chained-bft组件
)

// BlockInterface 对区块结构的限制
type BlockInterface interface {
	GetProposer() string
	GetHeight() int64
	GetBlockid() []byte
	// ATTENTION: 该存储在block中统一仍叫Justify字段,为了兼容保持名称不变
	GetConsensusStorage() ([]byte, error)
	GetTimestamp() int64
	// 特定共识需要向raw block更新专有字段, eg. PoW挖矿时需更新nonce
	SetItem(string, interface{}) error
	// 使用PoW时需要调用该方法进行散列存在性证明
	MakeBlockId() ([]byte, error)
	// 获取上一区块hash
	GetPreHash() []byte
}

// ConsensusConfig 特定共识的字段标示
type ConsensusConfig struct {
	// 获取本次共识的类型名称
	ConsensusName string `json:"name"`
	// 获取本次共识专属属性
	Config string `json:"config"`
	// 获取本次共识的起始高度
	BeginHeight int64 `json:"-"`
	// 本次共识在consensus slice中的index
	Index int `json:"-"`
}

// LedgerCtxInConsensus 使用到的ledger接口
type LedgerCtxInConsensus interface {
	QueryBlockByHeight(int64) (BlockInterface, error)
	QueryBlockHeader([]byte) (BlockInterface, error)
	GetSnapShotWithBlock(blockId []byte) (ledger.XMSnapshotReader, error)
	GetTipXMSnapshotReader() (ledger.XMSnapshotReader, error) // 获取当前最新快照， 原来utxoVM快照
	GetGenesisConsensusConf() []byte                          // 获取账本创始块共识配置
	GetTipBlock() BlockInterface
	// ConsensusCommit(blockId []byte) bool // 共识向账本发送落盘消息，此后该区块将不被回滚
}

// P2pCtxInConsensus 依赖p2p接口
type P2pCtxInConsensus interface {
	SendMessage(context.Context, *xuperp2p.XuperMessage, ...p2p.OptionFunc) error
	NewSubscriber(xuperp2p.XuperMessage_MessageType, interface{}, ...p2p.SubscriberOption) p2p.Subscriber
	Register(p2p.Subscriber) error
	UnRegister(p2p.Subscriber) error
}

// CryptoClientInConsensus 依赖加密接口
type CryptoClientInConsensus interface {
	GetEcdsaPublicKeyFromJsonStr(keyStr string) (*ecdsa.PublicKey, error)
	GetAddressFromPublicKey(pub *ecdsa.PublicKey) (string, error)
	VerifyAddressUsingPublicKey(address string, pub *ecdsa.PublicKey) (bool, uint8)
	VerifyECDSA(k *ecdsa.PublicKey, signature, msg []byte) (valid bool, err error)
	SignECDSA(k *ecdsa.PrivateKey, msg []byte) (signature []byte, err error)
}

// ConsensusCtx 共识领域级上下文
type ConsensusCtx struct {
	xcontext.BaseCtx
	contract.KernRegistry
	StartHeight  int64
	BcName       string
	Ledger       LedgerCtxInConsensus
	P2p          P2pCtxInConsensus
	CryptoClient *CryptoClient
}

type CryptoClient struct {
	Address      xaddress.Address
	CryptoClient CryptoClientInConsensus
}
