package xuper

import (
	"fmt"
	"sync"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines"
	"github.com/xuperchain/xupercore/lib/logs"
)

// xuperchain执行引擎
type XuperEngine struct {
	// 引擎级上下文
	sysCtx engines.BCEngineCtx
	// 链实例
	chains sync.Map
}

func NewXuperEngine() *XuperEngine {
	return &XuperEngine{}
}

// 向工厂注册自己的创建方法
func init() {
	blockchain.Register(BCEngineName, NewXuperEngine)
}

// 初始化执行引擎环境上下文
func (t *XuperEngine) Init(lg logs.Logger, envCfg *engines.EnvConfig) error {
	// 初始化引擎运行上下文

	// 启动P2P网络

	// 加载区块链

	// 注册P2P网络事件订阅

}

// 启动执行引擎
func (t *XuperEngine) Start() error {
	// 启动矿工

	// 启动定时任务

	// 启动P2P网络事件消费
}

// 关闭执行引擎
func (t *XuperEngine) Stop() {
	// 关闭P2Pw网络

	// 关闭定时任务

	// 关闭矿工
}

// 获取执行引擎环境
func (t *XuperEngine) GetEngineCtx() sysctx.BCEngineCtx {

}

// 获取对应链执行环境
func (t *XuperEngine) GetChainCtx(chainName string) sysctx.ChainCtx {

}

// 发起交易提交操作
func (t *XuperEngine) SubmitTx(ctx xctx.ComOperateCtx, req interface{}) (interface{}, error) {
	// 转换消息类型
	transSubmitReq(req)

	// 获取对应链实例

	// 通过对应链实例提交交易
}

// 注册并启动链
func (t *XuperEngine) RegisterChain(chainName string, chainObj *XuperChain) error {

}

// 关闭并卸载链
func (t *XuperEngine) UnloadChain(chainName string) error {

}

// 订阅网络事件
func (t *XuperEngine) SubscriberNetEvent() error {

}

// 处理提交交易网络事件
func (t *XuperEngine) handleSubmitTx() {

}

// 处理批量提交交易网络事件
func (t *XuperEngine) handleBatchSubmitTx() {

}

// 处理接收到新块网络事件
func (t *XuperEngine) handleRecvBlock() {

}

// 处理收到新区块ID网络事件
func (t *XuperEngine) handleRecvBlockID() {

}

// 处理查询区块信息网络事件
func (t *XuperEngine) handleQueryBlock() {

}

// 处理查询区块链状态网络事件
func (t *XuperEngine) handleQueryChainStatus() {

}

// 处理确认区块链状态网络事件
func (t *XuperEngine) handleConfirmChainStatus() {

}

// 处理查询节点rpc端口网络事件
func (t *XuperEngine) handleQueryRPCPort() {

}
