// 统一定义状态机对外暴露功能
package state

import (
	"fmt"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/lib/logs"
)

type XuperState struct {
	lctx *def.LedgerCtx
	log  logs.Logger
}

func NewXuperState(lctx *def.LedgerCtx) (*XuperState, error) {
	if lctx == nil {
		return nil, fmt.Errrof("create ledger failed because context set error")
	}

	obj := &XuperState{
		lctx: lctx,
		log:  lctx.XLog,
	}

	return obj, nil
}

// 选择足够金额的utxo
func (t *XuperState) SelectUtxos() {

}

// 获取一批未确认交易（用于矿工打包区块）
func (t *XuperState) GetUnconfirmedTx() {

}

// 校验交易
func (t *XuperState) VerifyTx() {

}

// 执行交易
func (t *XuperState) DoTx() {

}

// 执行区块
func (t *XuperState) Play() {

}

// 回滚全部未确认交易
func (t *XuperState) RollBackUnconfirmedTx() {

}

// 同步账本和状态机
func (t *XuperState) Walk() {

}

// 获取状态机tip block id
func (t *XuperState) GetTipBlockid() {

}

// 查询交易
func (t *XuperState) QueryTx() {

}

// 查询账余额
func (t *XuperState) GetBalance() {

}

// 查找状态机meta信息
func (t *XuperState) GetMeta() {

}
