// Package utxo is the key part of XuperChain, this module keeps all Unspent Transaction Outputs.
//
// For a transaction, the UTXO checks the tokens used in reference transactions are unspent, and
// reject the transaction if the initiator doesn't have enough tokens.
// UTXO also checks the signature and permission of transaction members.
package utxo

import (
	"bytes"
	"container/list"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/meta"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/xmodel"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/permission/acl"
	aclu "github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"github.com/xuperchain/xupercore/lib/cache"
	crypto_base "github.com/xuperchain/xupercore/lib/crypto/client/base"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

// 常用VM执行错误码
var (
	ErrNoEnoughUTXO        = errors.New("no enough money(UTXO) to start this transaction")
	ErrUTXONotFound        = errors.New("this utxo can not be found")
	ErrInputOutputNotEqual = errors.New("input's amount is not equal to output's")
	ErrUnexpected          = errors.New("this is a unexpected error")
	ErrNegativeAmount      = errors.New("amount in transaction can not be negative number")
	ErrUTXOFrozen          = errors.New("utxo is still frozen")
	ErrUTXODuplicated      = errors.New("found duplicated utxo in same tx")
)

// package constants
const (
	UTXOLockExpiredSecond = 60
	LatestBlockKey        = "pointer"
	UTXOCacheSize         = 1000
	OfflineTxChanBuffer   = 100000

	// TxVersion 为所有交易使用的版本
	TxVersion = 3

	FeePlaceholder = "$"
	UTXOTotalKey   = "xtotal"
)

type TxLists []*pb.Transaction

type UtxoVM struct {
	metaHandle        *meta.Meta
	ldb               kvdb.Database
	Mutex             *sync.RWMutex // utxo leveldb表读写锁
	MutexMem          *sync.Mutex   // 内存锁定状态互斥锁
	SpLock            *SpinLock     // 自旋锁,根据交易涉及的utxo和改写的变量
	mutexBalance      *sync.Mutex   // 余额Cache锁
	lockKeys          map[string]*UtxoLockItem
	lockKeyList       *list.List // 按锁定的先后顺序，方便过期清理
	lockExpireTime    int        // 临时锁定的最长时间
	UtxoCache         *UtxoCache
	log               logs.Logger
	ledger            *ledger.Ledger           // 引用的账本对象
	utxoTable         kvdb.Database            // utxo表
	OfflineTxChan     chan []*pb.Transaction   // 未确认tx的通知chan
	PrevFoundKeyCache *cache.LRUCache          // 上一次找到的可用utxo key，用于加速GenerateTx
	utxoTotal         *big.Int                 // 总资产
	cryptoClient      crypto_base.CryptoClient // 加密实例
	ModifyBlockAddr   string                   // 可修改区块链的监管地址
	BalanceCache      *cache.LRUCache          //余额cache,加速GetBalance查询
	CacheSize         int                      //记录构造utxo时传入的cachesize
	BalanceViewDirty  map[string]int           //balanceCache 标记dirty: addr -> sequence of view
	unconfirmTxInMem  *sync.Map                //未确认Tx表的内存镜像
	unconfirmTxAmount int64                    // 未确认的Tx数目，用于监控
	bcname            string
}

// InboundTx is tx wrapper
type InboundTx struct {
	tx    *pb.Transaction
	txBuf []byte
}

type UtxoLockItem struct {
	timestamp int64
	holder    *list.Element
}

type contractChainCore struct {
	*acl.Manager // ACL manager for read/write acl table
	*UtxoVM
	*ledger.Ledger
}

func GenUtxoKey(addr []byte, txid []byte, offset int32) string {
	return fmt.Sprintf("%s_%x_%d", addr, txid, offset)
}

// GenUtxoKeyWithPrefix generate UTXO key with given prefix
func GenUtxoKeyWithPrefix(addr []byte, txid []byte, offset int32) string {
	baseUtxoKey := GenUtxoKey(addr, txid, offset)
	return pb.UTXOTablePrefix + baseUtxoKey
}

// checkInputEqualOutput 校验交易的输入输出是否相等
func (uv *UtxoVM) CheckInputEqualOutput(tx *pb.Transaction) error {
	// first check outputs
	outputSum := big.NewInt(0)
	for _, txOutput := range tx.TxOutputs {
		amount := big.NewInt(0)
		amount.SetBytes(txOutput.Amount)
		if amount.Cmp(big.NewInt(0)) < 0 {
			return ErrNegativeAmount
		}
		outputSum.Add(outputSum, amount)
	}
	// then we check inputs
	inputSum := big.NewInt(0)
	curLedgerHeight := uv.ledger.GetMeta().TrunkHeight
	utxoDedup := map[string]bool{}
	for _, txInput := range tx.TxInputs {
		addr := txInput.FromAddr
		txid := txInput.RefTxid
		offset := txInput.RefOffset
		utxoKey := GenUtxoKey(addr, txid, offset)
		if utxoDedup[utxoKey] {
			uv.log.Warn("found duplicated utxo in same tx", "utxoKey", utxoKey, "txid", utils.F(tx.Txid))
			return ErrUTXODuplicated
		}
		utxoDedup[utxoKey] = true
		var amountBytes []byte
		var frozenHeight int64
		uv.UtxoCache.Lock()
		if l2Cache, exist := uv.UtxoCache.All[string(addr)]; exist {
			uItem := l2Cache[pb.UTXOTablePrefix+utxoKey]
			if uItem != nil {
				amountBytes = uItem.Amount.Bytes()
				frozenHeight = uItem.FrozenHeight
			}
		}
		uv.UtxoCache.Unlock()
		if amountBytes == nil {
			uBinary, findErr := uv.utxoTable.Get([]byte(utxoKey))
			if findErr != nil {
				if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
					uv.log.Info("not found utxo key:", "utxoKey", utxoKey)
					return ErrUTXONotFound
				}
				uv.log.Warn("unexpected leveldb error when do checkInputEqualOutput", "findErr", findErr)
				return findErr
			}
			uItem := &UtxoItem{}
			uErr := uItem.Loads(uBinary)
			if uErr != nil {
				return uErr
			}
			amountBytes = uItem.Amount.Bytes()
			frozenHeight = uItem.FrozenHeight
		}
		amount := big.NewInt(0)
		amount.SetBytes(amountBytes)
		if !bytes.Equal(amountBytes, txInput.Amount) {
			txInputAmount := big.NewInt(0)
			txInputAmount.SetBytes(txInput.Amount)
			uv.log.Warn("unexpected error, txInput amount missmatch utxo amount",
				"in_utxo", amount, "txInputAmount", txInputAmount, "txid", utils.F(tx.Txid), "reftxid", utils.F(txid))
			return ErrUnexpected
		}
		if frozenHeight > curLedgerHeight || frozenHeight == -1 {
			uv.log.Warn("this utxo still be frozen", "frozenHeight", frozenHeight, "ledgerHeight", curLedgerHeight)
			return ErrUTXOFrozen
		}
		inputSum.Add(inputSum, amount)
	}
	if inputSum.Cmp(outputSum) == 0 {
		return nil
	}
	if inputSum.Cmp(big.NewInt(0)) == 0 && tx.Coinbase {
		// coinbase交易，输入输出不必相等, 特殊处理
		return nil
	}
	uv.log.Warn("input != output", "inputSum", inputSum, "outputSum", outputSum)
	return ErrInputOutputNotEqual
}

// utxo是否处于临时锁定状态
func (uv *UtxoVM) isLocked(utxoKey []byte) bool {
	uv.MutexMem.Lock()
	defer uv.MutexMem.Unlock()
	_, exist := uv.lockKeys[string(utxoKey)]
	return exist
}

// 解锁utxo key
func (uv *UtxoVM) UnlockKey(utxoKey []byte) {
	uv.MutexMem.Lock()
	defer uv.MutexMem.Unlock()
	uv.log.Trace("    unlock utxo key", "key", string(utxoKey))
	lockItem := uv.lockKeys[string(utxoKey)]
	if lockItem != nil {
		uv.lockKeyList.Remove(lockItem.holder)
		delete(uv.lockKeys, string(utxoKey))
	}
}

// 试图临时锁定utxo, 返回是否锁定成功
func (uv *UtxoVM) tryLockKey(key []byte) bool {
	uv.MutexMem.Lock()
	defer uv.MutexMem.Unlock()
	if _, exist := uv.lockKeys[string(key)]; !exist {
		holder := uv.lockKeyList.PushBack(key)
		uv.lockKeys[string(key)] = &UtxoLockItem{timestamp: time.Now().Unix(), holder: holder}
		uv.log.Trace("  lock utxo key", "key", string(key))
		return true
	}
	return false
}

// 清理过期的utxo锁定
func (uv *UtxoVM) clearExpiredLocks() {
	uv.MutexMem.Lock()
	defer uv.MutexMem.Unlock()
	now := time.Now().Unix()
	for {
		topItem := uv.lockKeyList.Front()
		if topItem == nil {
			break
		}
		topKey := topItem.Value.([]byte)
		lockItem, exist := uv.lockKeys[string(topKey)]
		if !exist {
			uv.lockKeyList.Remove(topItem)
		} else if lockItem.timestamp+int64(uv.lockExpireTime) <= now {
			uv.lockKeyList.Remove(topItem)
			delete(uv.lockKeys, string(topKey))
		} else {
			break
		}
	}
}

// NewUtxoVM 构建一个UtxoVM对象
//   @param ledger 账本对象
//   @param store path, utxo 数据的保存路径
//   @param xlog , 日志handler
func NewUtxo(sctx *context.StateCtx, metaHandle *meta.Meta, stateDB kvdb.Database) (*UtxoVM, error) {
	return MakeUtxo(sctx, metaHandle, UTXOCacheSize, UTXOLockExpiredSecond, stateDB)
}

// MakeUtxoVM 这个函数比NewUtxoVM更加可订制化
func MakeUtxo(sctx *context.StateCtx, metaHandle *meta.Meta, cachesize, tmplockSeconds int,
	stateDB kvdb.Database) (*UtxoVM, error) {
	utxoMutex := &sync.RWMutex{}
	utxoVM := &UtxoVM{
		metaHandle:        metaHandle,
		ldb:               stateDB,
		Mutex:             utxoMutex,
		MutexMem:          &sync.Mutex{},
		SpLock:            NewSpinLock(),
		mutexBalance:      &sync.Mutex{},
		lockKeys:          map[string]*UtxoLockItem{},
		lockKeyList:       list.New(),
		lockExpireTime:    tmplockSeconds,
		log:               sctx.XLog,
		ledger:            sctx.Ledger,
		utxoTable:         kvdb.NewTable(stateDB, pb.UTXOTablePrefix),
		UtxoCache:         NewUtxoCache(cachesize),
		OfflineTxChan:     make(chan []*pb.Transaction, OfflineTxChanBuffer),
		PrevFoundKeyCache: cache.NewLRUCache(cachesize),
		utxoTotal:         big.NewInt(0),
		BalanceCache:      cache.NewLRUCache(cachesize),
		CacheSize:         cachesize,
		BalanceViewDirty:  map[string]int{},
		cryptoClient:      sctx.Crypt,
		bcname:            sctx.BCName,
	}

	utxoTotalBytes, findTotalErr := utxoVM.metaHandle.MetaTable.Get([]byte(UTXOTotalKey))
	if findTotalErr == nil {
		total := big.NewInt(0)
		total.SetBytes(utxoTotalBytes)
		utxoVM.utxoTotal = total
	} else {
		if def.NormalizedKVError(findTotalErr) != def.ErrKVNotFound {
			return nil, findTotalErr
		}
		utxoVM.utxoTotal = big.NewInt(0)
	}
	return utxoVM, nil
}

func (uv *UtxoVM) UpdateUtxoTotal(delta *big.Int, batch kvdb.Batch, inc bool) {
	if inc {
		uv.utxoTotal = uv.utxoTotal.Add(uv.utxoTotal, delta)
	} else {
		uv.utxoTotal = uv.utxoTotal.Sub(uv.utxoTotal, delta)
	}
	batch.Put(append([]byte(pb.MetaTablePrefix), []byte(UTXOTotalKey)...), uv.utxoTotal.Bytes())
}

// parseUtxoKeys extract (txid, offset) from key of utxo item
func (uv *UtxoVM) parseUtxoKeys(uKey string) ([]byte, int, error) {
	keyTuple := strings.Split(uKey[1:], "_") // [1:] 是为了剔除表名字前缀
	N := len(keyTuple)
	if N < 2 {
		uv.log.Warn("unexpected utxo key", "uKey", uKey)
		return nil, 0, ErrUnexpected
	}
	refTxid, err := hex.DecodeString(keyTuple[N-2])
	if err != nil {
		return nil, 0, err
	}
	offset, err := strconv.Atoi(keyTuple[N-1])
	if err != nil {
		return nil, 0, err
	}
	return refTxid, offset, nil
}

//SelectUtxos 选择足够的utxo
//输入: 转账人地址、公钥、金额、是否需要锁定utxo
//输出：选出的utxo、utxo keys、实际构成的金额(可能大于需要的金额)、错误码
func (uv *UtxoVM) SelectUtxos(fromAddr string, totalNeed *big.Int, needLock, excludeUnconfirmed bool) ([]*protos.TxInput, [][]byte, *big.Int, error) {
	if totalNeed.Cmp(big.NewInt(0)) == 0 {
		return nil, nil, big.NewInt(0), nil
	}
	curLedgerHeight := uv.ledger.GetMeta().TrunkHeight
	willLockKeys := make([][]byte, 0)
	foundEnough := false
	utxoTotal := big.NewInt(0)
	cacheKeys := map[string]bool{} // 先从cache里找找，不够再从leveldb找,因为leveldb prefix scan比较慢
	txInputs := []*protos.TxInput{}
	uv.clearExpiredLocks()
	uv.UtxoCache.Lock()
	if l2Cache, exist := uv.UtxoCache.Available[fromAddr]; exist {
		for uKey, uItem := range l2Cache {
			if uItem.FrozenHeight > curLedgerHeight || uItem.FrozenHeight == -1 {
				uv.log.Trace("utxo still frozen, skip it", "uKey", uKey, " fheight", uItem.FrozenHeight)
				continue
			}
			refTxid, offset, err := uv.parseUtxoKeys(uKey)
			if err != nil {
				return nil, nil, nil, err
			}
			if needLock {
				if uv.tryLockKey([]byte(uKey)) {
					willLockKeys = append(willLockKeys, []byte(uKey))
				} else {
					uv.log.Debug("can not lock the utxo key, conflict", "uKey", uKey)
					continue
				}
			} else if uv.isLocked([]byte(uKey)) {
				uv.log.Debug("skip locked utxo key", "uKey", uKey)
				continue
			}
			if excludeUnconfirmed { //必须依赖已经上链的tx的UTXO
				isOnChain := uv.ledger.IsTxInTrunk(refTxid)
				if !isOnChain {
					continue
				}
			}
			uv.UtxoCache.Use(fromAddr, uKey)
			utxoTotal.Add(utxoTotal, uItem.Amount)
			txInput := &protos.TxInput{
				RefTxid:      refTxid,
				RefOffset:    int32(offset),
				FromAddr:     []byte(fromAddr),
				Amount:       uItem.Amount.Bytes(),
				FrozenHeight: uItem.FrozenHeight,
			}
			txInputs = append(txInputs, txInput)
			cacheKeys[uKey] = true
			if utxoTotal.Cmp(totalNeed) >= 0 {
				foundEnough = true
				break
			}
		}
	}
	uv.UtxoCache.Unlock()
	if !foundEnough {
		// 底层key: table_prefix from_addr "_" txid "_" offset
		addrPrefix := pb.UTXOTablePrefix + fromAddr + "_"
		var middleKey []byte
		preFoundUtxoKey, mdOK := uv.PrevFoundKeyCache.Get(fromAddr)
		if mdOK {
			middleKey = preFoundUtxoKey.([]byte)
		}
		it := kvdb.NewQuickIterator(uv.ldb, []byte(addrPrefix), middleKey)
		defer it.Release()
		for it.Next() {
			key := append([]byte{}, it.Key()...)
			uBinary := it.Value()
			uItem := &UtxoItem{}
			uErr := uItem.Loads(uBinary)
			if uErr != nil {
				return nil, nil, nil, uErr
			}
			if _, inCache := cacheKeys[string(key)]; inCache {
				continue // cache已经命中了，跳过
			}
			if uItem.FrozenHeight > curLedgerHeight || uItem.FrozenHeight == -1 {
				uv.log.Trace("utxo still frozen, skip it", "key", string(key), "fheight", uItem.FrozenHeight)
				continue
			}
			refTxid, offset, err := uv.parseUtxoKeys(string(key))
			if err != nil {
				return nil, nil, nil, err
			}
			if needLock {
				if uv.tryLockKey(key) {
					willLockKeys = append(willLockKeys, key)
				} else {
					uv.log.Debug("can not lock the utxo key, conflict", "key", string(key))
					continue
				}
			} else if uv.isLocked(key) {
				uv.log.Debug("skip locked utxo key", "key", string(key))
				continue
			}
			if excludeUnconfirmed { //必须依赖已经上链的tx的UTXO
				isOnChain := uv.ledger.IsTxInTrunk(refTxid)
				if !isOnChain {
					continue
				}
			}
			txInput := &protos.TxInput{
				RefTxid:      refTxid,
				RefOffset:    int32(offset),
				FromAddr:     []byte(fromAddr),
				Amount:       uItem.Amount.Bytes(),
				FrozenHeight: uItem.FrozenHeight,
			}
			txInputs = append(txInputs, txInput)
			utxoTotal.Add(utxoTotal, uItem.Amount) // utxo累加
			if utxoTotal.Cmp(totalNeed) >= 0 {     // 找到了足够的utxo用于支付
				foundEnough = true
				uv.PrevFoundKeyCache.Add(fromAddr, key)
				break
			}
		}
		if it.Error() != nil {
			return nil, nil, nil, it.Error()
		}
	}
	if !foundEnough {
		for _, lk := range willLockKeys {
			uv.UnlockKey(lk)
		}
		return nil, nil, nil, ErrNoEnoughUTXO // 余额不足啦
	}
	return txInputs, willLockKeys, utxoTotal, nil
}

// addBalance 增加cache中的Balance
func (uv *UtxoVM) AddBalance(addr []byte, delta *big.Int) {
	uv.mutexBalance.Lock()
	defer uv.mutexBalance.Unlock()
	balance, hitCache := uv.BalanceCache.Get(string(addr))
	if hitCache {
		iBalance := balance.(*big.Int)
		iBalance.Add(iBalance, delta)
	} else {
		uv.BalanceViewDirty[string(addr)] = uv.BalanceViewDirty[string(addr)] + 1
	}
}

// subBalance 减少cache中的Balance
func (uv *UtxoVM) SubBalance(addr []byte, delta *big.Int) {
	uv.mutexBalance.Lock()
	defer uv.mutexBalance.Unlock()
	balance, hitCache := uv.BalanceCache.Get(string(addr))
	if hitCache {
		iBalance := balance.(*big.Int)
		iBalance.Sub(iBalance, delta)
	} else {
		uv.BalanceViewDirty[string(addr)] = uv.BalanceViewDirty[string(addr)] + 1
	}
}

//获得一个账号的余额，inLock表示在调用此函数时已经对uv.mutex加过锁了
func (uv *UtxoVM) GetBalance(addr string) (*big.Int, error) {
	cachedBalance, ok := uv.BalanceCache.Get(addr)
	if ok {
		uv.log.Debug("hit getbalance cache", "addr", addr)
		uv.mutexBalance.Lock()
		balanceCopy := big.NewInt(0).Set(cachedBalance.(*big.Int))
		uv.mutexBalance.Unlock()
		return balanceCopy, nil
	}
	addrPrefix := fmt.Sprintf("%s%s_", pb.UTXOTablePrefix, addr)
	utxoTotal := big.NewInt(0)
	uv.mutexBalance.Lock()
	myBalanceView := uv.BalanceViewDirty[addr]
	uv.mutexBalance.Unlock()
	it := uv.ldb.NewIteratorWithPrefix([]byte(addrPrefix))
	defer it.Release()
	for it.Next() {
		uBinary := it.Value()
		uItem := &UtxoItem{}
		uErr := uItem.Loads(uBinary)
		if uErr != nil {
			return nil, uErr
		}
		utxoTotal.Add(utxoTotal, uItem.Amount) // utxo累加
	}
	if it.Error() != nil {
		return nil, it.Error()
	}
	uv.mutexBalance.Lock()
	defer uv.mutexBalance.Unlock()
	if myBalanceView != uv.BalanceViewDirty[addr] {
		return utxoTotal, nil
	}
	_, exist := uv.BalanceCache.Get(addr)
	if !exist {
		//首次填充cache
		uv.BalanceCache.Add(addr, utxoTotal)
		delete(uv.BalanceViewDirty, addr)
	}
	balanceCopy := big.NewInt(0).Set(utxoTotal)
	return balanceCopy, nil
}

// Close 关闭utxo vm, 目前主要是关闭leveldb
func (uv *UtxoVM) Close() {
	uv.ldb.Close()
}

// GetTotal 返回当前vm的总资产
func (uv *UtxoVM) GetTotal() *big.Int {
	result := big.NewInt(0)
	result.SetBytes(uv.utxoTotal.Bytes())
	return result
}

// ScanWithPrefix 通过前缀获得一个连续读取的迭代器
func (uv *UtxoVM) ScanWithPrefix(prefix []byte) kvdb.Iterator {
	return uv.ldb.NewIteratorWithPrefix(prefix)
}

// RemoveUtxoCache 清理utxoCache
func (uv *UtxoVM) RemoveUtxoCache(address string, utxoKey string) {
	uv.log.Trace("RemoveUtxoCache", "address", address, "utxoKey", utxoKey)
	uv.UtxoCache.Remove(address, utxoKey)
}

// NewBatch return batch instance
func (uv *UtxoVM) NewBatch() kvdb.Batch {
	return uv.ldb.NewBatch()
}

// SetModifyBlockAddr set modified block addr
func (uv *UtxoVM) SetModifyBlockAddr(addr string) {
	uv.ModifyBlockAddr = addr
	uv.log.Info("set modified block addr", "addr", addr)
}

// GetAccountContracts get account contracts, return a slice of contract names
func (uv *UtxoVM) GetAccountContracts(account string) ([]string, error) {
	contracts := []string{}
	if aclu.IsAccount(account) != 1 {
		uv.log.Warn("GetAccountContracts valid account name error", "error", "account name is not valid")
		return nil, errors.New("account name is not valid")
	}
	prefKey := pb.ExtUtxoTablePrefix + string(xmodel.MakeRawKey(aclu.GetAccount2ContractBucket(), []byte(account+aclu.GetACLSeparator())))
	it := uv.ldb.NewIteratorWithPrefix([]byte(prefKey))
	defer it.Release()
	for it.Next() {
		key := string(it.Key())
		ret := strings.Split(key, aclu.GetACLSeparator())
		size := len(ret)
		if size >= 1 {
			contracts = append(contracts, ret[size-1])
		}
	}
	if it.Error() != nil {
		return nil, it.Error()
	}
	return contracts, nil
}

// QueryUtxoRecord query utxo record details
func (uv *UtxoVM) QueryUtxoRecord(accountName string, displayCount int64) (*pb.UtxoRecordDetail, error) {
	utxoRecordDetail := &pb.UtxoRecordDetail{}

	addrPrefix := fmt.Sprintf("%s%s_", pb.UTXOTablePrefix, accountName)
	it := uv.ldb.NewIteratorWithPrefix([]byte(addrPrefix))
	defer it.Release()

	openUtxoCount := big.NewInt(0)
	openUtxoAmount := big.NewInt(0)
	openItem := []*pb.UtxoKey{}
	lockedUtxoCount := big.NewInt(0)
	lockedUtxoAmount := big.NewInt(0)
	lockedItem := []*pb.UtxoKey{}
	frozenUtxoCount := big.NewInt(0)
	frozenUtxoAmount := big.NewInt(0)
	frozenItem := []*pb.UtxoKey{}

	for it.Next() {
		key := append([]byte{}, it.Key()...)
		utxoItem := new(UtxoItem)
		uErr := utxoItem.Loads(it.Value())
		if uErr != nil {
			continue
		}

		if uv.isLocked(key) {
			lockedUtxoCount.Add(lockedUtxoCount, big.NewInt(1))
			lockedUtxoAmount.Add(lockedUtxoAmount, utxoItem.Amount)
			if displayCount > 0 {
				lockedItem = append(lockedItem, MakeUtxoKey(key, utxoItem.Amount.String()))
				displayCount--
			}
			continue
		}
		if utxoItem.FrozenHeight > uv.ledger.GetMeta().GetTrunkHeight() || utxoItem.FrozenHeight == -1 {
			frozenUtxoCount.Add(frozenUtxoCount, big.NewInt(1))
			frozenUtxoAmount.Add(frozenUtxoAmount, utxoItem.Amount)
			if displayCount > 0 {
				frozenItem = append(frozenItem, MakeUtxoKey(key, utxoItem.Amount.String()))
				displayCount--
			}
			continue
		}
		openUtxoCount.Add(openUtxoCount, big.NewInt(1))
		openUtxoAmount.Add(openUtxoAmount, utxoItem.Amount)
		if displayCount > 0 {
			openItem = append(openItem, MakeUtxoKey(key, utxoItem.Amount.String()))
			displayCount--
		}
	}
	if it.Error() != nil {
		return utxoRecordDetail, it.Error()
	}

	utxoRecordDetail.OpenUtxo = &pb.UtxoRecord{
		UtxoCount:  openUtxoCount.String(),
		UtxoAmount: openUtxoAmount.String(),
		Item:       openItem,
	}
	utxoRecordDetail.LockedUtxo = &pb.UtxoRecord{
		UtxoCount:  lockedUtxoCount.String(),
		UtxoAmount: lockedUtxoAmount.String(),
		Item:       lockedItem,
	}
	utxoRecordDetail.FrozenUtxo = &pb.UtxoRecord{
		UtxoCount:  frozenUtxoCount.String(),
		UtxoAmount: frozenUtxoAmount.String(),
		Item:       frozenItem,
	}

	return utxoRecordDetail, nil
}

func (uv *UtxoVM) QueryAccountContainAK(address string) ([]string, error) {
	accounts := []string{}
	if aclu.IsAccount(address) != 0 {
		return accounts, errors.New("address is not valid")
	}
	prefixKey := pb.ExtUtxoTablePrefix + aclu.GetAK2AccountBucket() + "/" + address
	it := uv.ldb.NewIteratorWithPrefix([]byte(prefixKey))
	defer it.Release()
	for it.Next() {
		key := string(it.Key())
		ret := strings.Split(key, aclu.GetAKAccountSeparator())
		size := len(ret)
		if size >= 1 {
			accounts = append(accounts, ret[size-1])
		}
	}
	if it.Error() != nil {
		return []string{}, it.Error()
	}
	return accounts, nil
}

func MakeUtxoKey(key []byte, amount string) *pb.UtxoKey {
	keyTuple := bytes.Split(key[1:], []byte("_")) // [1:] 是为了剔除表名字前缀
	tmpUtxoKey := &pb.UtxoKey{
		RefTxid: string(keyTuple[len(keyTuple)-2]),
		Offset:  string(keyTuple[len(keyTuple)-1]),
		Amount:  amount,
	}

	return tmpUtxoKey
}
