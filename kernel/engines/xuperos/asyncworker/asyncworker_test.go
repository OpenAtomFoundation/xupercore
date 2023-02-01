package asyncworker

import (
	"encoding/json"
	"io/ioutil"
	"os"
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

	testBcName = "parachain"
)

func newTx() []*protos.FilteredTransaction {
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

func newTxs() []*protos.FilteredTransaction {
	txs := newTx()
	txs = append(txs, &protos.FilteredTransaction{
		Txid: "txid_2",
		Events: []*protos.ContractEvent{
			{
				Contract: "$parachain",
				Name:     "CreateBlockChain",
				Body:     []byte("hello3"),
			},
		},
	})
	return txs
}

func newAsyncWorker() *AsyncWorkerImpl {
	aw := AsyncWorkerImpl{
		bcname: testBcName,
		filter: &protos.BlockFilter{
			Bcname:   testBcName,
			Contract: `^\$`,
		},
		close: make(chan struct{}, 1),
	}
	return &aw
}

func handleCreateChain(ctx common.TaskContext) error {
	return nil
}

func TestRegisterHandler(t *testing.T) {
	aw := newAsyncWorker()
	th, err := NewTestHelper()
	if err != nil {
		t.Errorf("NewTestHelper error")
	}
	defer th.close()
	aw.finishTable = th.db
	aw.log = th.log
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	if aw.methods["$parachain"] == nil {
		t.Errorf("RegisterHandler register contract error")
		return
	}
	if aw.methods["$parachain"]["CreateBlockChain"] == nil {
		t.Errorf("RegisterHandler register event error")
	}
	aw.RegisterHandler("", "", handleCreateChain)
	aw.RegisterHandler("parachain", "CreateBlockChain", handleCreateChain)
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
}

func TestGetAsyncTask(t *testing.T) {
	aw := newAsyncWorker()
	_, err := aw.getAsyncTask("", "CreateBlockChain")
	if err == nil {
		t.Errorf("getAsyncTask error")
		return
	}
	_, err = aw.getAsyncTask("$parachain", "CreateBlockChain")
	if err == nil {
		t.Errorf("getAsyncTask error")
		return
	}
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

func TestCursor(t *testing.T) {
	aw := newAsyncWorker()
	th, err := NewTestHelper()
	if err != nil {
		t.Errorf("NewTestHelper error")
	}
	defer th.close()
	aw.finishTable = th.db
	aw.log = th.log
	_, err = aw.reloadCursor()
	if err != emptyErr {
		t.Errorf("reload error, err=%v", err)
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
	_ = aw.finishTable.Put([]byte(testBcName), cursorBuf)
	cursor, err = aw.reloadCursor()
	if err != nil {
		t.Errorf("reloadCursor err=%v", err)
		return
	}
	if cursor.BlockHeight != 1 || cursor.TxIndex != 0 || cursor.EventIndex != 0 {
		t.Errorf("reloadCursor value error")
		return
	}
	err = aw.storeCursor(asyncWorkerCursor{
		BlockHeight: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoAsyncTasks(t *testing.T) {
	aw := newAsyncWorker()
	th, err := NewTestHelper()
	if err != nil {
		t.Errorf("NewTestHelper error")
	}
	defer th.close()
	aw.finishTable = th.db
	aw.log = th.log
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	err = aw.doAsyncTasks(newTx(), 3, nil)
	if err != nil {
		t.Errorf("doAsyncTasks error")
		return
	}
	cursor, err := aw.reloadCursor()
	if err != nil {
		t.Errorf("reloadCursor error")
		return
	}
	if cursor.BlockHeight != 3 || cursor.TxIndex != 0 || cursor.EventIndex != 0 {
		t.Errorf("doAsyncTasks block cursor error")
	}

	// 模拟中断存储cursor
	cursor = &asyncWorkerCursor{
		BlockHeight: 5,
		TxIndex:     int64(1),
		EventIndex:  int64(0),
	}
	cursorBuf, _ := json.Marshal(cursor)
	_ = aw.finishTable.Put([]byte(testBcName), cursorBuf)
	if err := aw.doAsyncTasks(newTxs(), 5, cursor); err != nil {
		t.Fatal(err)
	}
	if cursor.BlockHeight != 5 || cursor.TxIndex != 1 || cursor.EventIndex != 0 {
		t.Errorf("doAsyncTasks block break cursor error")
	}
}

func TestStartAsyncTask(t *testing.T) {
	aw := newAsyncWorker()
	th, err := NewTestHelper()
	if err != nil {
		t.Errorf("NewTestHelper error")
	}
	defer th.close()
	aw.finishTable = th.db
	aw.log = th.log
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	// TODO: deal with error
	_ = aw.Start()
	aw.Stop()
}

type TestHelper struct {
	basedir string
	db      kvdb.Database
	log     logs.Logger
}

func NewTestHelper() (*TestHelper, error) {
	basedir, err := ioutil.TempDir("", "asyncworker-test")
	if err != nil {
		return nil, err
	}

	dir := utils.GetCurFileDir()
	lcfg, err := lconf.LoadLedgerConf(filepath.Join(dir, ledgerPath))
	if err != nil {
		return nil, err
	}
	// 目前仅使用默认设置
	kvParam := &kvdb.KVParameter{
		DBPath:                basedir,
		KVEngineType:          lcfg.KVEngineType,
		MemCacheSize:          ledger.MemCacheSize,
		FileHandlersCacheSize: ledger.FileHandlersCacheSize,
		OtherPaths:            lcfg.OtherPaths,
		StorageType:           lcfg.StorageType,
	}
	baseDB, err := kvdb.CreateKVInstance(kvParam)
	if err != nil {
		return nil, err
	}
	tmpFinishTable := kvdb.NewTable(baseDB, FinishTablePrefix)

	// log实例
	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(basedir, "/log")
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), logPath)
	log, _ := logs.NewLogger("", "asyncworker")

	th := &TestHelper{
		basedir: basedir,
		db:      tmpFinishTable,
		log:     log,
	}
	return th, nil
}

func (th *TestHelper) close() {
	th.db.Close()
	os.RemoveAll(th.basedir)
}
