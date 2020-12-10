package agent

import (
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
)

type LedgerAgent struct {
	log      logs.Logger
	chainCtx common.ChainCtx
}

func NewLedgerAgent(chainCtx common.ChainCtx) *LedgerAgent {
	return &LedgerAgent{
		log:      chainCtx.GetXLog(),
		chainCtx: chainCtx,
	}
}

// 从创世块获取创建合约账户消耗gas
func (t *LedgerAgent) GetNewAccountGas() (int64, error) {
	return 0, nil
}

// 从创世块获取加密算法类型
func (t *LedgerAgent) GetCryptoType() (int, error) {
	return 0, nil
}

// 获取状态机最新确认快照
func (t *LedgerAgent) GetTipXMSnapshotReader() (ledger.XMSnapshotReader, error) {
	return nil, nil
}
