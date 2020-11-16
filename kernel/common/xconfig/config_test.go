package xconfig

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xuperchain/xupercore/lib/utils"
)

func TestLoadEnvConf(t *testing.T) {
	envCfg, err := LoadEnvConf(getConfFile())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(envCfg.GenDirAbsPath(envCfg.ConfDir))
	fmt.Println(envCfg.GenDirAbsPath(envCfg.DataDir))
	fmt.Println(envCfg.GenDirAbsPath(envCfg.LogDir))
	fmt.Println(envCfg.GenConfFilePath(envCfg.LogConf))
}

func getConfFile() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/env.yaml")
}
