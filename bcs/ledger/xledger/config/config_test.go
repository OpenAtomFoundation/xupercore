package config

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xuperchain/xupercore/lib/utils"
)

func TestLoadLedgerConf(t *testing.T) {
	ledgerCfg, err := LoadLedgerConf(getConfFile())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(ledgerCfg)
}

func getConfFile() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/ledger.yaml")
}
