package def

import "time"

// 引擎常量配置
const (
	// 引擎名
	BCEngineName = "xuperos"
	// 主链
	RootChain = "xuper"
	// 区块链配置
	BlockChainConfig = "xuper.json"
	// 节点模式
	NodeModeFastSync = "FastSync"
)

// 矿工状态
const (
	// EngineSafeModel 表示安全的同步
	MinerSafeModel = iota
	// EngineNormal 表示正常状态
	MinerNormal
)

// 广播模式
const (
	FullBroadCastMode = iota
	InteractiveBroadCastMode
	MixedBroadCastMode
)

// 交易
const (
	TxIdCacheGcTime = 180 * time.Second
)
