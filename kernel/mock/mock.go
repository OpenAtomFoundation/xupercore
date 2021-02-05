package mock

import (
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
	return xconf.LoadEnvConf(econfPath)
}

func GetNetConfPathForTest() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/network.yaml")
}

func InitLogForTest() error {
	ecfg, err := NewEnvConfForTest()
	if err != nil {
		return err
	}

	logs.InitLog(ecfg.GenConfFilePath(ecfg.LogConf), ecfg.GenDirAbsPath(ecfg.LogDir))
	return nil
}
