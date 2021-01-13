package agent

import (
	"encoding/json"
	"fmt"
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
	amount := t.chainCtx.Ledger.GenesisBlock.GetConfig().GetNewAccountResourceAmount()
	return amount, nil
}

// 从创世块获取加密算法类型
func (t *LedgerAgent) GetCryptoType() (string, error) {
	cryptoType := t.chainCtx.Ledger.GenesisBlock.GetConfig().GetCryptoType()
	return cryptoType, nil
}

// 从创世块获取共识配置
func (t *LedgerAgent) GetConsensusConf() ([]byte, error) {
	consensusConf := t.chainCtx.Ledger.GenesisBlock.GetConfig().GenesisConsensus
	data, err := json.Marshal(consensusConf)
	if err != nil {
		return nil, fmt.Errorf("marshal consensus conf error: %s", err)
	}
	return data, nil
}

// 查询区块
func (t *LedgerAgent) QueryBlock(blkId []byte) (ledger.BlockHandle, error) {
	block, err := t.chainCtx.Ledger.QueryBlock(blkId)
	if err != nil {
		return nil, err
	}

	return NewBlockAgent(block), nil
}

func (t *LedgerAgent) QueryBlockByHeight(height int64) (ledger.BlockHandle, error) {
	block, err := t.chainCtx.Ledger.QueryBlockByHeight(height)
	if err != nil {
		return nil, err
	}

	return NewBlockAgent(block), nil
}

func (t *LedgerAgent) GetTipBlock() ledger.BlockHandle {
	meta := t.chainCtx.Ledger.GetMeta()
	blkAgent, _ := t.QueryBlock(meta.TipBlockid)
	return blkAgent
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
