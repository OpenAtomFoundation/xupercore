package xcontext

import (
	"fmt"

	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// 通用操作级上下文，不需要扩展的场景直接选用
// 有特殊需要扩展的自行定义。不是所有的方法都需要，按需选用
type ComOpCtx interface {
	BaseCtx
	GetTimer() *timer.XTimer
	IsVaild() bool
}

type ComOpCtxImpl struct {
	BaseCtxImpl
	// 便于记录各阶段处理耗时
	Timer *timer.XTimer
}

func CreateComOpCtx(xlog logs.Logger, tmr *timer.XTimer) (ComOpCtx, error) {
	if xlog == nil || tmr == nil {
		return nil, fmt.Errorf("create operate context failed because some param are missing")
	}

	ctx := new(ComOpCtx)
	ctx.XLog = xlog
	ctx.Timer = tmr

	return ctx, nil
}

func (t *ComOpCtx) GetTimer() *timer.XTimer {
	return t.Timer
}

func (t *ComOpCtx) IsVaild() bool {
	if t.XLog == nil || t.Timer == nil {
		return false
	}

	return true
}
