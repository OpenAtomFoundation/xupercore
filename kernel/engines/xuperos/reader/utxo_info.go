package reader

import (
	"math/big"

	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
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
	// 选择合适金额的utxo
	SelectUTXO(account string, need *big.Int, isLock, isExclude bool) (*lpb.UtxoOutput, error)
	// 按最大交易大小选择utxo
	SelectUTXOBySize(account string, isLock, isExclude bool) (*lpb.UtxoOutput, error)
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

func (t *utxoReader) GetBalance(address string) (string, error) {
	balance, err := t.chainCtx.State.GetBalance(address)
	if err != nil {
		t.log.Warn("get balance error", "err", err)
		return "", common.CastError(err)
	}

	return balance.String(), nil
}

func (t *utxoReader) GetFrozenBalance(account string) (string, error) {
	balance, err := t.chainCtx.State.GetFrozenBalance(account)
	if err != nil {
		t.log.Warn("get frozen balance error", "err", err)
		return "", common.CastError(err)
	}

	return balance.String(), nil
}

func (t *utxoReader) GetBalanceDetail(account string) ([]*lpb.BalanceDetailInfo, error) {
	tokenDetails, err := t.chainCtx.State.GetBalanceDetail(account)
	if err != nil {
		t.log.Warn("get balance detail error", "err", err)
		return nil, common.CastError(err)
	}

	return tokenDetails, nil
}

func (t *utxoReader) QueryUtxoRecord(account string, count int64) (*lpb.UtxoRecordDetail, error) {
	utxoRecord, err := t.chainCtx.State.QueryUtxoRecord(account, count)
	if err != nil {
		t.log.Warn("get utxo record error", "err", err)
		return nil, common.CastError(err)
	}

	return utxoRecord, nil
}

func (t *utxoReader) SelectUTXO(account string,
	need *big.Int, isLock, isExclude bool) (*lpb.UtxoOutput, error) {

	utxos, _, totalSelected, err := t.chainCtx.State.SelectUtxos(account, need, isLock, isExclude)
	if err != nil {
		t.log.Warn("failed to select utxo", "err", err)
		return nil, common.CastError(err)
	}

	utxoList := make([]*lpb.Utxo, 0, len(utxos))
	for _, v := range utxos {
		utxo := &lpb.Utxo{}
		utxo.RefTxid = v.RefTxid
		utxo.Amount = v.Amount
		utxo.RefOffset = v.RefOffset
		utxo.ToAddr = v.FromAddr
		utxoList = append(utxoList, utxo)
		t.log.Trace("Select utxo list", "refTxid", utils.F(v.RefTxid), "refOffset", v.RefOffset, "amount", new(big.Int).SetBytes(v.Amount).String())
	}

	out := &lpb.UtxoOutput{
		UtxoList:      utxoList,
		TotalSelected: totalSelected.String(),
	}
	return out, nil
}

func (t *utxoReader) SelectUTXOBySize(account string, isLock, isExclude bool) (*lpb.UtxoOutput, error) {
	utxos, _, totalSelected, err := t.chainCtx.State.SelectUtxosBySize(account, isLock, isExclude)
	if err != nil {
		t.log.Warn("failed to select utxo", "err", err)
		return nil, common.CastError(err)
	}

	utxoList := make([]*lpb.Utxo, 0, len(utxos))
	for _, v := range utxos {
		utxo := &lpb.Utxo{}
		utxo.RefTxid = v.RefTxid
		utxo.Amount = v.Amount
		utxo.RefOffset = v.RefOffset
		utxo.ToAddr = v.FromAddr
		utxoList = append(utxoList, utxo)
		t.log.Trace("Select utxo list", "refTxid", utils.F(v.RefTxid), "refOffset", v.RefOffset, "amount", new(big.Int).SetBytes(v.Amount).String())
	}

	out := &lpb.UtxoOutput{
		UtxoList:      utxoList,
		TotalSelected: totalSelected.String(),
	}
	return out, nil
}
