// package txn deals with tx data
package tx

import (
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/pb"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

type Tx struct {
	log logs.Logger
	ldb kvdb.Database
	unconfirmedTable kvdb.Database
	unconfirmTxAmount int64
	unconfirmTxInMem  *sync.Map
	avgDelay int64
}

// Transaction is the internal represents of transaction
type Transaction struct {
	*pb.Transaction
}

func NewTx(lctx *def.LedgerCtx, stateDB kvdb.DataBase) (*Tx, error){
	return &Tx{
		log: lctx.XLog,
		ldb: stateDB,
		unconfirmedTable: kvdb.NewTable(baseDB, pb.UnconfirmedTablePrefix),
		unconfirmTxInMem: &sync.Map{},
	}, nil
}

// 生成奖励TX
func GenerateAwardTx(address []byte, awardAmount string, desc []byte) (Transaction, error) {
	utxoTx := &pb.Transaction{Version: TxVersion}
	amount := big.NewInt(0)
	amount.SetString(awardAmount, 10) // 10进制转换大整数
	if amount.Cmp(big.NewInt(0)) < 0 {
		return nil, ErrNegativeAmount
	}
	txOutput := &pb.TxOutput{}
	txOutput.ToAddr = []byte(address)
	txOutput.Amount = amount.Bytes()
	utxoTx.TxOutputs = append(utxoTx.TxOutputs, txOutput)
	utxoTx.Desc = desc
	utxoTx.Coinbase = true
	utxoTx.Timestamp = time.Now().UnixNano()
	utxoTx.Txid, _ = txhash.MakeTransactionID(utxoTx)
	return utxoTx, nil
}

// 生成只有Desc的空交易
func GenerateEmptyTx(desc []byte) (Transaction, error) {
	utxoTx := &pb.Transaction{Version: TxVersion}
	utxoTx.Desc = desc
	utxoTx.Timestamp = time.Now().UnixNano()
	txid, err := txhash.MakeTransactionID(utxoTx)
	utxoTx.Txid = txid
	utxoTx.Autogen = true
	return utxoTx, err
}

// 通过创世块配置生成创世区块交易
func GenerateRootTx(js []byte) (Transaction, error) {
	jsObj := &RootJSON{}
	jsErr := json.Unmarshal(js, jsObj)
	if jsErr != nil {
		return nil, jsErr
	}
	utxoTx := &pb.Transaction{Version: RootTxVersion}
	for _, pd := range jsObj.Predistribution {
		amount := big.NewInt(0)
		amount.SetString(pd.Quota, 10) // 10进制转换大整数
		if amount.Cmp(big.NewInt(0)) < 0 {
			return nil, ErrNegativeAmount
		}
		txOutput := &pb.TxOutput{}
		txOutput.ToAddr = []byte(pd.Address)
		txOutput.Amount = amount.Bytes()
		utxoTx.TxOutputs = append(utxoTx.TxOutputs, txOutput)
	}
	utxoTx.Desc = js
	utxoTx.Coinbase = true
	utxoTx.Txid, _ = txhash.MakeTransactionID(utxoTx)
	return utxoTx, nil
}

func ParseContractTransferRequest(requests []*pb.InvokeRequest) (string, *big.Int, error) {
	// found is the flag of whether the contract already carries the amount parameter
	var found bool
	amount := new(big.Int)
	var contractName string
	for _, req := range requests {
		amountstr := req.GetAmount()
		if amountstr == "" {
			continue
		}
		if found {
			return "", nil, errors.New("duplicated contract transfer amount")
		}
		_, ok := amount.SetString(amountstr, 10)
		if !ok {
			return "", nil, errors.New("bad amount in request")
		}
		found = true
		contractName = req.GetContractName()
	}
	return contractName, amount, nil
}


// QueryTx 查询一笔交易，从unconfirm表中查询
func (t *Tx) QueryTx(txid []byte) (*pb.Transaction, error) {
	pbBuf, findErr := t.unconfirmedTable.Get(txid)
	if findErr != nil {
		if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
			return nil, ErrTxNotFound
		}
		t.log.Warn("unexpected leveldb error, when do QueryTx, it may corrupted.", "findErr", findErr)
		return nil, findErr
	}
	tx := &pb.Transaction{}
	pbErr := proto.Unmarshal(pbBuf, tx)
	if pbErr != nil {
		t.log.Warn("failed to unmarshal tx", "pbErr", pbErr)
		return nil, pbErr
	}
	return tx, nil
}

// GetUnconfirmedTx 挖掘一批unconfirmed的交易打包，返回的结果要保证是按照交易执行的先后顺序
// maxSize: 打包交易最大的长度（in byte）, -1 表示不限制
func (t *Tx) GetUnconfirmedTx(dedup bool) ([]*pb.Transaction, error) {
	var selectedTxs []*pb.Transaction
	txMap, txGraph, _, loadErr := t.sortUnconfirmedTx()
	if loadErr != nil {
		return nil, loadErr
	}
	// 拓扑排序，输出的顺序是被依赖的在前，依赖方在后
	outputTxList, unexpectedCyclic, _ := TopSortDFS(txGraph)
	if unexpectedCyclic { // 交易之间检测出了环形的依赖关系
		t.log.Warn("transaction conflicted", "unexpectedCyclic", unexpectedCyclic)
		return nil, ErrUnexpected
	}
	for _, txid := range outputTxList {
		//todo dedup目前调用都是传false可以去掉了?
		if dedup && ledger.ledger.IsTxInTrunk([]byte(txid)) {
			continue
		}
		selectedTxs = append(selectedTxs, txMap[txid])
	}
	return selectedTxs, nil
}

// 加载所有未确认的订单表到内存
// 参数: dedup : true-删除已经确认tx, false-保留已经确认tx
// 返回: txMap : txid -> Transaction
//        txGraph:  txid ->  [依赖此txid的tx]
func (t *Tx) sortUnconfirmedTx() (map[string]*pb.Transaction, TxGraph, map[string]bool, error) {
	// 构造反向依赖关系图, key是被依赖的交易
	txMap := map[string]*pb.Transaction{}
	delayedTxMap := map[string]bool{}
	txGraph := TxGraph{}
	t.unconfirmTxInMem.Range(func(k, v interface{}) bool {
		txid := k.(string)
		txMap[txid] = v.(*pb.Transaction)
		txGraph[txid] = []string{}
		return true
	})
	var totalDelay int64
	now := time.Now().UnixNano()
	for txID, tx := range txMap {
		txDelay := (now - tx.ReceivedTimestamp)
		totalDelay += txDelay
		if uint32(txDelay/1e9) > t.maxConfirmedDelay {
			delayedTxMap[txID] = true
		}
		for _, refTx := range tx.TxInputs {
			refTxID := string(refTx.RefTxid)
			if _, exist := txMap[refTxID]; !exist {
				// 说明引用的tx不是在unconfirm里面
				continue
			}
			txGraph[refTxID] = append(txGraph[refTxID], txID)
		}
		for _, txIn := range tx.TxInputsExt {
			refTxID := string(txIn.RefTxid)
			if _, exist := txMap[refTxID]; !exist {
				continue
			}
			txGraph[refTxID] = append(txGraph[refTxID], txID)
		}
	}
	txMapSize := int64(len(txMap))
	if txMapSize > 0 {
		avgDelay := totalDelay / txMapSize //平均unconfirm滞留时间
		microSec := avgDelay / 1e6
		t.log.Info("average unconfirm delay", "micro-senconds", microSec, "count", txMapSize)
		t.avgDelay = microSec
	}
	t.unconfirmTxAmount = txMapSize
	return txMap, txGraph, delayedTxMap, nil
}


//从disk还原unconfirm表到内存, 初始化的时候
func (t *Tx) loadUnconfirmedTxFromDisk() error {
	iter := t.ldb.NewIteratorWithPrefix([]byte(pb.UnconfirmedTablePrefix))
	defer iter.Release()
	count := 0
	for iter.Next() {
		rawKey := iter.Key()
		txid := string(rawKey[1:])
		t.log.Trace("  load unconfirmed tx from db", "txid", fmt.Sprintf("%x", txid))
		txBuf := iter.Value()
		tx := &pb.Transaction{}
		pbErr := proto.Unmarshal(txBuf, tx)
		if pbErr != nil {
			return pbErr
		}
		t.unconfirmTxInMem.Store(txid, tx)
		count++
	}
	t.unconfirmTxAmount = int64(count)
	return nil
}