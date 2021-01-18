package reader

import (
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	consBase "github.com/xuperchain/xupercore/kernel/consensus/base"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
)

type ConsensusReader interface {
	// 获取共识状态
	GetConsStatus() (consBase.ConsensusStatus, error)
	// 共识特定共识类型的操作后续统一通过合约操作
	// tdpos目前已经提供的rpc接口，看是否有业务依赖
	// 视情况决定是不是需要继续支持，需要支持走代理合约调用
}

type consensusReader struct {
	chainCtx *common.ChainCtx
	baseCtx  xctx.XContext
	log      logs.Logger
}

func NewConsensusReader(chainCtx *common.ChainCtx, baseCtx xctx.XContext) ConsensusReader {
	if chainCtx == nil || baseCtx == nil {
		return nil
	}

	reader := &consensusReader{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
		log:      baseCtx.GetLog(),
	}

	return reader
}

func (t *consensusReader) GetConsStatus() (consBase.ConsensusStatus, error) {
	cons, _ := t.chainCtx.Consensus.GetConsensusStatus()
	return cons, nil
}
