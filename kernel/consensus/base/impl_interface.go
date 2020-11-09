package base

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

// ConsensusInterface 定义了一个共识实例需要实现的接口
type ConsensusImplInterface interface {
	// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
	CompeteMaster(height int64) (bool, bool, error)
	// CheckMinerMatch 查看block是否合法
	CheckMinerMatch(ctx xcontext.BaseCtx, block cctx.BlockInterface) (bool, error)
	// ProcessBeforeMiner 开始挖矿前进行相应的处理
	ProcessBeforeMiner(timestamp int64) (map[string]interface{}, bool, error)
	// ProcessConfirmBlock 用于确认块后进行相应的处理
	ProcessConfirmBlock(block cctx.BlockInterface) error
	// GetStatus 获取区块链共识信息
	GetConsensusStatus() (ConsensusStatus, error)
	// 共识实例的挂起逻辑
	Stop() error
}

/* ConsensusStatus 定义了一个共识实例需要返回的各种状态，需特定共识实例实现相应接口
 */
type ConsensusStatus interface {
	GetVersion() int64
	// pluggable consensus共识item起始上一blockid
	GetConsensusBeginInfo() []byte
	// 获取共识item所在consensus slice中的index
	GetStepConsensusIndex() int64
	// 获取共识类型
	GetConsensusType() int
	// 获取当前状态机term
	GetCurrentTerm() int64
	// 获取当前矿工信息
	GetCurrentValidatorsInfo() []byte
}
