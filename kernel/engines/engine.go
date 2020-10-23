package engines

import (
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
)

// 为了简化引擎复杂度，采用执行引擎和信息读取组件分离的设计

// 执行引擎对外暴露运行环境上下文获取接口：GetEngineCtx、GetChainCtx
// 信息读取组件通过运行环境上下文获取相应组件句柄
// 由应用方按需注入到相应的信息读取组件完成读操作

// 考虑到可扩展性，需要做到框架数据结构无关，SubmitTx交易结构采用了interface
// 由对应引擎提供对应数据结构的类型转换函数，供应用层选择使用

// 区块链执行引擎
type BCEngine interface {
	// 初始化引擎
	Init(*EnvConfig) error
	// 启动引擎
	Run()
	// 退出引擎
	Stop()
	// 获取区块链引擎上下文
	GetEngineCtx() BCEngineCtx
	// 获取指定链上下文
	GetChainCtx(string) ChainCtx
	// 提交交易
	SubmitTx(xctx.ComOpCtx, interface{}) (interface{}, error)
}

// 面向接口编程

// 定义引擎对网络组件依赖接口约束
type XNetwork interface {
}

// 定义引擎对账本组件依赖接口约束
// 由于账本并非通用实现，所以定义为空interface，使用时需要由具体引擎转义
// 具体引擎的utils包需要提供转义方法，由应用方根据需要选择调用
type XLedger interface {
}

// 定义引擎对共识组件依赖接口约束
type XConsensus interface {
}

// 定义引擎对合约组件依赖接口约束
type XContract interface {
}

// 定引擎义对权限组件依赖接口约束
type XPermission interface {
}

// 定义引擎对加密组件依赖接口约束
type XCrypto interface {
}
