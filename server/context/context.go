package context

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/server/common"
)

// 请求级别上下文
type ReqCtx interface {
	GetEngine() def.Engine
	GetLog() logs.Logger
	GetTimer() *timer.XTimer
	GetClientIp() string
}

type ReqCtxImpl struct {
	engine   def.Engine
	log      logs.Logger
	timer    *timer.XTimer
	clientIp string
}

func NewReqCtx(engine def.Engine, reqId, clientIp string) (ReqCtx, error) {
	if engine == nil {
		return nil, fmt.Errorf("new request context failed because engine is nil")
	}

	log, err := logs.NewLogger(reqId, common.SubModName)
	if err != nil {
		return nil, fmt.Errorf("new request context failed because new logger failed.err:%s", err)
	}

	ctx := &ReqCtxImpl{
		engine:   engine,
		log:      log,
		timer:    timer.NewXTimer(),
		clientIp: clientIp,
	}

	return ctx, nil
}

func (t *ReqCtxImpl) GetEngine() def.Engine {
	return t.engine
}

func (t *ReqCtxImpl) GetLog() logs.Logger {
	return t.log
}

func (t *ReqCtxImpl) GetTimer() *timer.XTimer {
	return t.timer
}

func (t *ReqCtxImpl) GetClientIp() string {
	return t.clientIp
}
