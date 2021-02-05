package reader

import (
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/xpb"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

type LedgerReader interface {
	// 查询交易信息（QueryTx）
	QueryTx(txId []byte) (*xpb.TxInfo, error)
	// 查询区块ID信息（GetBlock）
	QueryBlock(blkId []byte, needContent bool) (*xpb.BlockInfo, error)
	// 通过区块高度查询区块信息（GetBlockByHeight）
	QueryBlockByHeight(height int64, needContent bool) (*xpb.BlockInfo, error)
}

type ledgerReader struct {
	chainCtx *common.ChainCtx
	baseCtx  xctx.XContext
	log      logs.Logger
}

func NewLedgerReader(chainCtx *common.ChainCtx, baseCtx xctx.XContext) LedgerReader {
	if chainCtx == nil || baseCtx == nil {
		return nil
	}

	reader := &ledgerReader{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
		log:      baseCtx.GetLog(),
	}

	return reader
}

func (t *ledgerReader) QueryTx(txId []byte) (*xpb.TxInfo, error) {
	out := &xpb.TxInfo{}
	tx, err := t.chainCtx.Ledger.QueryTransaction(txId)
	if err != nil {
		t.log.Warn("ledger query tx error", "txId", utils.F(txId), "error", err)
		out.Status = lpb.TransactionStatus_TX_NOEXIST
		if err == ledger.ErrTxNotFound {
			// 查询unconfirmed表
			tx, _, err = t.chainCtx.State.QueryTx(txId)
			if err != nil {
				t.log.Warn("state query tx error", "txId", utils.F(txId), "error", err)
				return nil, common.ErrTxNotExist
			}
			t.log.Debug("state query tx succeeded", "txId", utils.F(txId))
			out.Status = lpb.TransactionStatus_TX_UNCONFIRM
			out.Tx = tx
			return out, nil
		}

		return nil, common.ErrTxNotExist
	}

	// 查询block状态，是否被分叉
	block, err := t.chainCtx.Ledger.QueryBlockHeader(tx.Blockid)
	if err != nil {
		t.log.Warn("query block error", "txId", utils.F(txId), "blockId", utils.F(tx.Blockid), "error", err)
		return nil, common.ErrBlockNotExist
	}

	t.log.Debug("query block succeeded", "txId", utils.F(txId), "blockId", utils.F(tx.Blockid))
	meta := t.chainCtx.Ledger.GetMeta()
	out.Tx = tx
	if block.InTrunk {
		out.Distance = meta.TrunkHeight - block.Height
		out.Status = lpb.TransactionStatus_TX_CONFIRM
	} else {
		out.Status = lpb.TransactionStatus_TX_FURCATION
	}

	return out, nil
}

// 注意不需要交易内容的时候不要查询
func (t *ledgerReader) QueryBlock(blkId []byte, needContent bool) (*xpb.BlockInfo, error) {
	out := &xpb.BlockInfo{}
	block, err := t.chainCtx.Ledger.QueryBlock(blkId)
	if err != nil {
		if err == ledger.ErrBlockNotExist {
			out.Status = lpb.BlockStatus_BLOCK_NOEXIST
			return out, common.ErrBlockNotExist
		}

		t.log.Warn("query block error", "err", err)
		return nil, common.ErrBlockNotExist
	}

	if needContent {
		out.Block = block
	}

	if block.InTrunk {
		out.Status = lpb.BlockStatus_BLOCK_TRUNK
	} else {
		out.Status = lpb.BlockStatus_BLOCK_BRANCH
	}

	return out, nil
}

// 注意不需要交易内容的时候不要查询
func (t *ledgerReader) QueryBlockByHeight(height int64, needContent bool) (*xpb.BlockInfo, error) {
	out := &xpb.BlockInfo{}
	block, err := t.chainCtx.Ledger.QueryBlockByHeight(height)
	if err != nil {
		if err == ledger.ErrBlockNotExist {
			out.Status = lpb.BlockStatus_BLOCK_NOEXIST
			return out, nil
		}

		t.log.Warn("query block by height error", "err", err)
		return nil, common.ErrBlockNotExist
	}

	if needContent {
		out.Block = block
	}

	if block.InTrunk {
		out.Status = lpb.BlockStatus_BLOCK_TRUNK
	} else {
		out.Status = lpb.BlockStatus_BLOCK_BRANCH
	}

	return out, nil
}
