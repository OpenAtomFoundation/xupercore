// 明确定义该模块需要的上下文信息，方便代码阅读和理解
package context

import (
	"context"
	"fmt"
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/network/config"
	"github.com/xuperchain/xupercore/lib/logs"
)

// 考虑到有些对象是有状态的，需要单例实现
// 有些上下文是领域级别的，有些是操作级别的
// 所以对领域级别的上下文和操作级别的上下文分别定义

// 领域级上下文
type DomainCtx interface {
	context.Context

	GetLog() logs.Logger
	GetP2PConf() *config.Config
	GetMetricSwitch() bool
	SetMetricSwitch(s bool)
	IsValid() bool
}

type DomainCtxImpl struct {
	xcontext.BaseCtx
	P2PConf      *config.Config
	MetricSwitch bool
}

// 必须设置的在参数直接指定，可选的通过对应的Set方法设置
func CreateDomainCtx(confPath string) (DomainCtx, error) {
	if confPath == "" {
		return nil, fmt.Errorf("create domain context failed because some param are missing")
	}

	// 加载配置
	cfg, err := config.LoadP2PConf(confPath)
	if err != nil {
		return nil, fmt.Errorf("create object context failed because config load fail.err:%v", err)
	}

	log, err := logs.NewLogger("", "network")
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because new logger error. err:%v", err)
	}

	ctx := new(DomainCtxImpl)
	ctx.XLog = log
	ctx.P2PConf = cfg
	// 可选参数设置默认值
	ctx.MetricSwitch = false

	return ctx, nil
}

func (t *DomainCtxImpl) GetLog() logs.Logger {
	return t.XLog
}

func (t *DomainCtxImpl) GetP2PConf() *config.Config {
	return t.P2PConf
}

func (t *DomainCtxImpl) GetMetricSwitch() bool {
	return t.MetricSwitch
}

func (t *DomainCtxImpl) SetMetricSwitch(s bool) {
	t.MetricSwitch = s
}

func (t *DomainCtxImpl) IsValid() bool {
	if t.XLog == nil || t.P2PConf == nil {
		return false
	}

	return true
}
