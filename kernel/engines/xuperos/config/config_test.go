package config

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xuperchain/xupercore/lib/utils"
)

func TestLoadEngineConf(t *testing.T) {
	engCfg, err := LoadEngineConf(getConfFile())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(engCfg)
}

func getConfFile() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/engine.yaml")
}
