package asyncworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	lconf "github.com/xuperchain/xupercore/bcs/ledger/xledger/config"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/event"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/protos"
)

const (
	FinishTablePrefix = "FT"
)

var (
	// ErrRetry从TaskHandler返回指示这次任务应该被重试
	ErrRetry  = errors.New("retry task")
	eventType = protos.SubscribeType_BLOCK

	emptyErr  = errors.New("Haven't store cursor before")
	cursorErr = errors.New("DB stored an invalid cursor")
)

type AsyncWorkerImpl struct {
	bcname  string
	mutex   sync.Mutex
	methods map[string]map[string]common.TaskHandler // 句柄存储

	filter *protos.BlockFilter // 订阅event事件时的筛选正则
	router *event.Router       // 通过router进行事件订阅

	baseDB      kvdb.Database //持久化执行过的异步任务
	finishTable kvdb.Database //保存临时的block区块

	close chan struct{}

	log logs.Logger
}

func NewAsyncWorkerImpl(bcName string, e common.Engine) (*AsyncWorkerImpl, error) {
	aw := &AsyncWorkerImpl{
		filter: &protos.BlockFilter{
			Bcname:   bcName,
			Contract: `^\$`,
		},
		close:  make(chan struct{}, 1),
		router: event.NewRouter(e),
		log:    e.Context().XLog,
	}

	// new kvdb instance
	envCfg := e.Context().EnvCfg
	lcfg, err := lconf.LoadLedgerConf(envCfg.GenConfFilePath(envCfg.LedgerConf))
	storePath := envCfg.GenDataAbsPath(e.Context().EnvCfg.ChainDir)
	storePath = filepath.Join(storePath, bcName)
	asyncDBPath := filepath.Join(storePath, def.AsyncWorkerDirName)
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
		aw.log.Error("fail to open asyncworker leveldb.", "error", err)
		return nil, err
	}
	aw.finishTable = kvdb.NewTable(baseDB, FinishTablePrefix)
	return aw, nil
}

func (aw *AsyncWorkerImpl) RegisterHandler(contract string, event string, handler common.TaskHandler) {
	if contract == "" || event == "" {
		aw.log.Warn("RegisterHandler require contract and event as parameters.")
		return
	}

	aw.mutex.Lock()
	defer aw.mutex.Unlock()
	// 先查看method是否合法
	if aw.methods == nil {
		aw.methods = make(map[string]map[string]common.TaskHandler)
	}
	methodMap, ok := aw.methods[contract]
	if !ok {
		methodMap = make(map[string]common.TaskHandler)
		aw.methods[contract] = methodMap
	}
	_, ok = methodMap[event]
	if ok {
		aw.log.Warn("async task method exists", "contract", contract, "event", event)
		return
	}
	methodMap[event] = handler
	// 注册event订阅，以区块粒度，当上链时触发事件调用
	aw.addBlockFilter(contract, event)
}

// addBlockFilter
func (aw *AsyncWorkerImpl) addBlockFilter(contract, event string) {
	if contract != "" {
		aw.filter.Contract += "|" + contract
	}
	if event != "" {
		aw.filter.EventName += "|" + event
	}
}

func (aw *AsyncWorkerImpl) StartAsyncTask() (err error) {
	// trick方法, 此处确保所有RegisterHandler处理完毕之后再起goroutine
	time.Sleep(time.Second * 10)

	// 尚未执行的存留异步任务查缺补漏
	cursor, err := aw.reloadCursor()
	if err != nil && err != emptyErr {
		aw.log.Error("couldn't do async task because of a reload cursor error")
		return err
	}
	// 若成功返回游标，则证明当前为重启异步任务逻辑，此时需要在事件订阅中明确游标
	if err == nil {
		bRange := protos.BlockRange{
			Start: fmt.Sprintf("%d", cursor.BlockHeight),
		}
		aw.filter.Range = &bRange
	}

	filterBuf, err := proto.Marshal(aw.filter)
	if err != nil {
		aw.log.Error("couldn't do async task because of a filter marshal error", "err", err)
		return err
	}

	// encfunc 提供iter.Data()对应的序列化方法, iter提供指向固定filter的迭代器
	encodeFunc, iter, err := aw.router.Subscribe(eventType, filterBuf)
	if err != nil {
		aw.log.Error("couldn't do async task because of a subscribe error", "err", err)
		return err
	}

	go func() {
		for {
			select {
			case <-aw.close:
				iter.Close()
				aw.log.Warn("async task loop shut down.")
				return
			}
		}
	}()

	go func() {
		for iter.Next() {
			payload := iter.Data()
			buf, err := encodeFunc(payload)
			if err != nil {
				aw.log.Error("couldn't do async task because of a encode error", "error", err)
				break
			}
			switch eventType {
			case protos.SubscribeType_BLOCK:
				var block protos.FilteredBlock
				err = proto.Unmarshal(buf, &block)
				if err != nil {
					aw.log.Error("couldn't do async task because of a block unmarshal error", "error", err)
					break
				}
				// 当且仅当断点有效，且当前高度为断点存储高度时，需要过滤部分已做异步任务
				if cursor != nil && block.BlockHeight == cursor.BlockHeight {
					aw.doAsyncTasks(block.Txs, block.BlockHeight, cursor)
					continue
				}
				aw.doAsyncTasks(block.Txs, block.BlockHeight, nil)
			}
		}
	}()
	return
}

func (aw *AsyncWorkerImpl) doAsyncTasks(txs []*protos.FilteredTransaction, height int64, cursor *asyncWorkerCursor) error {
	for index, tx := range txs {
		if tx.Events == nil {
			continue
		}
		// 断点之前的tx不需要再次执行了
		if cursor != nil && int64(index) < cursor.TxIndex {
			continue
		}
		for eventIndex, event := range tx.Events {
			// 断点之前的tx不需要再次执行了
			if cursor != nil && int64(index) == cursor.TxIndex && int64(eventIndex) < cursor.EventIndex {
				continue
			}
			handler, err := aw.getAsyncTask(event.Contract, event.Name)
			if err != nil {
				aw.log.Error("getAsyncTask meets error", "err", err)
				continue
			}
			ctx := newTaskContextImpl(event.Body)
			err = handler(ctx)
			if err != nil {
				aw.log.Error("do async task error", "err", err, "contract", event.Contract, "event", event.Name)
				continue
			}
			// 执行完毕后进行持久化
			cursor := asyncWorkerCursor{
				BlockHeight: height,
				TxIndex:     int64(index),
				EventIndex:  int64(eventIndex),
			}
			cursorBuf, err := json.Marshal(cursor)
			if err != nil {
				aw.log.Warn("marshal cursor failed when doAsyncTasks", "err", err)
				return err
			}
			aw.finishTable.Put([]byte(aw.bcname), cursorBuf)
		}
	}
	return nil
}

// reload 从上一次执行恢复，需要在断点处开始无缺漏的执行到当前高度，后在启动新的订阅协程
func (aw *AsyncWorkerImpl) reloadCursor() (*asyncWorkerCursor, error) {
	buf, err := aw.finishTable.Get([]byte(aw.bcname))
	if err != nil && def.NormalizedKVError(err) == def.ErrKVNotFound {
		return nil, emptyErr
	}
	if err != nil {
		aw.log.Error("get cursor failed when reloadCursor", "err", err)
		return nil, err
	}
	var cursor asyncWorkerCursor
	err = json.Unmarshal(buf, &cursor)
	if err != nil {
		return nil, err
	}
	if cursor.BlockHeight <= 0 {
		return nil, cursorErr
	}
	return &cursor, nil
}

func (aw *AsyncWorkerImpl) getAsyncTask(contract, event string) (common.TaskHandler, error) {
	// 串行调用，无需锁
	if contract == "" {
		return nil, fmt.Errorf("contract cannot be empty")
	}
	contractMap, ok := aw.methods[contract]
	if !ok {
		return nil, fmt.Errorf("async contract '%s' not found", contract)
	}
	handler, ok := contractMap[event]
	if !ok {
		return nil, fmt.Errorf("kernel method '%s' for '%s' not exists", event, contract)
	}
	return handler, nil
}

func (aw *AsyncWorkerImpl) Stop() {
	aw.close <- struct{}{}
	aw.baseDB.Close()
}

type asyncWorkerCursor struct {
	BlockHeight int64 `json:"block_height"`
	TxIndex     int64 `json:"tx_index"`
	EventIndex  int64 `json:"event_index"`
}
