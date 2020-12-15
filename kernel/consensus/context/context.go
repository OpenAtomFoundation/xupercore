// 明确定义该模块需要的上下文信息，方便代码阅读和理解
package context

import (
	"github.com/xuperchain/xupercore/kernel/common/xaddress"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/kernel/network"
	cryptoBase "github.com/xuperchain/xupercore/lib/crypto/client/base"
)

type BlockInterface ledger.BlockHandle
type Address xaddress.Address
type CryptoClient cryptoBase.CryptoClient
type P2pCtxInConsensus network.Network

// LedgerCtxInConsensus使用到的ledger接口
type LedgerRely interface {
	GetConsensusConf() ([]byte, error)
	QueryBlock(blkId []byte) (ledger.BlockHandle, error)
	QueryBlockByHeight(int64) (ledger.BlockHandle, error)
	GetTipBlock() ledger.BlockHandle
	GetTipXMSnapshotReader() (ledger.XMSnapshotReader, error)
	CreateSnapshot(blkId []byte) (ledger.XMReader, error)
	GetTipSnapshot() (ledger.XMReader, error)
}

// ConsensusCtx共识运行环境上下文
type ConsensusCtx struct {
	xctx.BaseCtx
	BcName   string
	Address  *Address
	Crypto   cryptoBase.CryptoClient
	Contract contract.Manager
	Ledger   LedgerRely
	Network  network.Network
}
