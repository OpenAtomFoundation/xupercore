package meta

import (
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/kernel/mock"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/protos"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	ledger_pkg "github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	txn "github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	crypto_client "github.com/xuperchain/xupercore/lib/crypto/client"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
)

// common test data
const (
	BobAddress   = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	AliceAddress = "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"
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

func TestMetaGetFunc(t *testing.T) {
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
	//创建链的时候分配财富
	tx, err := txn.GenerateRootTx([]byte(`
       {
        "version" : "1"
        , "consensus" : {
                "miner" : "0x00000000000"
        }
        , "predistribution":[
                {
                        "address" : "` + BobAddress + `",
                        "quota" : "10000000"
                },
				{
                        "address" : "` + AliceAddress + `",
                        "quota" : "20000000"
                }

        ]
        , "maxblocksize" : "128"
        , "period" : "5000"
        , "award" : "1000"
		} 
    `))
	if err != nil {
		t.Fatal(err)
	}

	block, _ := ledger.FormatRootBlock([]*pb.Transaction{tx})
	t.Logf("blockid %x", block.Blockid)
	confirmStatus := ledger.ConfirmBlock(block, true)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail")
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
	metaHadler, err := NewMeta(sctx, ldb)
	if err != nil {
		t.Fatal(err)
	}
	maxBlockSize := metaHadler.GetMaxBlockSize()
	t.Log("get succ", "maxBlockSize", maxBlockSize)
	slideWindow := metaHadler.GetIrreversibleSlideWindow()
	t.Log("get succ", "slideWindow", slideWindow)
	forbid := metaHadler.GetForbiddenContract()
	t.Log("get succ", "forbidContract", forbid)
	gchain := metaHadler.GetGroupChainContract()
	t.Log("get succ", "groupChainContract", gchain)
	gasPrice := metaHadler.GetGasPrice()
	t.Log("get succ", "gasPrice", gasPrice)
	bHeight := metaHadler.GetIrreversibleBlockHeight()
	t.Log("get succ", "blockHeight", bHeight)
	amount := metaHadler.GetNewAccountResourceAmount()
	t.Log("get succ", "newAccountResourceAmount", amount)
	contracts := metaHadler.GetReservedContracts()
	if len(contracts) == 0 {
		t.Log("empty reserved contracts")
	}
	_, err = metaHadler.MaxTxSizePerBlock()
	if err != nil {
		t.Fatal(err)
	}

	batch := ldb.NewBatch()
	err = metaHadler.UpdateIrreversibleBlockHeight(2, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateNextIrreversibleBlockHeightForPrune(3, 2, 1, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateNextIrreversibleBlockHeight(3, 3, 1, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateIrreversibleSlideWindow(2, batch)
	if err != nil {
		t.Fatal(err)
	}
	gasPrice = &protos.GasPrice{
		CpuRate:  100,
		MemRate:  100000,
		DiskRate: 1,
		XfeeRate: 1,
	}
	err = metaHadler.UpdateGasPrice(gasPrice, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateNewAccountResourceAmount(500, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateMaxBlockSize(64, batch)
	if err != nil {
		t.Fatal(err)
	}
	reqs := make([]*protos.InvokeRequest, 0, 1)
	request := &protos.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: "identity",
		MethodName:   "verify",
		Args:         map[string][]byte{},
	}
	reqs = append(reqs, request)
	err = metaHadler.UpdateReservedContracts(reqs, batch)
	if err != nil {
		t.Fatal(err)
	}
	upForbidRequest := &protos.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: "forbidden",
		MethodName:   "get1",
		Args:         map[string][]byte{},
	}
	err = metaHadler.UpdateForbiddenContract(upForbidRequest, batch)
	if err != nil {
		t.Fatal(err)
	}
}
