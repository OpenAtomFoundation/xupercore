package context

import (
	"testing"

	"github.com/xuperchain/xupercore/kernel/network/config"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

var log logs.LogDriver

func GetLog() (logs.Logger, error) {
	if log == nil {
		lg, err := logs.GetLog()
		if err != nil {
			return nil, err
		}
		log = lg
	}

	return logs.GetLogFitter(log)
}

func TestCreateDomainCtx(t *testing.T) {
	logf, err := GetLog()
	if err != nil {
		t.Errorf("new log failed.err:%v", err)
	}

	octx, _ := CreateDomainCtx(logf, config.GetNetConfFile())
	octx.GetLog().Debug("test CreateDomainCtx", "cfg", octx.GetP2PConf())
}

func TestCreateOperateCtx(t *testing.T) {
	logf, err := GetLog()
	if err != nil {
		t.Errorf("new log failed.err:%v", err)
	}

	fctx, _ := CreateOperateCtx(logf, timer.NewXTimer())
	fctx.GetLog().Debug("test CreateOperateCtx", "timer", fctx.GetTimer().Print())
}
