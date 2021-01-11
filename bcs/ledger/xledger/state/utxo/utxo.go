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
	"github.com/xuperchain/xuperchain/core/common"
	xlog "github.com/xuperchain/xuperchain/core/common/log"
	"github.com/xuperchain/xuperchain/core/global"
	"math/big"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/permission/acl"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"github.com/xuperchain/xupercore/lib/cache"
	crypto_base "github.com/xuperchain/xupercore/lib/crypto/client/base"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

// 常用VM执行错误码
var (
	ErrNoEnoughUTXO        = errors.New("no enough money(UTXO) to start this transaction")
	ErrUTXONotFound        = errors.New("this utxo can not be found")
	ErrInputOutputNotEqual = errors.New("input's amount is not equal to output's")
	ErrTxNotFound          = errors.New("this tx can not be found in unconfirmed-table")
	ErrUnexpected          = errors.New("this is a unexpected error")
	ErrNegativeAmount      = errors.New("amount in transaction can not be negative number")
	ErrUTXOFrozen          = errors.New("utxo is still frozen")
	ErrTxSizeLimitExceeded = errors.New("tx size limit exceeded")
	ErrInvalidAutogenTx    = errors.New("found invalid autogen-tx")
	ErrUTXODuplicated      = errors.New("found duplicated utxo in same tx")
	ErrRWSetInvalid        = errors.New("RWSet of transaction invalid")
	ErrACLNotEnough        = errors.New("ACL not enough")
	ErrInvalidSignature    = errors.New("the signature is invalid or not match the address")

	ErrGasNotEnough   = errors.New("Gas not enough")
	ErrInvalidAccount = errors.New("Invalid account")
	ErrVersionInvalid = errors.New("Invalid tx version")
	ErrInvalidTxExt   = errors.New("Invalid tx ext")
	ErrTxTooLarge     = errors.New("Tx size is too large")

	ErrGetReservedContracts = errors.New("Get reserved contracts error")
	ErrParseContractUtxos   = errors.New("Parse contract utxos error")
	ErrContractTxAmout      = errors.New("Contract transfer amount error")
)

// package constants
const (
	UTXOLockExpiredSecond = 60
	LatestBlockKey        = "pointer"
	UTXOCacheSize         = 1000
	OfflineTxChanBuffer   = 100000

	// TxVersion 为所有交易使用的版本
	TxVersion = 1
	// BetaTxVersion 为当前代码支持的最高交易版本
	BetaTxVersion = 3

	RootTxVersion             = 0
	FeePlaceholder            = "$"
	UTXOTotalKey              = "xtotal"
	UTXOContractExecutionTime = 500
	DefaultMaxConfirmedDelay  = 300
)

type TxLists []*pb.Transaction

type UtxoVM struct {
	Meta                 *pb.UtxoMeta // utxo meta
	MetaTmp              *pb.UtxoMeta // tmp utxo meta
	MutexMeta            *sync.Mutex  // access control for meta
	ldb                  kvdb.Database
	Mutex                *sync.RWMutex // utxo leveldb表读写锁
	MutexMem             *sync.Mutex   // 内存锁定状态互斥锁
	SpLock               *SpinLock     // 自旋锁,根据交易涉及的utxo和改写的变量
	mutexBalance         *sync.Mutex   // 余额Cache锁
	lockKeys             map[string]*UtxoLockItem
	lockKeyList          *list.List // 按锁定的先后顺序，方便过期清理
	lockExpireTime       int        // 临时锁定的最长时间
	UtxoCache            *UtxoCache
	log                  logs.Logger
	ledger               *ledger.Ledger           // 引用的账本对象
	latestBlockid        []byte                   // 当前vm最后一次执行到的blockid
	utxoTable            kvdb.Database            // utxo表
	OfflineTxChan        chan []*pb.Transaction   // 未确认tx的通知chan
	PrevFoundKeyCache    *cache.LRUCache          // 上一次找到的可用utxo key，用于加速GenerateTx
	utxoTotal            *big.Int                 // 总资产
	cryptoClient         crypto_base.CryptoClient // 加密实例
	ModifyBlockAddr      string                   // 可修改区块链的监管地址
	BalanceCache         *cache.LRUCache          //余额cache,加速GetBalance查询
	CacheSize            int                      //记录构造utxo时传入的cachesize
	BalanceViewDirty     map[string]int           //balanceCache 标记dirty: addr -> sequence of view
	contractExectionTime int
	unconfirmTxInMem     *sync.Map //未确认Tx表的内存镜像
	maxConfirmedDelay    uint32    // 交易处于unconfirm状态的最长时间，超过后会被回滚
	unconfirmTxAmount    int64     // 未确认的Tx数目，用于监控
	avgDelay             int64     // 平均上链延时
	bcname               string

	// 最新区块高度通知装置
	heightNotifier *BlockHeightNotifier
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
	baseUtxoKey := genUtxoKey(addr, txid, offset)
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
			uv.log.Warn("found duplicated utxo in same tx", "utxoKey", utxoKey, "txid", global.F(tx.Txid))
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
				if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
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
				"in_utxo", amount, "txInputAmount", txInputAmount, "txid", fmt.Sprintf("%x", tx.Txid), "reftxid", fmt.Sprintf("%x", txid))
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
	uv.mutexMem.Lock()
	defer uv.mutexMem.Unlock()
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
	uv.mutexMem.Lock()
	defer uv.mutexMem.Unlock()
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
func NewUtxo(sctx *def.StateCtx) (*UtxoVM, error) {
	return MakeUtxo(sctx, UTXOCacheSize, UTXOLockExpiredSecond, UTXOContractExecutionTime)
}

// MakeUtxoVM 这个函数比NewUtxoVM更加可订制化
func MakeUtxo(sctx *def.StateCtx, cachesize int, tmplockSeconds, contractExectionTime int) (*UtxoVM, error) {
	// new kvdb instance
	kvParam := &kvdb.KVParameter{
		DBPath:                filepath.Join(sctx.LedgerCfg.StorePath, "utxoVM"),
		KVEngineType:          sctx.LedgerCfg.KVEngineType,
		MemCacheSize:          ledger.MemCacheSize,
		FileHandlersCacheSize: ledger.FileHandlersCacheSize,
		OtherPaths:            sctx.LedgerCfg.OtherPaths,
		StorageType:           sctx.LedgerCfg.StorageType,
	}
	baseDB, err := kvdb.CreateKVInstance(kvParam)
	if err != nil {
		xlog.Warn("fail to open leveldb", "dbPath", sctx.LedgerCfg.StorePath+"/utxoVM", "err", err)
		return nil, err
	}

	utxoMutex := &sync.RWMutex{}
	utxoVM := &UtxoVM{
		Meta:                 &pb.UtxoMeta{},
		MetaTmp:              &pb.UtxoMeta{},
		MutexMeta:            &sync.Mutex{},
		ldb:                  baseDB,
		Mutex:                utxoMutex,
		MutexMem:             &sync.Mutex{},
		SpLock:               NewSpinLock(),
		mutexBalance:         &sync.Mutex{},
		lockKeys:             map[string]*UtxoLockItem{},
		lockKeyList:          list.New(),
		lockExpireTime:       tmplockSeconds,
		log:                  sctx.XLog,
		ledger:               sctx.Ledger,
		utxoTable:            kvdb.NewTable(baseDB, pb.UTXOTablePrefix),
		UtxoCache:            NewUtxoCache(cachesize),
		OfflineTxChan:        make(chan []*pb.Transaction, OfflineTxChanBuffer),
		PrevFoundKeyCache:    cache.NewLRUCache(cachesize),
		utxoTotal:            big.NewInt(0),
		BalanceCache:         cache.NewLRUCache(cachesize),
		CacheSize:            cachesize,
		BalanceViewDirty:     map[string]int{},
		contractExectionTime: contractExectionTime,
		cryptoClient:         sctx.Crypt,
		maxConfirmedDelay:    DefaultMaxConfirmedDelay,
		bcname:               sctx.BCName,
		heightNotifier:       NewBlockHeightNotifier(),
	}

	latestBlockid, findErr := utxoVM.metaTable.Get([]byte(LatestBlockKey))
	if findErr == nil {
		utxoVM.latestBlockid = latestBlockid
	} else {
		if def.NormalizedKVError(findErr) != def.ErrKVNotFound {
			return nil, findErr
		}
	}
	utxoTotalBytes, findTotalErr := utxoVM.metaTable.Get([]byte(UTXOTotalKey))
	if findTotalErr == nil {
		total := big.NewInt(0)
		total.SetBytes(utxoTotalBytes)
		utxoVM.utxoTotal = total
	} else {
		if def.NormalizedKVError(findTotalErr) != def.ErrKVNotFound {
			return nil, findTotalErr
		}
		//说明是1.1.1版本，没有utxo total字段, 估算一个
		//utxoVM.utxoTotal = ledger.GetEstimatedTotal()
		utxoVM.utxoTotal = big.NewInt(0)
		xlog.Info("utxo total is estimated", "total", utxoVM.utxoTotal)
	}
	loadErr := tx.LoadUnconfirmedTxFromDisk()
	if loadErr != nil {
		xlog.Warn("faile to load unconfirmed tx from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// cp not reference
	newMeta := proto.Clone(utxoVM.meta).(*pb.UtxoMeta)
	utxoVM.metaTmp = newMeta
	return utxoVM, nil
}

// ClearCache 清空cache, 写盘失败的时候
func (uv *UtxoVM) ClearCache() {
	uv.UtxoCache = NewUtxoCache(uv.CacheSize)
	uv.PrevFoundKeyCache = cache.NewLRUCache(uv.CacheSize)
	uv.clearBalanceCache()
	uv.log.Warn("clear utxo cache")
}

func (uv *UtxoVM) clearBalanceCache() {
	uv.log.Warn("clear balance cache")
	uv.balanceCache = cache.NewLRUCache(uv.cacheSize) //清空balanceCache
	uv.balanceViewDirty = map[string]int{}            //清空cache dirty flag表
	uv.model3.CleanCache()
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
func (uv *UtxoVM) SelectUtxos(fromAddr string, totalNeed *big.Int, needLock, excludeUnconfirmed bool) ([]*pb.TxInput, [][]byte, *big.Int, error) {
	if totalNeed.Cmp(big.NewInt(0)) == 0 {
		return nil, nil, big.NewInt(0), nil
	}
	curLedgerHeight := uv.ledger.GetMeta().TrunkHeight
	willLockKeys := make([][]byte, 0)
	foundEnough := false
	utxoTotal := big.NewInt(0)
	cacheKeys := map[string]bool{} // 先从cache里找找，不够再从leveldb找,因为leveldb prefix scan比较慢
	txInputs := []*pb.TxInput{}
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
			txInput := &pb.TxInput{
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
		preFoundUtxoKey, mdOK := uv.prevFoundKeyCache.Get(fromAddr)
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
			txInput := &pb.TxInput{
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
				uv.prevFoundKeyCache.Add(fromAddr, key)
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
	balance, hitCache := uv.balanceCache.Get(string(addr))
	if hitCache {
		iBalance := balance.(*big.Int)
		iBalance.Add(iBalance, delta)
	} else {
		uv.balanceViewDirty[string(addr)] = uv.balanceViewDirty[string(addr)] + 1
	}
}

// subBalance 减少cache中的Balance
func (uv *UtxoVM) SubBalance(addr []byte, delta *big.Int) {
	uv.mutexBalance.Lock()
	defer uv.mutexBalance.Unlock()
	balance, hitCache := uv.balanceCache.Get(string(addr))
	if hitCache {
		iBalance := balance.(*big.Int)
		iBalance.Sub(iBalance, delta)
	} else {
		uv.balanceViewDirty[string(addr)] = uv.balanceViewDirty[string(addr)] + 1
	}
}

//获得一个账号的余额，inLock表示在调用此函数时已经对uv.mutex加过锁了
func (uv *UtxoVM) GetBalance(addr string) (*big.Int, error) {
	cachedBalance, ok := uv.balanceCache.Get(addr)
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
	myBalanceView := uv.balanceViewDirty[addr]
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
	if myBalanceView != uv.balanceViewDirty[addr] {
		return utxoTotal, nil
	}
	_, exist := uv.balanceCache.Get(addr)
	if !exist {
		//首次填充cache
		uv.balanceCache.Add(addr, utxoTotal)
		delete(uv.balanceViewDirty, addr)
	}
	balanceCopy := big.NewInt(0).Set(utxoTotal)
	return balanceCopy, nil
}

// QueryAccountACLWithConfirmed query account's ACL with confirm status
func (uv *UtxoVM) QueryAccountACLWithConfirmed(accountName string) (*pb.Acl, bool, error) {
	return uv.queryAccountACLWithConfirmed(accountName)
}

// QueryContractMethodACLWithConfirmed query contract method's ACL with confirm status
func (uv *UtxoVM) QueryContractMethodACLWithConfirmed(contractName string, methodName string) (*pb.Acl, bool, error) {
	return uv.queryContractMethodACLWithConfirmed(contractName, methodName)
}

// GetBalance 查询Address的可用余额
func (uv *UtxoVM) GetBalance(addr string) (*big.Int, error) {
	return uv.getBalance(addr)
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

// GetFromTable 从一个表读取一个key的value
func (uv *UtxoVM) GetFromTable(tablePrefix []byte, key []byte) ([]byte, error) {
	realKey := append([]byte(tablePrefix), key...)
	return uv.ldb.Get(realKey)
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

// SetMaxConfirmedDelay set the max value of tx confirm delay. If beyond, tx will be rollbacked
func (uv *UtxoVM) SetMaxConfirmedDelay(seconds uint32) {
	uv.maxConfirmedDelay = seconds
	uv.log.Info("set max confirmed delay of tx", "seconds", seconds)
}

// SetModifyBlockAddr set modified block addr
func (uv *UtxoVM) SetModifyBlockAddr(addr string) {
	uv.ModifyBlockAddr = addr
	uv.log.Info("set modified block addr", "addr", addr)
}

// GetAccountContracts get account contracts, return a slice of contract names
func (uv *UtxoVM) GetAccountContracts(account string) ([]string, error) {
	contracts := []string{}
	if acl.IsAccount(account) != 1 {
		uv.log.Warn("GetAccountContracts valid account name error", "error", "account name is not valid")
		return nil, errors.New("account name is not valid")
	}
	prefKey := pb.ExtUtxoTablePrefix + string(xmodel.MakeRawKey(utils.GetAccount2ContractBucket(), []byte(account+utils.GetACLSeparator())))
	it := uv.ldb.NewIteratorWithPrefix([]byte(prefKey))
	defer it.Release()
	for it.Next() {
		key := string(it.Key())
		ret := strings.Split(key, utils.GetACLSeparator())
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
	utxoRecordDetail := &pb.UtxoRecordDetail{
		Header: &pb.Header{},
	}

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

	utxoRecordDetail.OpenUtxoRecord = &pb.UtxoRecord{
		UtxoCount:  openUtxoCount.String(),
		UtxoAmount: openUtxoAmount.String(),
		Item:       openItem,
	}
	utxoRecordDetail.LockedUtxoRecord = &pb.UtxoRecord{
		UtxoCount:  lockedUtxoCount.String(),
		UtxoAmount: lockedUtxoAmount.String(),
		Item:       lockedItem,
	}
	utxoRecordDetail.FrozenUtxoRecord = &pb.UtxoRecord{
		UtxoCount:  frozenUtxoCount.String(),
		UtxoAmount: frozenUtxoAmount.String(),
		Item:       frozenItem,
	}

	return utxoRecordDetail, nil
}

// WaitBlockHeight wait util the height of current block >= target
func (uv *UtxoVM) WaitBlockHeight(target int64) int64 {
	return uv.heightNotifier.WaitHeight(target)
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
