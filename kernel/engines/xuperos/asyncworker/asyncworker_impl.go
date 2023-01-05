package asyncworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto" //nolint:staticcheck

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
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

	emptyErr        = errors.New("Haven't store cursor before")
	cursorErr       = errors.New("DB stored an invalid cursor")
	emptyDBErr      = errors.New("Haven't get valid db")
	emptyCounterErr = errors.New("Haven't get valid router")
)

type AsyncWorkerImpl struct {
	bcname  string
	mutex   sync.Mutex
	methods map[string]map[string]common.TaskHandler // 句柄存储

	filter      *protos.BlockFilter // 订阅event事件时的筛选正则
	router      *event.Router       // 通过router进行事件订阅
	finishTable kvdb.Database       //保存临时的block区块

	close chan struct{}

	log logs.Logger
}

func NewAsyncWorkerImpl(bcName string, e common.Engine, baseDB kvdb.Database) (*AsyncWorkerImpl, error) {
	if baseDB == nil {
		e.Context().XLog.Error("NewAsyncWorkerImpl error, baseDB empty")
		return nil, emptyDBErr
	}
	aw := &AsyncWorkerImpl{
		filter: &protos.BlockFilter{
			Bcname:   bcName,
			Contract: `^\$`,
		},
		close:  make(chan struct{}, 1),
		router: event.NewRouter(e),
		log:    e.Context().XLog,
	}
	aw.finishTable = kvdb.NewTable(baseDB, FinishTablePrefix)
	return aw, nil
}

func (aw *AsyncWorkerImpl) RegisterHandler(contract string, event string, handler common.TaskHandler) {
	if contract == "" || event == "" {
		aw.log.Warn("RegisterHandler require contract and event as parameters.")
		return
	}
	if !strings.HasPrefix(contract, "$") {
		aw.log.Error("RegisterHandler require contract has prefix $, refuse register.")
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

func (aw *AsyncWorkerImpl) Start() (err error) {
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

	if aw.router == nil {
		return emptyCounterErr
	}

	// encfunc 提供iter.Data()对应的序列化方法, iter提供指向固定filter的迭代器
	_, iter, err := aw.router.Subscribe(eventType, filterBuf)
	if err != nil {
		aw.log.Error("couldn't do async task because of a subscribe error", "err", err)
		return err
	}

	go func() {
		<-aw.close
		iter.Close()
		aw.log.Warn("async task loop shut down.")
	}()

	go func() {
		for iter.Next() {
			payload := iter.Data()
			block, ok := payload.(*protos.FilteredBlock)
			if !ok {
				aw.log.Error("couldn't do async task because of a transfer error")
				break
			}
			if eventType != protos.SubscribeType_BLOCK {
				aw.log.Error("couldn't do async task because of eventType error", "have", eventType, "want", protos.SubscribeType_BLOCK)
				break
			}
			// 当且仅当断点有效，且当前高度为断点存储高度时，需要过滤部分已做异步任务
			var curBlockCursor *asyncWorkerCursor
			if cursor != nil && block.BlockHeight == cursor.BlockHeight {
				curBlockCursor = cursor
			}
			// TODO: deal with error
			_ = aw.doAsyncTasks(block.Txs, block.BlockHeight, curBlockCursor)
		}
	}()
	return
}

func (aw *AsyncWorkerImpl) doAsyncTasks(txs []*protos.FilteredTransaction, height int64, cursor *asyncWorkerCursor) error {
	var lastTxIndex, lastEventIndex int64
	for index, tx := range txs {
		lastTxIndex = int64(index)
		if tx.Events == nil {
			continue
		}
		// 断点之前的tx不需要再次执行了
		if cursor != nil && int64(index) < cursor.TxIndex {
			continue
		}
		for eventIndex, event := range tx.Events {
			lastEventIndex = int64(eventIndex)
			// 断点之前的tx不需要再次执行了
			if cursor != nil && int64(index) == cursor.TxIndex && int64(eventIndex) <= cursor.EventIndex {
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
			aw.log.Info("do async task success", "contract", event.Contract, "event", event.Name,
				"txIndex", index, "eventIndex", eventIndex)

			// 执行完毕后进行持久化
			newCursor := asyncWorkerCursor{
				BlockHeight: height,
				TxIndex:     int64(index),
				EventIndex:  int64(eventIndex),
			}
			_ = aw.storeCursor(newCursor) // ignore error which logged inside method
		}
	}
	// 该block已经处理完毕，此时需要记录到游标里，避免后续事件遍历负担
	newCursor := asyncWorkerCursor{
		BlockHeight: height,
		TxIndex:     lastTxIndex,
		EventIndex:  lastEventIndex,
	}
	return aw.storeCursor(newCursor)
}

func (aw *AsyncWorkerImpl) storeCursor(cursor asyncWorkerCursor) error {
	cursorBuf, err := json.Marshal(cursor)
	if err != nil {
		aw.log.Warn("marshal cursor failed when storeCursor", "err", err, "cursor", cursor)
		return err
	}
	err = aw.finishTable.Put([]byte(aw.bcname), cursorBuf)
	if err != nil {
		aw.log.Warn("finishTable put data error when storeCursor", "err", err, "bcname", aw.bcname, "cursor", cursor)
		return err
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
	close(aw.close)
	aw.finishTable.Close()
}

type asyncWorkerCursor struct {
	BlockHeight int64 `json:"block_height,required"` //nolint:staticcheck
	TxIndex     int64 `json:"tx_index,required"`     //nolint:staticcheck
	EventIndex  int64 `json:"event_index,required"`  //nolint:staticcheck
}
