// 明确定义该模块需要的上下文信息，方便代码阅读和理解
package context

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
)

type ConsensusComponent int

const (
	Empty      ConsensusComponent = iota //无
	ChainedBFT                           // 使用了chained-bft组件
)

type BlockInterface interface {
	GetProposer() string
	GetHeight() int64
	GetBlockid() []byte
	GetConsensusStorage() []byte // 不能再叫GetJustify了
	GetTimestamp() int64
}

type MetaInterface interface {
	GetTrunkHeight() int64
	GetTipBlockid() []byte
}

// 特定共识的字段标示
type ConsensusConfig struct {
	// 获取本次共识的类型名称
	ConsensusName string `json:"name"`
	// 获取本次共识专属属性
	Config string `json:"config"`
	// 获取本次共识的起始Blockid，即起始高度的上一区块blockid
	BeginBlockid []byte `json:"-"`
}

// 使用到的ledger接口
type LedgerCtxInConsensus interface {
	GetMeta() MetaInterface
	QueryBlock([]byte) BlockInterface
	QueryBlockByHeight(int64) BlockInterface
	Truncate() error
	GetConsensusConf() []byte
	GetGenesisBlock() BlockInterface
}

type P2pCtxInConsensus interface {
	GetLocalAddress() string
	GetCurrentPeerAddress() []string
	// TODO: 接上network封装的两个func
	// SendMessage() error
	// SendMessageWithResponse() ([]byte, error)
}

type CryptoClientInConsensus interface {
	// TODO: 接上密码库的func
	// GetEcdsaPrivateKeyFromJSON([]byte) ([]byte, error)
	// MakeVoteMsgSign() error
	// MakePhaseMsgSign() error
	// VerifyPhaseMsgSign() error
	// VerifyVoteMsgSign() error
}

// 共识领域级上下文
type ConsensusCtx struct {
	BcName       string
	Ledger       LedgerCtxInConsensus
	BCtx         xcontext.BaseCtx
	P2p          P2pCtxInConsensus
	CryptoClient CryptoClientInConsensus
}

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

// kernel.KContext的fake定义
// TODO: 接上合约的func
type FakeKContext interface {
	Arg() []byte
}
