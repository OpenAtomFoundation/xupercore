package mock

import (
	"path/filepath"

	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

func NewEnvConfForTest() (*xconf.EnvConf, error) {
	dir := utils.GetCurFileDir()
	econfPath := filepath.Join(dir, "conf/env.yaml")

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
