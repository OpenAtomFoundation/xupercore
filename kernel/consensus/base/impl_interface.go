package base

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

// ConsensusInterface 定义了一个共识实例需要实现的接口，用于bcs具体共识的实现
type ConsensusImplInterface interface {
	// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
	CompeteMaster(height int64) (bool, bool, error)
	// CheckMinerMatch 查看block是否合法
	CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error)
	// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回truncate目标(如需裁剪), 返回写consensusStorage, 返回err
	ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error)
	// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
	CalculateBlock(block cctx.BlockInterface) error
	// ProcessConfirmBlock 用于确认块后进行相应的处理
	ProcessConfirmBlock(block cctx.BlockInterface) error
	// GetStatus 获取区块链共识信息
	GetConsensusStatus() (ConsensusStatus, error)

	// 共识实例的挂起逻辑, 另: 若共识实例发现绑定block结构有误，会直接停掉当前共识实例并panic
	Stop() error
	// 共识实例的启动逻辑
	Start() error
	// 共识占用blockinterface的专有存储，特定共识需要提供parse接口，在此作为接口高亮
	ParseConsensusStorage(block cctx.BlockInterface) (interface{}, error)
}

// tdpos的查询接口还是保持不变
// queryConsensusStatus

/* ConsensusStatus 定义了一个共识实例需要返回的各种状态，需特定共识实例实现相应接口
 */
type ConsensusStatus interface {
	// 获取共识版本号
	GetVersion() int64
	// pluggable consensus共识item起始高度
	GetConsensusBeginInfo() int64
	// 获取共识item所在consensus slice中的index
	GetStepConsensusIndex() int
	// 获取共识类型
	GetConsensusName() string
	// 获取当前状态机term
	GetCurrentTerm() int64
	// 获取当前矿工信息
	GetCurrentValidatorsInfo() []byte
}
