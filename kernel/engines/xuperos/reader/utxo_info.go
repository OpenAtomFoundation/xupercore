package reader

import (
	"github.com/xuperchain/xuperchain/core/global"
	"github.com/xuperchain/xuperchain/core/pb"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
)

type UtxoReader interface {
	// 获取账户余额
	GetBalance(account string) (string, error)
	// 获取账户冻结余额
	GetFrozenBalance(account string) (string, error)
	// 获取账户余额详情
	GetBalanceDetail(account string) ([]*lpb.BalanceDetailInfo, error)
	// 拉取固定数目的utxo
	QueryUtxoRecord(account string, count int64) (*lpb.UtxoRecordDetail, error)
	// 按最大交易大小选择utxo
	SelectUTXOBySize(account string, isLock, isExclude bool) (*lpb.UtxoOutput, error)
	// 选择合适金额的utxo
	SelectUTXO(account string, need *big.Int, isLock, isExclude bool) (*lpb.UtxoOutput, error)
}

type utxoReader struct {
	chainCtx *common.ChainCtx
	baseCtx  xctx.XContext
	log      logs.Logger
}

func NewUtxoReader(chainCtx *common.ChainCtx, baseCtx xctx.XContext) UtxoReader {
	reader := &utxoReader{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
		log:      baseCtx.GetLog(),
	}

	return reader
}
