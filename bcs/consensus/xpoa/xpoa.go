package xpoa

import (
	"github.com/xuperchain/xupercore/kernel/consensus"
	cbase "github.com/xuperchain/xupercore/kernel/consensus/base"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

func init() {
	consensus.Register("xpoa", NewXpoaConsensus)
}

type XpoaConsensus struct {
}

type XpoaStatus struct {
}

func (xs *XpoaStatus) GetVersion() int64 {
	return 0
}
func (xs *XpoaStatus) GetStepConsensusInfo() {

}
func (xs *XpoaStatus) GetCurrentConsensusInfo() {

}

// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
func (xc *XpoaConsensus) CompeteMaster(height int64) (bool, bool, error) {
	return true, true, nil
}

// CheckMinerMatch 当前block是否合法
func (xc *XpoaConsensus) CheckMinerMatch(consensusCtx cctx.ConsensusOperateCtx, block cctx.BlockInterface) (bool, error) {
	return true, nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理
func (xc *XpoaConsensus) ProcessBeforeMiner(timestamp int64) (map[string]interface{}, bool, error) {
	return nil, true, nil
}

// ProcessConfirmBlock 用于确认块后进行相应的处理
func (xc *XpoaConsensus) ProcessConfirmBlock(block cctx.BlockInterface) error {
	return nil
}

// GetStatus 获取区块链共识信息
func (xc *XpoaConsensus) GetConsensusStatus() (cbase.ConsensusStatus, error) {
	return &XpoaStatus{}, nil
}

func (xc *XpoaConsensus) KernMethodRegister() []func() {
	return nil
}

func NewXpoaConsensus(cCtx cctx.ConsensusCtx, cCfg cctx.ConsensusConfig) cbase.ConsensusImplInterface {
	return &XpoaConsensus{}
}
