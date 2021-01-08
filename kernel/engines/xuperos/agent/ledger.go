package agent

import (
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/logs"
)

type LedgerAgent struct {
	log      logs.Logger
	chainCtx *common.ChainCtx
}

func NewLedgerAgent(chainCtx *common.ChainCtx) *LedgerAgent {
	return &LedgerAgent{
		log:      chainCtx.GetLog(),
		chainCtx: chainCtx,
	}
}

// 从创世块获取创建合约账户消耗gas
func (t *LedgerAgent) GetNewAccountGas() (int64, error) {
	return 0, nil
}

// 从创世块获取加密算法类型
func (t *LedgerAgent) GetCryptoType() (string, error) {
	return "", nil
}

// 从创世块获取共识配置
func (t *LedgerAgent) GetConsensusConf() ([]byte, error) {
	return nil, nil
}

// 查询区块
func (t *LedgerAgent) QueryBlock(blkId []byte) (ledger.BlockHandle, error) {
	return nil, nil
}

func (t *LedgerAgent) QueryBlockByHeight(int64) (ledger.BlockHandle, error) {
	return nil, nil
}

func (t *LedgerAgent) GetTipBlock() ledger.BlockHandle {
	return nil
}

// 获取状态机最新确认高度快照（只有Get方法，直接返回[]byte）
func (t *LedgerAgent) GetTipXMSnapshotReader() (ledger.XMSnapshotReader, error) {
	return nil, nil
}

// 根据指定blockid创建快照（Select方法不可用）
func (t *LedgerAgent) CreateSnapshot(blkId []byte) (ledger.XMReader, error) {
	return nil, nil
}

// 获取最新确认高度快照（Select方法不可用）
func (t *LedgerAgent) GetTipSnapshot() (ledger.XMReader, error) {
	return nil, nil
}

// 获取最新状态数据
func (t *LedgerAgent) CreateXMReader() (ledger.XMReader, error) {
	return nil, nil
}
