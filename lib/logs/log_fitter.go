package logs

import (
	"fmt"
	"os"
	"sync"
)

// Reserve common key
const (
	CommFieldLogId = "log_id"
	CommFieldPid   = "pid"
	CommFieldCall  = "call"
)

const (
	DefaultCallDepth = 4
)

// 底层日志库约束接口
type LogDriver interface {
	Error(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Trace(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
}

// 在日志库之上做一层轻量级封装，方便日志字段组装和日志库替换
// 对外提供功能接口
type Logger interface {
	GetLogId() string
	SetCommField(key string, value interface{})
	SetInfoField(key string, value interface{})
	Error(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Trace(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
}

// Logger Fitter
type LogFitter struct {
	logger       LogDriver
	logId        string
	pid          int
	commFields   []interface{}
	commFieldLck *sync.RWMutex
	infoFields   []interface{}
	infoFieldLck *sync.RWMutex
	callDepth    int
}

func NewLogger(logger LogDriver, logId string) (*LogFitter, error) {
	if logger == nil {
		return nil, fmt.Errorf("new logger param error")
	}
	if logId == "" {
		logId = GenLogId()
	}

	lf := &LogFitter{
		logger:       logger,
		logId:        logId,
		pid:          os.Getpid(),
		commFields:   make([]interface{}, 0),
		commFieldLck: &sync.RWMutex{},
		infoFields:   make([]interface{}, 0),
		infoFieldLck: &sync.RWMutex{},
		callDepth:    DefaultCallDepth,
	}

	return lf, nil
}

func (t *LogFitter) GetLogId() string {
	return t.logId
}

func (t *LogFitter) SetCommField(key string, value interface{}) {
	if !t.isInit() || key == "" || value == nil {
		return
	}

	t.commFieldLck.Lock()
	defer t.commFieldLck.Unlock()

	t.commFields = append(t.commFields, key, value)
}

func (t *LogFitter) SetInfoField(key string, value interface{}) {
	if !t.isInit() || key == "" || value == nil {
		return
	}

	t.infoFieldLck.Lock()
	defer t.infoFieldLck.Unlock()

	t.infoFields = append(t.infoFields, key, value)
}

func (t *LogFitter) Error(msg string, ctx ...interface{}) {
	if !t.isInit() {
		return
	}
	t.logger.Error(msg, t.fmtCommLogger(ctx...)...)
}

func (t *LogFitter) Warn(msg string, ctx ...interface{}) {
	if !t.isInit() {
		return
	}
	t.logger.Warn(msg, t.fmtCommLogger(ctx...)...)
}

func (t *LogFitter) Info(msg string, ctx ...interface{}) {
	if !t.isInit() {
		return
	}
	t.logger.Info(msg, t.fmtInfoLogger(ctx...)...)
}

func (t *LogFitter) Trace(msg string, ctx ...interface{}) {
	if !t.isInit() {
		return
	}
	t.logger.Trace(msg, t.fmtCommLogger(ctx...)...)
}

func (t *LogFitter) Debug(msg string, ctx ...interface{}) {
	if !t.isInit() {
		return
	}

	t.logger.Debug(msg, t.fmtCommLogger(ctx...)...)
}

func (t *LogFitter) getCommField() []interface{} {
	t.commFieldLck.RLock()
	defer t.commFieldLck.RUnlock()

	return t.commFields
}

func (t *LogFitter) genBaseField() []interface{} {
	fileLine, _ := GetFuncCall(t.callDepth)

	comCtx := make([]interface{}, 0)
	// 保持log_id是第一个写入，方便替换
	comCtx = append(comCtx, CommFieldLogId, t.logId)
	comCtx = append(comCtx, CommFieldCall, fileLine)
	comCtx = append(comCtx, CommFieldPid, t.pid)

	return comCtx
}

func (t *LogFitter) fmtCommLogger(ctx ...interface{}) []interface{} {
	if len(ctx)%2 != 0 {
		last := ctx[len(ctx)-1]
		ctx = ctx[:len(ctx)-1]
		ctx = append(ctx, "unknow", last)
	}

	// Ensure consistent output sequence
	comCtx := t.genBaseField()
	// 如果设置了log_id，用设置的log_id替换公共字段
	if len(ctx) > 1 && fmt.Sprintf("%v", ctx[0]) == CommFieldLogId {
		comCtx[1] = ctx[1]
		ctx = ctx[2:]
	}
	comCtx = append(comCtx, t.getCommField()...)
	comCtx = append(comCtx, ctx...)

	return comCtx
}

func (t *LogFitter) getInfoField() []interface{} {
	t.infoFieldLck.RLock()
	defer t.infoFieldLck.RUnlock()

	return t.infoFields
}

func (t *LogFitter) fmtInfoLogger(ctx ...interface{}) []interface{} {
	if len(ctx)%2 != 0 {
		last := ctx[len(ctx)-1]
		ctx = ctx[:len(ctx)-1]
		ctx = append(ctx, "unknow", last)
	}

	comCtx := t.genBaseField()
	// 如果设置了log_id，用设置的log_id替换公共字段
	if len(ctx) > 1 && fmt.Sprintf("%v", ctx[0]) == CommFieldLogId {
		comCtx[1] = ctx[1]
		ctx = ctx[2:]
	}
	comCtx = append(comCtx, t.getCommField()...)
	comCtx = append(comCtx, t.getInfoField()...)
	comCtx = append(comCtx, ctx...)

	t.clearInfoFields()
	return comCtx
}

func (t *LogFitter) clearInfoFields() {
	t.infoFieldLck.RLock()
	defer t.infoFieldLck.RUnlock()

	t.infoFields = t.infoFields[:0]
}

func (t *LogFitter) isInit() bool {
	if t.logger == nil || t.commFields == nil || t.infoFields == nil ||
		t.commFieldLck == nil || t.infoFieldLck == nil {
		return false
	}

	return true
}
