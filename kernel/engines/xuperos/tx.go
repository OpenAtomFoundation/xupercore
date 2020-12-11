// 交易处理
package xuperos

import (
	"context"
	"github.com/patrickmn/go-cache"
	"github.com/xuperchain/xuperchain/core/global"
	"github.com/xuperchain/xuperchain/core/pb"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	netPB "github.com/xuperchain/xupercore/kernel/network/pb"
	"github.com/xuperchain/xupercore/lib/logs"
	"time"
)

// 负责交易交易处理相关逻辑封装
type txProcessor struct {
	ctx *def.ChainCtx
	log logs.Logger

	state   def.XState
	handled *cache.Cache
	expired time.Duration
}

func NewTxProcessor(ctx *def.ChainCtx) *txProcessor {
	obj := &txProcessor{
		ctx: ctx,
		log: ctx.XLog,

		handled: cache.New(ctx.EngCfg.TxIdCacheExpiredTime, def.TxIdCacheGcTime),
	}

	return obj
}

// 验证交易
func (t *txProcessor) verifyTx(in *pb.Transaction) (bool, error) {
	txId := in.GetTxid()
	if txId == nil {
		return false, ErrTxIdNil
	}

	txIdStr := string(txId)
	if _, exist := t.handled.Get(txIdStr); exist {
		return false, ErrTxDuplicate
	}

	t.handled.Set(txIdStr, true, t.expired)
	return t.ctx.State.VerifyTx(in)
}

// 提交交易到状态机和未确认交易池
func (t *txProcessor) submitTx(in *pb.Transaction) error {
	err := t.ctx.State.DoTx(in)
	if err != nil && err != utxo.ErrAlreadyInUnconfirmed {
		t.handled.Delete(string(in.GetTxid()))
	}

	return err
}

// TODO: 由那个模块控制结束：state || chain
// 周期repost本地未上链的交易
func (t *txProcessor) repostOfflineTx() {
	state := t.ctx.State
	for txs := range state.GetOfflineTx() {
		header := &pb.Header{Logid: global.Glogid()}
		batch := &pb.BatchTxs{Header: header}
		for _, tx := range txs {
			if _, ok := state.HasTx(tx.Txid); !ok {
				continue //跳过已经unconfirmed的
			}
			txStatus := &pb.TxStatus{
				Header: header,
				Bcname: t.ctx.BCName,
				Txid:   tx.Txid,
				Tx:     tx,
			}
			batch.Txs = append(batch.Txs, txStatus)
		}

		t.log.Debug("repost batch tx list", "size", len(batch.Txs))

		opts := []p2p.MessageOption{
			p2p.WithBCName(t.ctx.BCName),
			p2p.WithLogId(header.GetLogid()),
		}
		msg := p2p.NewMessage(netPB.XuperMessage_BATCHPOSTTX, batch, opts...)
		go func() {
			err := t.ctx.Net.SendMessage(context.Background(), msg)
			if err != nil {
				t.log.Warn("broadcast offline tx error", "log_id", header.GetLogid(), "error", err)
			}
		}()
	}
}