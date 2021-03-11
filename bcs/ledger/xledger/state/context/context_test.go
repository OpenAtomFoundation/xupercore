package context

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/kernel/mock"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
)

func TestNewNetCtx(t *testing.T) {
	workspace, dirErr := ioutil.TempDir("/tmp", "")
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	mock.InitLogForTest()

	ecfg, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}

	lctx, err := ledger.NewLedgerCtx(ecfg, "xuper")
	if err != nil {
		t.Fatal(err)
	}
	lctx.EnvCfg.ChainDir = workspace

	genesisConf := []byte(`
		{
    "version": "1",
    "predistribution": [
        {
            "address": "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
            "quota": "100000000000000000000"
        }
    ],
    "maxblocksize": "16",
    "award": "1000000",
    "decimals": "8",
    "award_decay": {
        "height_gap": 31536000,
        "ratio": 1
    },
    "gas_price": {
        "cpu_rate": 1000,
        "mem_rate": 1000000,
        "disk_rate": 1,
        "xfee_rate": 1
    },
    "new_account_resource_amount": 1000,
    "genesis_consensus": {
        "name": "single",
        "config": {
            "miner": "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
            "period": 3000
        }
    }
}
    `)
	ledgerIns, err := ledger.CreateLedger(lctx, genesisConf)
	if err != nil {
		t.Fatal(err)
	}
	gcc, err := client.CreateCryptoClient("gm")
	if err != nil {
		t.Errorf("gen crypto client fail.err:%v", err)
	}
	sctx, err := NewStateCtx(ecfg, "xuper", ledgerIns, gcc)
	if err != nil {
		t.Fatal(err)
	}
	sctx.XLog.Debug("test NewNetCtx succ", "sctx", sctx)

	isInit := sctx.IsInit()
	t.Log("is init", isInit)
}
