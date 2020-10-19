package engines

import (
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/lib/logs"
)

// 为了简化引擎复杂度，采用执行引擎和信息读取组件分离的设计

// 执行引擎对外暴露运行环境上下文获取接口：GetEngineCtx、GetChainCtx
// 信息读取组件通过运行环境上下文获取相应组件句柄
// 由应用方按需注入到相应的信息读取组件完成读操作

// 考虑到可扩展性，PreExec和Submit相应结构采用了interface
// 由对应引擎提供对应数据结构的类型转换函数，供应用层选择使用

// 区块链执行引擎
type BCEngine interface {
	// 初始化引擎
	Init(logs.LogDriver, *EnvConfig) error
	// 启动引擎
	Start() error
	// 关闭引擎
	Stop()
	// 获取区块链引擎上下文
	GetEngineCtx() BCEngineCtx
	// 获取指定链上下文
	GetChainCtx(string) ChainCtx
	// 提交交易
	SubmitTx(xctx.ComOperateCtx, interface{}) (interface{}, error)
}
