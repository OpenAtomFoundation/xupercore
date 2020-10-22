// 定义公共上下文结构，明确定义上下文结构，方便代码阅读
package xcontext

import (
	"context"
	"time"

	"github.com/xuperchain/xupercore/lib/logs"
)

// 考虑到后续对操作生命周期做控制，先实现空context.Context接口，方便扩展
// 同时定义扩展全局需要的公共成员，方便为各对象统一注入和管理
type BaseCtx interface {
	context.Context
	GetLog() logs.Logger
}

// 提供基础上下文实现，供其他领域组合扩展
type BaseCtxImpl struct {
	XLog logs.Logger
}

func Background(xlog logs.Logger) BaseCtx {
	return &BaseCtxImpl{xlog}
}

func (t *BaseCtxImpl) GetLog() logs.Logger {
	return t.XLog
}

func (t *BaseCtxImpl) Deadline() (deadline time.Time, ok bool) {
	return
}

func (t *BaseCtxImpl) Done() <-chan struct{} {
	return nil
}

func (t *BaseCtxImpl) Err() error {
	return nil
}

func (t *BaseCtxImpl) Value(key interface{}) interface{} {
	return nil
}
