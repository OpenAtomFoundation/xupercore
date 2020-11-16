// 明确定义该模块需要的上下文信息，方便代码阅读和理解
package context

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
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
	GetConsensusStorage() []byte
	GetTimestamp() int64
	// 特定共识需要向raw block更新专有字段, eg. PoW挖矿时需更新nonce
	SetItem(string, interface{}) error
	// 使用PoW时需要调用该方法进行散列存在性证明
	MakeBlockId() []byte
	// 获取上一区块hash
	GetPreHash() []byte
	GetPubkey() []byte
	GetSign() []byte
}

// MetaInterface ATTENTION:此部分仅供单测使用，任何共识实例不应该调用
type MetaInterface interface {
	GetTrunkHeight() int64
	GetTipBlockid() []byte
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
	GetMeta() MetaInterface // ATTENTION:此部分仅供单测使用，任何共识实例不应该调用
	QueryBlock([]byte) (BlockInterface, error)
	QueryBlockByHeight(int64) (BlockInterface, error)
	QueryBlockHeader([]byte) (BlockInterface, error)
	GetTipSnapShot() FakeXMReader    // 获取当前最新快照
	GetGenesisConsensusConf() []byte // 获取账本创始块共识配置
	// GetSnapShotWithBlock([]byte) FakeXMReader // 原来utxoVM快照
	VerifyMerkle(BlockInterface) error // 用于验证merkel跟是否合法
}

// FakeXMReader
// TODO: 后续在此处更新ledger的XMReader接口定义, or合约中定义
type FakeXMReader interface {
	Get(bucket string, key []byte) ([]byte, error)
	Select(bucket string, startKey []byte, endKey []byte) error
}

// P2pCtxInConsensus 依赖p2p接口
type P2pCtxInConsensus interface {
	GetLocalAddress() string
	GetCurrentPeerAddress() []string
	// TODO: 接上network封装的两个func
	// SendMessage() error
	// SendMessageWithResponse() ([]byte, error)
}

// CryptoClientInConsensus 依赖加密接口
type CryptoClientInConsensus interface {
	// TODO: 接上密码库的func
	GetEcdsaPublicKeyFromJSON([]byte) ([]byte, error)
	VerifyAddressUsingPublicKey(string, []byte) (bool, uint8)
	VerifyECDSA([]byte, []byte, []byte) (bool, error)
	// GetEcdsaPrivateKeyFromJSON([]byte) ([]byte, error)
	// MakeVoteMsgSign() error
	// MakePhaseMsgSign() error
	// VerifyPhaseMsgSign() error
	// VerifyVoteMsgSign() error
}

// ConsensusCtx 共识领域级上下文
type ConsensusCtx struct {
	BcName string
	// 共识定义的账本接口
	Ledger       LedgerCtxInConsensus
	BCtx         xcontext.BaseCtx
	P2p          P2pCtxInConsensus
	CryptoClient CryptoClientInConsensus
}

// CreateConsensusCtx 创建共识上下文，外界调用
func CreateConsensusCtx(bcName string, ledger LedgerCtxInConsensus, p2p P2pCtxInConsensus,
	cryptoClient CryptoClientInConsensus, bCtx xcontext.BaseCtx) ConsensusCtx {
	return ConsensusCtx{
		BcName:       bcName,
		Ledger:       ledger,
		BCtx:         bCtx,
		P2p:          p2p,
		CryptoClient: cryptoClient,
	}
}

// 多定义一个height
type FakeKernMethod func(ctx kernel.KContext, height int64) error

type FakeRegistry interface {
	RegisterKernMethod(contract, method string, handler FakeKernMethod)
}
