package config

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xuperchain/xupercore/lib/utils"
)

func TestGetDefLogConf(t *testing.T) {
	cfg := GetDefLogConf()
	fmt.Println(cfg)
}

func TestLoadLogConf(t *testing.T) {
	cfg, err := LoadLogConf(getConfFile())
	if err != nil {
		t.Errorf("load log config failed.err:%v", err)
	}

	fmt.Println(cfg)
}

func getConfFile() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/log.yaml")
}

func getLogDir() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "logs")
}
