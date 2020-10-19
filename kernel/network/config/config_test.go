package config

import (
	"fmt"
	"testing"
)

func TestGetDefP2PConf(t *testing.T) {
	cfg := GetDefP2PConf()
	fmt.Println(cfg)
}

func TestLoadP2PConf(t *testing.T) {
	cfg, err := LoadP2PConf(GetNetConfFile())
	if err != nil {
		t.Errorf("load p2p config failed.err:%v", err)
	}

	fmt.Println(cfg)
}
