package logs

import (
	"fmt"
	"testing"
)

func TestGetDefLogConf(t *testing.T) {
	cfg := GetDefLogConf()
	fmt.Println(cfg)
}

func TestLoadLogConf(t *testing.T) {
	cfg, err := LoadLogConf(GetConfFile())
	if err != nil {
		t.Errorf("load log config failed.err:%v", err)
	}

	fmt.Println(cfg)
}
