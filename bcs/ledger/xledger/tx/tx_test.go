package tx

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	ledger_pkg "github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/kernel/mock"
	crypto_client "github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
)

var GenesisConf = []byte(`
		{
    "version": "1",
    "predistribution": [
        {
            "address": "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN",
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
            "miner": "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN",
            "period": 3000
        }
    }
}
    `)

func TestTx(t *testing.T) {
	_, minerErr := GenerateAwardTx("miner-1", "1000", []byte("award"))
	if minerErr != nil {
		t.Fatal(minerErr)
	}
	_, etxErr := GenerateEmptyTx([]byte("empty"))
	if etxErr != nil {
		t.Fatal(minerErr)
	}
	_, rtxErr := GenerateRootTx(GenesisConf)
	if rtxErr != nil {
		t.Fatal(rtxErr)
	}

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
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))

	lctx, err := ledger_pkg.NewLedgerCtx(econf, "xuper")
	if err != nil {
		t.Fatal(err)
	}
	lctx.EnvCfg.ChainDir = workspace

	ledger, err := ledger_pkg.CreateLedger(lctx, GenesisConf)
	if err != nil {
		t.Fatal(err)
	}

	crypt, err := crypto_client.CreateCryptoClient(crypto_client.CryptoTypeDefault)
	if err != nil {
		t.Fatal(err)
	}

	sctx, err := context.NewStateCtx(econf, "xuper", ledger, crypt)
	if err != nil {
		t.Fatal(err)
	}
	sctx.EnvCfg.ChainDir = workspace

	storePath := sctx.EnvCfg.GenDataAbsPath(sctx.EnvCfg.ChainDir)
	storePath = filepath.Join(storePath, sctx.BCName)
	stateDBPath := filepath.Join(storePath, def.StateStrgDirName)
	kvParam := &kvdb.KVParameter{
		DBPath:                stateDBPath,
		KVEngineType:          sctx.LedgerCfg.KVEngineType,
		MemCacheSize:          ledger_pkg.MemCacheSize,
		FileHandlersCacheSize: ledger_pkg.FileHandlersCacheSize,
		OtherPaths:            sctx.LedgerCfg.OtherPaths,
		StorageType:           sctx.LedgerCfg.StorageType,
	}
	ldb, err := kvdb.CreateKVInstance(kvParam)
	if err != nil {
		t.Fatal(err)
	}
	txHandle, err := NewTx(sctx, ldb)
	if err != nil {
		t.Fatal(err)
	}
	err = txHandle.LoadUnconfirmedTxFromDisk()
	if err != nil {
		t.Fatal(err)
	}
	unConfirmedTx, err := txHandle.GetUnconfirmedTx(false)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("unconfirmed tx len:", len(unConfirmedTx))
	}
	txHandle.SetMaxConfirmedDelay(500)
	txMap, txGraph, txDelay, err := txHandle.SortUnconfirmedTx()
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("sort txs", "txMap", txMap, "txGraph", txGraph, "txDelay", txDelay)
	}
}
