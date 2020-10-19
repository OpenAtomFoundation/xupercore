package xcontext

import (
	"github.com/xuperchain/xupercore/lib/timer"
)

// 通用操作级上下文，有特殊需要的自己定义
// 不是所有的方法都需要，按需选用
type ComOperateCtx interface {
	BaseCtx
	GetTimer() *timer.XTimer
	IsVaild() bool
}

type ComOperateCtxImpl struct {
	BaseCtxImpl
	// 便于记录各阶段处理耗时
	Timer *timer.XTimer
}

func CreateComOperateCtx(xlog logs.Logger, tmr *timer.XTimer) (ComOperateCtx, error) {
	if xlog == nil || tmr == nil {
		return nil, fmt.Errorf("create operate context failed because some param are missing")
	}

	ctx := new(ComOperateCtxImpl)
	ctx.XLog = xlog
	ctx.Timer = tmr

	return ctx, nil
}

func (t *ComOperateCtxImpl) GetTimer() *timer.XTimer {
	return t.Timer
}

func (t *ComOperateCtxImpl) IsVaild() bool {
	if t.XLog == nil || t.Timer == nil {
		return false
	}

	return true
}
