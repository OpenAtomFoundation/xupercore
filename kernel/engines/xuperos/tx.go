// 交易处理
package xuperos

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// 负责交易交易处理相关逻辑封装
type txProcessor struct {
	log    logs.Logger
	timer  *timer.XTimer
	ledger def.XLedger
}

func NewTxProcessor(ctx *def.ChainCtx) *txProcessor {
	obj := &txProcessor{
		log:    ctx.XLog,
		timer:  ctx.Timer,
		ledger: ctx.Ledger,
	}

	return obj
}

// 验证交易
func (t *txProcessor) verifyTx() (bool, error) {

}

// 提交交易到状态机和未确认交易池
func (t *txProcessor) submitTx() error {

}
