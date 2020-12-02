package context

import (
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
	"log"
	"path/filepath"
)

var (
	workDir = filepath.Join(utils.GetCurFileDir(), "/../../..")
)

func MockLog() (logs.Logger, error) {
	logs.InitLog(filepath.Join(workDir, "/conf/log.yaml"), filepath.Join(workDir, "/logs"))
	return logs.NewLogger("123567890", "network")
}

func MockDomainCtx(paths ...string) DomainCtx {
	confPath := filepath.Join(workDir, "/conf/network.yaml")
	if len(paths) > 0 {
		confPath = paths[0]
	}

	xlog, err := MockLog()
	if err != nil {
		log.Printf("mock log error: %v", err)
		return new(DomainCtxImpl)
	}

	ctx, err := CreateDomainCtx(xlog, confPath)
	if err != nil {
		log.Printf("CreateDomainCtx error: %v", err)
		return new(DomainCtxImpl)
	}

	return ctx
}
