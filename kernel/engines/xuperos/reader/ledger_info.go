package reader

import (
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/pb"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
)

type TxInfo struct {
	Status   lpb.TransactionStatus
	Distance int64
	Tx       *lpb.Transaction
}

type BlockInfo struct {
	Status lpb.BlockStatus
	Block  *lpb.InternalBlock
}

type LedgerReader interface {
	// 查询交易信息（QueryTx）
	QueryTx(txId []byte) (*TxInfo, error)
	// 查询区块ID信息（GetBlock）
	QueryBlock(blkId []byte, needContent bool) (*BlockInfo, error)
	// 通过区块高度查询区块信息（GetBlockByHeight）
	QueryBlockByHeight(height int64, needContent bool) (*BlockInfo, error)
}

type ledgerReader struct {
	ctx *common.ChainCtx
	log logs.Logger
}

func NewLedgerReader(ctx *common.ChainCtx) LedgerReader {
	if ctx == nil {
		return nil
	}

	reader := &ledgerReader{
		ctx: ctx,
		log: ctx.GetLog(),
	}

	return reader
}

func (t *ledgerReader) QueryTx(txId []byte) (*TxInfo, error) {
	return nil, nil
}

func (t *ledgerReader) QueryBlock(blkId []byte, needContent bool) (*BlockInfo, error) {
	return nil, nil
}

func (t *ledgerReader) QueryBlockByHeight(height int64, needContent bool) (*BlockInfo, error) {
	return nil, nil
}
