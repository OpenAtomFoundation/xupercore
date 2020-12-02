package context

import (
	"testing"

	"github.com/xuperchain/xupercore/kernel/network/config"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var log *logs.LogFitter

func GetLog() (logs.Logger, error) {
	if log != nil {
		return log, nil
	}

	curDir := utils.GetCurFileDir() + "/../../.."
	logs.InitLog(curDir+"/conf/log.yaml", curDir+"/logs")
	xlog, err := logs.NewLogger("123567890", "p2p")
	if err != nil {
		return nil, err
	}

	xlog.SetCommField("submodule", "p2p")
	log = xlog
	return log, nil
}

func TestCreateDomainCtx(t *testing.T) {
	logf, err := GetLog()
	if err != nil {
		t.Errorf("new log failed.err:%v", err)
	}

	octx, _ := CreateDomainCtx(logf, config.GetNetConfFile())
	octx.GetLog().Debug("test CreateDomainCtx", "cfg", octx.GetP2PConf())
}
