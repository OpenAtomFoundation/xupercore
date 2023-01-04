package mock

import (
	"fmt"
	"path/filepath"

	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

func NewEnvConfForTest(paths ...string) (*xconf.EnvConf, error) {
	path := "conf/env.yaml"
	if len(paths) > 0 {
		path = paths[0]
	}

	dir := utils.GetCurFileDir()
	econfPath := filepath.Join(dir, path)
	econf, err := xconf.LoadEnvConf(econfPath)
	if err != nil {
		return nil, err
	}

	econf.RootPath = utils.GetCurFileDir()
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	return econf, nil
}

func InitLogForTest() {
	_, err := NewEnvConfForTest()
	if err != nil {
		fmt.Printf("InitLogForTest() error: %s", err)
	}
}
