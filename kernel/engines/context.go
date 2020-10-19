// 统一管理系统全局上下文
package engines

import (
	"sync"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/lib/logs"
)

// 区块链执行引擎级context，维护区块链运行环境
type BCEngineCtx interface {
	xcontext.BaseCtx
}

type BCEngineCtxImpl struct {
	// 基础上下文
	xcontext.BaseCtxImpl
	// 日志驱动
	logDrv logs.LogDriver
	// 系统级配置
	sysCfg *SysConfig
	// 网络
	net network.Network
	// 加密插件
	// crypto
}

func (t *BCEngineCtxImpl) GetSysConf() {

}

func (t *BCEngineCtxImpl) GetNet() {

}

// 链级别上下文，维护链级别上下文，每条平行链各有一个
type ChainCtx interface {
	xcontext.BaseCtx
}

type ChainCtxImpl struct {
	// 基础上下文
	xcontext.BaseCtxImpl
	// 链名
	bcname string
	// 账本，由于账本并非通用实现，所以定义为interface，使用时由具体引擎转义
	// 引擎utils包需要提供转义方法，由应用方根据需要选择调用
	ledger interface{}
	// 共识
	conse consensus.Consensus
	// 合约
	contract
}

func CreateChainCtx(ledger ledger.Ledger, conse consensus.Consensus) (ChainCtx, error) {

}

func GetBCName() string {

}

func GetLedger() ledger.Ledger {

}

func GetConse() consensus.Consensus {

}
