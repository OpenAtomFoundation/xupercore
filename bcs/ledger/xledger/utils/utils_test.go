package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuperchain/xupercore/kernel/mock"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
)

func TestCreateLedger(t *testing.T) {
	workspace, dirErr := ioutil.TempDir("/tmp", "")
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)

	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}

	genesisConf := econf.GenDataAbsPath("genesis/xuper.json")
	err = CreateLedger("xuper", genesisConf, econf)
	if err != nil {
		t.Fatal(err)
	}

	dataDir := econf.GenDataAbsPath(econf.ChainDir)
	fullpath := filepath.Join(dataDir, "xuper")
	os.RemoveAll(fullpath)
}
