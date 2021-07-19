package asyncworker

import (
	"encoding/json"
	"path/filepath"
	"testing"

	lconf "github.com/xuperchain/xupercore/bcs/ledger/xledger/config"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/mock"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

const (
	ledgerPath = "../../../../example/xchain/conf/ledger.yaml"
	logPath    = "../../../../example/xchain/conf/log.yaml"
)

var (
	tmpBaseDB      kvdb.Database
	tmpFinishTable kvdb.Database
)

func newTxs() []*protos.FilteredTransaction {
	var txs []*protos.FilteredTransaction
	txs = append(txs, &protos.FilteredTransaction{
		Txid: "txid_1",
		Events: []*protos.ContractEvent{
			{
				Contract: "$parachain",
				Name:     "CreateBlockChain",
				Body:     []byte("hello2"),
			},
		},
	})
	return txs
}

func newDB() error {
	dir := utils.GetCurFileDir()
	lcfg, err := lconf.LoadLedgerConf(filepath.Join(dir, ledgerPath))
	if err != nil {
		return err
	}
	asyncDBPath := filepath.Join(dir, "/tmp/db")
	// 目前仅使用默认设置
	kvParam := &kvdb.KVParameter{
		DBPath:                asyncDBPath,
		KVEngineType:          lcfg.KVEngineType,
		MemCacheSize:          ledger.MemCacheSize,
		FileHandlersCacheSize: ledger.FileHandlersCacheSize,
		OtherPaths:            lcfg.OtherPaths,
		StorageType:           lcfg.StorageType,
	}
	baseDB, err := kvdb.CreateKVInstance(kvParam)
	if err != nil {
		return err
	}
	if tmpBaseDB == nil {
		tmpBaseDB = baseDB
	}
	if tmpFinishTable == nil {
		tmpFinishTable = kvdb.NewTable(baseDB, FinishTablePrefix)
	}
	return nil
}

func newAsyncWorker() *AsyncWorkerImpl {
	aw := AsyncWorkerImpl{
		bcname: "xuper",
		filter: &protos.BlockFilter{
			Bcname:   "xuper",
			Contract: `^\$`,
		},
		close: make(chan struct{}, 1),
	}
	return &aw
}

func newAsyncWorkerWithDB(t *testing.T) (*AsyncWorkerImpl, error) {
	aw := newAsyncWorker()
	dir := utils.GetCurFileDir()

	// log实例
	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(dir, "/tmp/log")
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), logPath)
	log, _ := logs.NewLogger("", "asyncworker")
	aw.log = log

	// db实例
	err = newDB()
	if err != nil {
		t.Errorf("newDB error %v", err)
		return nil, err
	}

	aw.finishTable = tmpFinishTable

	return aw, nil
}

func handleCreateChain(ctx common.TaskContext) error {
	return nil
}

func TestRegisterHandler(t *testing.T) {
	aw := newAsyncWorker()
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	if aw.methods["$parachain"] == nil {
		t.Errorf("RegisterHandler register contract error")
		return
	}
	if aw.methods["$parachain"]["CreateBlockChain"] == nil {
		t.Errorf("RegisterHandler register event error")
	}
}

func TestGetAsyncTask(t *testing.T) {
	aw := newAsyncWorker()
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	handler, err := aw.getAsyncTask("$parachain", "CreateBlockChain")
	if err != nil {
		t.Errorf("getAsyncTask error")
		return
	}
	ctx := newTaskContextImpl([]byte("hello"))
	if handler(ctx) != nil {
		t.Errorf("getAsyncTask ctx error")
		return
	}
}

func TestReloadCursor(t *testing.T) {
	aw, err := newAsyncWorkerWithDB(t)
	if err != nil {
		t.Errorf("create db error, err=%v", err)
		return
	}
	// 执行完毕后进行持久化
	cursor := &asyncWorkerCursor{
		BlockHeight: 1,
		TxIndex:     int64(0),
		EventIndex:  int64(0),
	}
	cursorBuf, err := json.Marshal(cursor)
	if err != nil {
		t.Errorf("marshal cursor failed when doAsyncTasks, err=%v", err)
		return
	}
	aw.finishTable.Put([]byte("xuper"), cursorBuf)
	cursor, err = aw.reloadCursor()
	if err != nil {
		t.Errorf("reloadCursor err=%v", err)
		return
	}
	if cursor.BlockHeight != 1 || cursor.TxIndex != 0 || cursor.EventIndex != 0 {
		t.Errorf("reloadCursor value error")
		return
	}
	aw.finishTable.Delete([]byte("xuper"))
}

func TestDoAsyncTasks(t *testing.T) {
	aw, err := newAsyncWorkerWithDB(t)
	if err != nil {
		t.Errorf("create db error, err=%v", err)
		return
	}
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	err = aw.doAsyncTasks(newTxs(), 3, nil)
	if err != nil {
		t.Errorf("doAsyncTasks error")
		return
	}
	cursor, err := aw.reloadCursor()
	if cursor.BlockHeight != 3 {
		t.Errorf("doAsyncTasks reloadCursor error")
	}
}

func TestStartAsyncTask(t *testing.T) {
	aw, err := newAsyncWorkerWithDB(t)
	if err != nil {
		t.Errorf("create db error, err=%v", err)
		return
	}
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	aw.Start()
}
