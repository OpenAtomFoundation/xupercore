// package txn deals with tx data
package tx

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

var (
	ErrNegativeAmount = errors.New("amount in transaction can not be negative number")
	ErrTxNotFound     = errors.New("transaction not found")
	ErrUnexpected     = errors.New("this is a unexpected error")
)

const (
	TxVersion                = 1
	RootTxVersion            = 0
	DefaultMaxConfirmedDelay = 300
)

type Tx struct {
	log               logs.Logger
	ldb               kvdb.Database
	unconfirmedTable  kvdb.Database
	UnconfirmTxAmount int64
	// UnconfirmTxInMem  *sync.Map // 使用新版 mempool 就不用这个字段了。
	AvgDelay          int64
	ledger            *ledger.Ledger
	maxConfirmedDelay uint32

	Mempool *Mempool
}

// RootJSON xuper.json对应的struct，目前先只写了utxovm关注的字段
type RootJSON struct {
	Version   string `json:"version"`
	Consensus struct {
		Miner string `json:"miner"`
	} `json:"consensus"`
	Predistribution []struct {
		Address string `json:"address"`
		Quota   string `json:"quota"`
	} `json:"predistribution"`
}

func NewTx(sctx *context.StateCtx, stateDB kvdb.Database) (*Tx, error) {
	sctx.XLog.Info("NEW MEMPOOL !!!!")

	tx := &Tx{
		log:              sctx.XLog,
		ldb:              stateDB,
		unconfirmedTable: kvdb.NewTable(stateDB, pb.UnconfirmedTablePrefix),
		// UnconfirmTxInMem:  &sync.Map{},
		ledger:            sctx.Ledger,
		maxConfirmedDelay: DefaultMaxConfirmedDelay,
	}
	m := NewMempool(tx)
	tx.Mempool = m
	return tx, nil
}

// 生成奖励TX
func GenerateAwardTx(address, awardAmount string, desc []byte) (*pb.Transaction, error) {
	utxoTx := &pb.Transaction{Version: TxVersion}
	amount := big.NewInt(0)
	amount.SetString(awardAmount, 10) // 10进制转换大整数
	if amount.Cmp(big.NewInt(0)) < 0 {
		return nil, ErrNegativeAmount
	}
	txOutput := &protos.TxOutput{}
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
func GenerateEmptyTx(desc []byte) (*pb.Transaction, error) {
	utxoTx := &pb.Transaction{Version: TxVersion}
	utxoTx.Desc = desc
	utxoTx.Timestamp = time.Now().UnixNano()
	txid, err := txhash.MakeTransactionID(utxoTx)
	utxoTx.Txid = txid
	utxoTx.Autogen = true
	return utxoTx, err
}

// 生成只有读写集的空交易
func GenerateAutoTxWithRWSets(inputs []*protos.TxInputExt, outputs []*protos.TxOutputExt) (*pb.Transaction, error) {

	tx := &pb.Transaction{
		Coinbase:     false,
		Nonce:        utils.GenNonce(),
		Timestamp:    time.Now().UnixNano(),
		Version:      TxVersion,
		Autogen:      true,
		TxInputsExt:  inputs,
		TxOutputsExt: outputs,
	}

	txid, err := txhash.MakeTransactionID(tx)

	tx.Txid = txid

	return tx, err
}

// 通过创世块配置生成创世区块交易
func GenerateRootTx(js []byte) (*pb.Transaction, error) {
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
		txOutput := &protos.TxOutput{}
		txOutput.ToAddr = []byte(pd.Address)
		txOutput.Amount = amount.Bytes()
		utxoTx.TxOutputs = append(utxoTx.TxOutputs, txOutput)
	}
	utxoTx.Desc = js
	utxoTx.Coinbase = true
	utxoTx.Txid, _ = txhash.MakeTransactionID(utxoTx)
	return utxoTx, nil
}

func ParseContractTransferRequest(requests []*protos.InvokeRequest) (string, *big.Int, error) {
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
		if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
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
func (t *Tx) GetUnconfirmedTx(dedup bool, sizeLimit int) ([]*pb.Transaction, error) {
	result := make([]*pb.Transaction, 0, 100)

	f := func(tx *pb.Transaction) bool {
		if dedup && t.ledger.IsTxInTrunk([]byte(tx.Txid)) {
			return true
		}

		if sizeLimit > 0 {
			size := proto.Size(tx)
			if size > sizeLimit {
				return false
			}
			sizeLimit -= size
		}
		result = append(result, tx)
		return true
	}

	t.Mempool.Range(f)
	t.UnconfirmTxAmount = int64(len(result))
	return result, nil
}

// SortUnconfirmedTx 加载所有未确认的订单表到内存
// 参数: dedup : true-删除已经确认tx, false-保留已经确认tx
// 返回: txMap : txid -> Transaction
//        txGraph:  txid ->  [依赖此txid的tx]
func (t *Tx) SortUnconfirmedTx(sizeLimit int) ([]*pb.Transaction, map[string]*pb.Transaction, error) {
	// 构造反向依赖关系图, key是被依赖的交易
	// txMap := map[string]*pb.Transaction{}
	delayedTxMap := map[string]*pb.Transaction{}
	// txGraph := TxGraph{}

	result := make([]*pb.Transaction, 0, 100)

	var totalDelay int64
	now := time.Now().UnixNano()

	f := func(tx *pb.Transaction) bool {
		txDelay := (now - tx.ReceivedTimestamp)
		totalDelay += txDelay
		if uint32(txDelay/1e9) > t.maxConfirmedDelay {
			delayedTxMap[string(tx.GetTxid())] = tx
		}
		if sizeLimit > 0 {
			size := proto.Size(tx)
			if size > sizeLimit {
				return false
			}
			sizeLimit -= size
		}

		result = append(result, tx)
		return true
	}

	t.Mempool.Range(f)
	txMapSize := int64(len(result))
	if txMapSize > 0 {
		avgDelay := totalDelay / txMapSize //平均unconfirm滞留时间
		microSec := avgDelay / 1e6
		t.log.Info("average unconfirm delay", "micro-senconds", microSec, "count", txMapSize)
		t.AvgDelay = microSec
	}
	t.UnconfirmTxAmount = txMapSize
	return result, delayedTxMap, nil
}

//从disk还原unconfirm表到内存, 初始化的时候
func (t *Tx) LoadUnconfirmedTxFromDisk() error {
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
		// t.UnconfirmTxInMem.Store(txid, tx)
		t.Mempool.RetrieveTx(tx) // todo err
		count++
	}
	t.UnconfirmTxAmount = int64(count)
	return nil
}

func (t *Tx) SetMaxConfirmedDelay(seconds uint32) {
	t.maxConfirmedDelay = seconds
	t.log.Info("set max confirmed delay of tx", "seconds", seconds)
}
