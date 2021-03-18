package config

import (
	"fmt"
	"testing"

	"github.com/xuperchain/xupercore/kernel/mock"
)

func TestLoadP2PConf(t *testing.T) {
	envConf, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Errorf("mock env conf error: %v", err)
		return
	}

	cfg, err := LoadP2PConf(envConf.GenConfFilePath("network.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(cfg)
}
