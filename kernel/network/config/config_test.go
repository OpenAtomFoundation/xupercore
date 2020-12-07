package config

import (
	"fmt"
	"testing"

	"github.com/xuperchain/xupercore/kernel/mock"
)

func TestLoadP2PConf(t *testing.T) {
	cfg, err := LoadP2PConf(mock.GetNetConfPathForTest())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(cfg)
}
