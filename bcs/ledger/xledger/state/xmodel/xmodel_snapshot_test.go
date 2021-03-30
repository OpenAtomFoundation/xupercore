package xmodel

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	ledger_pkg "github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/mock"
	crypto_client "github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
	"github.com/xuperchain/xupercore/protos"
)

const (
	BobAddress = "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"
)

func TestGet(t *testing.T) {
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

	t1 := &pb.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &protos.TxOutput{Amount: []byte("888"), ToAddr: []byte(BobAddress)})
	t1.Coinbase = true
	t1.Desc = []byte(`{"maxblocksize" : "128"}`)
	t1.Txid, _ = txhash.MakeTransactionID(t1)
	block, err := ledger.FormatRootBlock([]*pb.Transaction{t1})
	if err != nil {
		t.Fatal(err)
	}

	confirmStatus := ledger.ConfirmBlock(block, true)
	if !confirmStatus.Succ {
		t.Fatal(fmt.Errorf("confirm block fail"))
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
	xmod, err := NewXModel(sctx, ldb)
	if err != nil {
		t.Fatal(err)
	}

	blkId, err := ledger.QueryBlockByHeight(0)
	if err != nil {
		t.Fatal(err)
	}

	xmsp, err := xmod.CreateSnapshot(blkId.Blockid)
	if err != nil {
		t.Log(err)
	}

	vData, err := xmsp.Get("proftestc", []byte("key_1"))
	if err != nil {
		t.Log(err)
	}

	fmt.Println(vData)
	fmt.Println(hex.EncodeToString(vData.RefTxid))

	ledger.Close()
}
