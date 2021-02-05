package consensus

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

// ConsensusInterface 定义了一个共识实例需要实现的接口，用于kernel外的调用
type ConsensusInterface interface {
	// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
	CompeteMaster(height int64) (bool, bool, error)
	// CheckMinerMatch 当前block是否合法
	CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error)
	// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回truncate目标(如需裁剪), 返回写consensusStorage, 返回err
	ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error)
	// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
	CalculateBlock(block cctx.BlockInterface) error
	// ProcessConfirmBlock 用于确认块后进行相应的处理
	ProcessConfirmBlock(block cctx.BlockInterface) error
	// GetStatus 获取区块链共识信息
	GetConsensusStatus() (base.ConsensusStatus, error)
}
