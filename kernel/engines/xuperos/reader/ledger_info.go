package reader

import (
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
)

type LedgerReader interface {
	// 查询交易信息（QueryTx）
	QueryTx(txId []byte) (*lpb.TxInfo, *common.Error)
	// 查询区块ID信息（GetBlock）
	QueryBlock(blkId []byte, needContent bool) (*lpb.BlockInfo, *common.Error)
	// 通过区块高度查询区块信息（GetBlockByHeight）
	QueryBlockByHeight(height int64, needContent bool) (*lpb.BlockInfo, *common.Error)
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

func (t *ledgerReader) QueryTx(txId []byte) (*TxInfo, *common.Error) {
	return nil, nil
}

// 注意不需要交易内容的时候不要查询
func (t *ledgerReader) QueryBlock(blkId []byte, needContent bool) (*BlockInfo, *common.Error) {
	return nil, nil
}

// 注意不需要交易内容的时候不要查询
func (t *ledgerReader) QueryBlockByHeight(height int64, needContent bool) (*BlockInfo, *common.Error) {
	return nil, nil
}
