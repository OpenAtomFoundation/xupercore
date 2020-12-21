package reader

import (
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
	ctx *common.ChainCtx
	log logs.Logger
}

func NewConsensusReader(ctx *common.ChainCtx) ConsensusReader {
	if ctx == nil {
		return nil
	}

	reader := &consensusReader{
		ctx: ctx,
		log: ctx.GetLog(),
	}

	return reader
}

func (t *consensusReader) GetConsStatus() (consBase.ConsensusStatus, error) {
	return t.ctx.Consensus.GetConsensusStatus()
}
