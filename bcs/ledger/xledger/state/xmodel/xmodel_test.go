package xmodel

import (
	"fmt"
	kledger "github.com/xuperchain/xupercore/kernel/ledger"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	ledger_pkg "github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/mock"
	crypto_client "github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
	"github.com/xuperchain/xupercore/protos"
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

func TestBaiscFunc(t *testing.T) {
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
	xModel, err := NewXModel(sctx, ldb)
	if err != nil {
		t.Fatal(err)
	}
	verData, err := xModel.Get("bucket1", []byte("hello"))
	if !IsEmptyVersionedData(verData) {
		t.Fatal("unexpected")
	}
	tx1 := &pb.Transaction{
		Txid: []byte("Tx1"),
		TxInputsExt: []*protos.TxInputExt{
			&protos.TxInputExt{
				Bucket: "bucket1",
				Key:    []byte("hello"),
			},
		},
		TxOutputsExt: []*protos.TxOutputExt{
			&protos.TxOutputExt{
				Bucket: "bucket1",
				Key:    []byte("hello"),
				Value:  []byte("you are the best!"),
			},
		},
	}
	batch := ldb.NewBatch()
	err = xModel.DoTx(tx1, batch)
	if err != nil {
		t.Fatal(err)
	}
	saveUnconfirmTx(tx1, batch)
	err = batch.Write()
	if err != nil {
		t.Fatal(err)
	}
	verData, err = xModel.Get("bucket1", []byte("hello"))
	if GetVersion(verData) != fmt.Sprintf("%x_%d", "Tx1", 0) {
		t.Fatal("unexpected", GetVersion(verData))
	}
	tx2 := &pb.Transaction{
		Txid: []byte("Tx2"),
		TxInputsExt: []*protos.TxInputExt{
			&protos.TxInputExt{
				Bucket:    "bucket1",
				Key:       []byte("hello"),
				RefTxid:   []byte("Tx1"),
				RefOffset: 0,
			},
			&protos.TxInputExt{
				Bucket: "bucket1",
				Key:    []byte("world"),
			},
		},
		TxOutputsExt: []*protos.TxOutputExt{
			&protos.TxOutputExt{
				Bucket: "bucket1",
				Key:    []byte("hello"),
				Value:  []byte("\x00"),
			},
			&protos.TxOutputExt{
				Bucket: "bucket1",
				Key:    []byte("world"),
				Value:  []byte("world is full of love!"),
			},
		},
	}
	_, err = ParseContractUtxoInputs(tx2)
	if err != nil {
		t.Fatal(err)
	}
	prefix := GenWriteKeyWithPrefix(tx2.TxOutputsExt[0])
	t.Log("gen prefix succ", "prefix", prefix)
	batch2 := ldb.NewBatch()
	err = xModel.DoTx(tx2, batch2)
	if err != nil {
		t.Fatal(err)
	}
	saveUnconfirmTx(tx2, batch2)
	err = batch2.Write()
	if err != nil {
		t.Fatal(err)
	}
	verData, err = xModel.Get("bucket1", []byte("hello"))
	if GetVersion(verData) != fmt.Sprintf("%x_%d", "Tx2", 0) {
		t.Fatal("unexpected", GetVersion(verData))
	}
	iter, err := xModel.Select("bucket1", []byte(""), []byte("\xff"))
	defer iter.Close()
	validKvCount := 0
	for iter.Next() {
		t.Logf("iter:  data %v, key: %s\n", iter.Value(), iter.Key())
		validKvCount++
	}
	if validKvCount != 1 {
		t.Fatal("unexpected", validKvCount)
	}
	_, isConfiremd, err := xModel.QueryTx(tx2.Txid)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("query succ", "isConfirmed", isConfiremd)
	}
	xModel.CleanCache()
	xModel.BucketCacheDelete("bucket1", GetVersion(verData))
	_, err = xModel.QueryBlock([]byte("123"))
	if err != nil {
		t.Log(err)
	}

	vData, err := xModel.GetFromLedger(tx2.TxInputsExt[0])
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(vData)
	}
	version := MakeVersion(tx2.Txid, 0)
	txid := GetTxidFromVersion(version)
	t.Log("txid", txid)
	vDatas := make([]*kledger.VersionedData, 0, 1)
	vDatas = append(vDatas, vData)
	GetTxInputs(vDatas)

	pds := []*kledger.PureData{
		&kledger.PureData{
			Bucket: "bucket1",
			Key:    []byte("key1"),
			Value:  []byte("value1"),
		},
	}
	GetTxOutputs(pds)
	vData, exists, err := xModel.GetWithTxStatus("bucket1", []byte("hello"))
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("get txStatus succ", "data", vData, "exist", exists)
	}

	err = xModel.UndoTx(tx1, batch)
	if err != nil {
		t.Fatal(err)
	}

	ledger.Close()
}
