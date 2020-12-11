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
	// 完全块广播模式，即直接广播原始块给所有相邻节点
	FullBroadCastMode = iota
	// 问询式块广播模式，即先广播新块的头部给相邻节点
	// 邻节点在没有相同块的情况下通过GetBlock主动获取块数据
	InteractiveBroadCastMode
	// 出块节点将新块用Full_BroadCast_Mode模式广播
	// 其他节点使用Interactive_BroadCast_Mode模式广播区块
	MixedBroadCastMode
)

// 交易
const (
	TxIdCacheGcTime = 180 * time.Second
)
