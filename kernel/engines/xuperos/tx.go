// 交易处理
package xuperos

import (
	"github.com/patrickmn/go-cache"
	"github.com/xuperchain/xuperchain/core/pb"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
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
