// Package utxo is the key part of XuperChain, this module keeps all Unspent Transaction Outputs.
//
// For a transaction, the UTXO checks the tokens used in reference transactions are unspent, and
// reject the transaction if the initiator doesn't have enough tokens.
// UTXO also checks the signature and permission of transaction members.
package utxo

import (
	"bytes"
	"container/list"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	xmodel_pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/state/xmodel/pb"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/permission/acl"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"github.com/xuperchain/xupercore/lib/cache"
	crypto_client "github.com/xuperchain/xupercore/lib/crypto/client"
	crypto_base "github.com/xuperchain/xupercore/lib/crypto/client/base"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/pb"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

// 常用VM执行错误码
var (
	ErrDoubleSpent             = errors.New("utxo can not be spent more than once")
	ErrAlreadyInUnconfirmed    = errors.New("this transaction is in unconfirmed state")
	ErrAlreadyConfirmed        = errors.New("this transaction is already confirmed")
	ErrNoEnoughUTXO            = errors.New("no enough money(UTXO) to start this transaction")
	ErrUTXONotFound            = errors.New("this utxo can not be found")
	ErrInputOutputNotEqual     = errors.New("input's amount is not equal to output's")
	ErrTxNotFound              = errors.New("this tx can not be found in unconfirmed-table")
	ErrPreBlockMissMatch       = errors.New("play block failed because pre-hash != latest_block")
	ErrUnexpected              = errors.New("this is a unexpected error")
	ErrNegativeAmount          = errors.New("amount in transaction can not be negative number")
	ErrUTXOFrozen              = errors.New("utxo is still frozen")
	ErrTxSizeLimitExceeded     = errors.New("tx size limit exceeded")
	ErrOverloaded              = errors.New("this node is busy, try again later")
	ErrInvalidAutogenTx        = errors.New("found invalid autogen-tx")
	ErrUnmatchedExtension      = errors.New("found unmatched extension")
	ErrUnsupportedContract     = errors.New("found unspported contract module")
	ErrUTXODuplicated          = errors.New("found duplicated utxo in same tx")
	ErrDestroyProofAlreadyUsed = errors.New("the destroy proof has been used before")
	ErrInvalidWithdrawAmount   = errors.New("withdraw amount is invalid")
	ErrServiceRefused          = errors.New("Service refused")
	ErrRWSetInvalid            = errors.New("RWSet of transaction invalid")
	ErrACLNotEnough            = errors.New("ACL not enough")
	ErrInvalidSignature        = errors.New("the signature is invalid or not match the address")

	ErrGasNotEnough   = errors.New("Gas not enough")
	ErrInvalidAccount = errors.New("Invalid account")
	ErrVersionInvalid = errors.New("Invalid tx version")
	ErrInvalidTxExt   = errors.New("Invalid tx ext")
	ErrTxTooLarge     = errors.New("Tx size is too large")

	ErrGetReservedContracts = errors.New("Get reserved contracts error")
	ErrInvokeReqParams      = errors.New("Invalid invoke request params")
	ErrParseContractUtxos   = errors.New("Parse contract utxos error")
	ErrContractTxAmout      = errors.New("Contract transfer amount error")
	ErrDuplicatedTx         = errors.New("Receive a duplicated tx")
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
	TxWaitTimeout             = 5
	DefaultMaxConfirmedDelay  = 300
)

type TxLists []*pb.Transaction

// UtxoVM UTXO VM
type UtxoVM struct {
	meta              *pb.UtxoMeta // utxo meta
	metaTmp           *pb.UtxoMeta // tmp utxo meta
	mutexMeta         *sync.Mutex  // access control for meta
	ldb               kvdb.Database
	mutex             *sync.RWMutex // utxo leveldb表读写锁
	mutexMem          *sync.Mutex   // 内存锁定状态互斥锁
	spLock            *SpinLock     // 自旋锁,根据交易涉及的utxo和改写的变量
	mutexBalance      *sync.Mutex   // 余额Cache锁
	lockKeys          map[string]*UtxoLockItem
	lockKeyList       *list.List // 按锁定的先后顺序，方便过期清理
	lockExpireTime    int        // 临时锁定的最长时间
	utxoCache         *UtxoCache
	log               logs.Logger
	ledger            *ledger.Ledger       // 引用的账本对象
	latestBlockid     []byte                   // 当前vm最后一次执行到的blockid
	utxoTable         kvdb.Database            // utxo表
	smartContract     *contract.SmartContract  // 智能合约执行机
	OfflineTxChan     chan []*pb.Transaction   // 未确认tx的通知chan
	prevFoundKeyCache *cache.LRUCache         // 上一次找到的可用utxo key，用于加速GenerateTx
	utxoTotal         *big.Int                 // 总资产
	cryptoClient      crypto_base.CryptoClient // 加密实例
	modifyBlockAddr   string                   // 可修改区块链的监管地址
	vmMgr3            *contract.VMManager
	vatHandler           *tx.VATHandler  // Verifiable Autogen Tx 生成器
	balanceCache         *cache.LRUCache //余额cache,加速GetBalance查询
	cacheSize            int              //记录构造utxo时传入的cachesize
	balanceViewDirty     map[string]int   //balanceCache 标记dirty: addr -> sequence of view
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

type UtxoLockItem struct {
	timestamp int64
	holder    *list.Element
}

type contractChainCore struct {
	*acli.Manager // ACL manager for read/write acl table
	*UtxoVM
	*ledger.Ledger
}

func genUtxoKey(addr []byte, txid []byte, offset int32) string {
	return fmt.Sprintf("%s_%x_%d", addr, txid, offset)
}

// GenUtxoKeyWithPrefix generate UTXO key with given prefix
func GenUtxoKeyWithPrefix(addr []byte, txid []byte, offset int32) string {
	baseUtxoKey := genUtxoKey(addr, txid, offset)
	return pb.UTXOTablePrefix + baseUtxoKey
}

// checkInputEqualOutput 校验交易的输入输出是否相等
func (uv *UtxoVM) checkInputEqualOutput(tx *pb.Transaction) error {
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
		utxoKey := genUtxoKey(addr, txid, offset)
		if utxoDedup[utxoKey] {
			uv.log.Warn("found duplicated utxo in same tx", "utxoKey", utxoKey, "txid", global.F(tx.Txid))
			return ErrUTXODuplicated
		}
		utxoDedup[utxoKey] = true
		var amountBytes []byte
		var frozenHeight int64
		uv.utxoCache.Lock()
		if l2Cache, exist := uv.utxoCache.All[string(addr)]; exist {
			uItem := l2Cache[pb.UTXOTablePrefix+utxoKey]
			if uItem != nil {
				amountBytes = uItem.Amount.Bytes()
				frozenHeight = uItem.FrozenHeight
			}
		}
		uv.utxoCache.Unlock()
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
	uv.mutexMem.Lock()
	defer uv.mutexMem.Unlock()
	_, exist := uv.lockKeys[string(utxoKey)]
	return exist
}

// 解锁utxo key
func (uv *UtxoVM) unlockKey(utxoKey []byte) {
	uv.mutexMem.Lock()
	defer uv.mutexMem.Unlock()
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
// TODO: remove isBeta parameter
func NewUtxo(lctx *def.LedgerCtx, ledger *ledger.Ledger) (*UtxoVM, error) {
	return MakeUtxo(lctx, ledger, lctx.XLog, UTXOCacheSize, UTXOLockExpiredSecond, UTXOContractExecutionTime)
}

// MakeUtxoVM 这个函数比NewUtxoVM更加可订制化
func MakeUtxo(lctx *def.LedgerCtx, ledger *ledger.Ledger, xlog logs.Logger,
	cachesize int, tmplockSeconds, contractExectionTime int) (*UtxoVM, error) {
	// new kvdb instance
	kvParam := &kvdb.KVParameter{
		DBPath:                filepath.Join(lctx.LedgerCfg.StorePath, "utxoVM"),
		KVEngineType:          lctx.LedgerCfg.KVEngineType,
		MemCacheSize:          ledger.MemCacheSize,
		FileHandlersCacheSize: ledger.FileHandlersCacheSize,
		OtherPaths:            lctx.LedgerCfg.OtherPaths,
		StorageType:           lctx.LedgerCfg.StorageType,
	}
	baseDB, err := kvdb.CreateKVInstance(kvParam)
	if err != nil {
		xlog.Warn("fail to open leveldb", "dbPath", lctx.LedgerCfg.StorePath+"/utxoVM", "err", err)
		return nil, err
	}

	// create crypto client
	cryptoClient, cryptoErr := crypto_client.CreateCryptoClient(lctx.CryptoType)
	if cryptoErr != nil {
		xlog.Warn("fail to create crypto client", "err", cryptoErr)
		return nil, cryptoErr
	}

	//todo create vmMgr
	vmManager, verr := contract.NewVMManager(xlog)
	if verr != nil {
		return nil, verr
	}
	utxoMutex := &sync.RWMutex{}
	utxoVM := &UtxoVM{
		meta:              &pb.UtxoMeta{},
		metaTmp:           &pb.UtxoMeta{},
		mutexMeta:         &sync.Mutex{},
		ldb:               baseDB,
		mutex:             utxoMutex,
		mutexMem:          &sync.Mutex{},
		spLock:            NewSpinLock(),
		mutexBalance:      &sync.Mutex{},
		lockKeys:          map[string]*UtxoLockItem{},
		lockKeyList:       list.New(),
		lockExpireTime:    tmplockSeconds,
		log:               xlog,
		ledger:            ledger,
		utxoTable:         kvdb.NewTable(baseDB, pb.UTXOTablePrefix),
		utxoCache:         NewUtxoCache(cachesize),
		smartContract:     contract.NewSmartContract(),
		vatHandler:        tx.NewVATHandler(),
		OfflineTxChan:     make(chan []*pb.Transaction, OfflineTxChanBuffer),
		prevFoundKeyCache: cache.NewLRUCache(cachesize),
		utxoTotal:         big.NewInt(0),
		balanceCache:         cache.NewLRUCache(cachesize),
		cacheSize:            cachesize,
		balanceViewDirty:     map[string]int{},
		contractExectionTime: contractExectionTime,
		cryptoClient:         cryptoClient,
		vmMgr3:               vmManager,
		maxConfirmedDelay:    DefaultMaxConfirmedDelay,
		bcname:               lctx.BCName,
		heightNotifier:       NewBlockHeightNotifier(),
	}

	latestBlockid, findErr := utxoVM.metaTable.Get([]byte(LatestBlockKey))
	if findErr == nil {
		utxoVM.latestBlockid = latestBlockid
	} else {
		if common.NormalizedKVError(findErr) != common.ErrKVNotFound {
			return nil, findErr
		}
	}
	utxoTotalBytes, findTotalErr := utxoVM.metaTable.Get([]byte(UTXOTotalKey))
	if findTotalErr == nil {
		total := big.NewInt(0)
		total.SetBytes(utxoTotalBytes)
		utxoVM.utxoTotal = total
	} else {
		if common.NormalizedKVError(findTotalErr) != common.ErrKVNotFound {
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
	// load consensus parameters
	utxoVM.meta.MaxBlockSize, loadErr = utxoVM.LoadMaxBlockSize()
	if loadErr != nil {
		xlog.Warn("failed to load maxBlockSize from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	utxoVM.meta.ForbiddenContract, loadErr = utxoVM.LoadForbiddenContract()
	if loadErr != nil {
		xlog.Warn("failed to load forbiddenContract from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	utxoVM.meta.ReservedContracts, loadErr = utxoVM.LoadReservedContracts()
	if loadErr != nil {
		xlog.Warn("failed to load reservedContracts from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	utxoVM.meta.NewAccountResourceAmount, loadErr = utxoVM.LoadNewAccountResourceAmount()
	if loadErr != nil {
		xlog.Warn("failed to load newAccountResourceAmount from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// load irreversible block height & slide window parameters
	utxoVM.meta.IrreversibleBlockHeight, loadErr = utxoVM.LoadIrreversibleBlockHeight()
	if loadErr != nil {
		xlog.Warn("failed to load irreversible block height from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	utxoVM.meta.IrreversibleSlideWindow, loadErr = utxoVM.LoadIrreversibleSlideWindow()
	if loadErr != nil {
		xlog.Warn("failed to load irreversibleSlide window from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// load gas price
	utxoVM.meta.GasPrice, loadErr = utxoVM.LoadGasPrice()
	if loadErr != nil {
		xlog.Warn("failed to load gas price from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// load group chain
	utxoVM.meta.GroupChainContract, loadErr = utxoVM.LoadGroupChainContract()
	if loadErr != nil {
		xlog.Warn("failed to load groupchain from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// cp not reference
	newMeta := proto.Clone(utxoVM.meta).(*pb.UtxoMeta)
	utxoVM.metaTmp = newMeta
	return utxoVM, nil
}

// ClearCache 清空cache, 写盘失败的时候
func (uv *UtxoVM) ClearCache() {
	uv.utxoCache = NewUtxoCache(uv.cacheSize)
	uv.prevFoundKeyCache = common.NewLRUCache(uv.cacheSize)
	uv.clearBalanceCache()
	uv.model3.CleanCache()
	uv.log.Warn("clear utxo cache")
}

func (uv *UtxoVM) clearBalanceCache() {
	uv.log.Warn("clear balance cache")
	uv.balanceCache = common.NewLRUCache(uv.cacheSize) //清空balanceCache
	uv.balanceViewDirty = map[string]int{}             //清空cache dirty flag表
	uv.model3.CleanCache()
}

// RegisterVM add new contract VM
func (uv *UtxoVM) RegisterVM(name string, vm contract.ContractInterface, priv int) bool {
	return uv.smartContract.RegisterHandler(name, vm, priv)
}

// UnRegisterVM remove contract VM
func (uv *UtxoVM) UnRegisterVM(name string, priv int) {
	uv.smartContract.Remove(name, priv)
}

// RegisterVM3 add Xuper3 contract VM
func (uv *UtxoVM) RegisterVM3(module string, vm contract.VirtualMachine) error {
	return uv.vmMgr3.RegisterVM(module, vm)
}

// RegisterVAT add VAT
func (uv *UtxoVM) RegisterVAT(name string, vat vat.VATInterface, whiteList map[string]bool) {
	uv.vatHandler.RegisterHandler(name, vat, whiteList)
	uv.log.Trace("RegisterVAT", "vathandler", uv.vatHandler)
}

// UnRegisterVAT remove VAT
func (uv *UtxoVM) UnRegisterVAT(name string) {
	uv.vatHandler.Remove(name)
}

func (uv *UtxoVM) updateLatestBlockid(newBlockid []byte, batch kvdb.Batch, reason string) error {
	// FIXME: 如果在高频的更新场景中可能有性能问题，需要账本加上cache
	blk, err := uv.ledger.QueryBlockHeader(newBlockid)
	if err != nil {
		return err
	}
	batch.Put(append([]byte(pb.MetaTablePrefix), []byte(LatestBlockKey)...), newBlockid)
	writeErr := batch.Write()
	if writeErr != nil {
		uv.ClearCache()
		uv.log.Warn(reason, "writeErr", writeErr)
		return writeErr
	}
	uv.latestBlockid = newBlockid
	uv.heightNotifier.UpdateHeight(blk.GetHeight())
	return nil
}

func (uv *UtxoVM) updateUtxoTotal(delta *big.Int, batch kvdb.Batch, inc bool) {
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
	uv.utxoCache.Lock()
	if l2Cache, exist := uv.utxoCache.Available[fromAddr]; exist {
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
			uv.utxoCache.Use(fromAddr, uKey)
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
	uv.utxoCache.Unlock()
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
			if utxoTotal.Cmp(totalNeed) >= 0 { // 找到了足够的utxo用于支付
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
			uv.unlockKey(lk)
		}
		return nil, nil, nil, ErrNoEnoughUTXO // 余额不足啦
	}
	return txInputs, willLockKeys, utxoTotal, nil
}

// DebugTx print transaction info in log for debug
func (uv *UtxoVM) DebugTx(tx *pb.Transaction) error {
	uv.log.Debug("debug tx", "txid", fmt.Sprintf("%x", tx.Txid))
	for offset, txInput := range tx.TxInputs {
		addr := txInput.FromAddr
		txid := txInput.RefTxid
		refOffset := txInput.RefOffset
		amountBytes := txInput.Amount
		amount := big.NewInt(0)
		amount.SetBytes(amountBytes)
		uv.log.Debug("txinput", "offset", offset, "addr", string(addr),
			"reftxid", fmt.Sprintf("%x", txid), "refoffset", refOffset, "amount", amount)
	}
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		amount := big.NewInt(0)
		amount.SetBytes(txOutput.Amount)
		uv.log.Debug("txoutput", "offset", offset, "addr", string(addr), "amount", amount)
	}
	return nil
}

// addBalance 增加cache中的Balance
func (uv *UtxoVM) addBalance(addr []byte, delta *big.Int) {
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
func (uv *UtxoVM) subBalance(addr []byte, delta *big.Int) {
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

// payFee 扣除小费给矿工
func (uv *UtxoVM) payFee(tx *pb.Transaction, batch kvdb.Batch, block *pb.InternalBlock) error {
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if !bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		addr = block.Proposer // 占位符替换为矿工
		utxoKey := GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		uItem := &UtxoItem{}
		uItem.Amount = big.NewInt(0)
		uItem.Amount.SetBytes(txOutput.Amount)
		uItemBinary, uErr := uItem.Dumps()
		if uErr != nil {
			return uErr
		}
		batch.Put([]byte(utxoKey), uItemBinary) // 插入本交易产生的utxo
		uv.addBalance(addr, uItem.Amount)
		uv.utxoCache.Insert(string(addr), utxoKey, uItem)
		uv.log.Trace("    insert fee utxo key", "utxoKey", utxoKey, "amount", uItem.Amount.String())
	}
	return nil
}

// undoPayFee 回滚小费
func (uv *UtxoVM) undoPayFee(tx *pb.Transaction, batch kvdb.Batch, block *pb.InternalBlock) error {
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if !bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		addr = block.Proposer
		utxoKey := GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		// 删除产生的UTXO
		batch.Delete([]byte(utxoKey))
		uv.utxoCache.Remove(string(addr), utxoKey)
		uv.subBalance(addr, big.NewInt(0).SetBytes(txOutput.Amount))
		uv.log.Info("undo delete fee utxo key", "utxoKey", utxoKey)
	}
	return nil
}

func (uv *UtxoVM) runContract(blockid []byte, tx *pb.Transaction, autogenTxList *[]*pb.Transaction, deadline int64) error {
	// 去掉高度判断，因为在CreateBlockChain的时候，没有传递矿工的地址和keys。
	if tx.Autogen && autogenTxList != nil { // 自动生成的tx, 需要校验下
		if len(*autogenTxList) == 0 {
			uv.log.Warn("autogenTxList has been drained")
			return ErrInvalidAutogenTx
		}
		if !bytes.Equal(tx.Desc, (*autogenTxList)[0].Desc) {
			uv.log.Warn("mismatch contract desc", "expected", string((*autogenTxList)[0].Desc), "got", string(tx.Desc))
			return ErrInvalidAutogenTx
		}
		uv.log.Debug("autogen tx contract checked ok", "desc", string(tx.Desc))
		*autogenTxList = (*autogenTxList)[1:] //pop front
	}
	if txDesc, ok := uv.isSmartContract(tx.Desc); ok { // 交易需要执行智能合约
		txDesc.Tx = tx
		txDesc.Deadline = deadline

		if uv.MustVAT(txDesc) && !txDesc.Tx.Autogen {
			return fmt.Errorf("this contract %s.%s can only be auto generated by proposal", txDesc.Module, txDesc.Method)
		}

		if scErr := uv.smartContract.Run(txDesc); scErr != nil {
			uv.log.Warn("failed to Run contract", "scErr", scErr, "txid", fmt.Sprintf("%x", txDesc.Tx.Txid))
			return scErr
		}
	}
	return nil
}

// VerifyTx check the tx signature and permission
func (uv *UtxoVM) VerifyTx(tx *pb.Transaction) (bool, error) {
	isValid, err := uv.ImmediateVerifyTx(tx, false)
	if err != nil || !isValid {
		uv.log.Warn("ImmediateVerifyTx failed", "error", err,
			"AuthRequire ", tx.AuthRequire, "AuthRequireSigns ", tx.AuthRequireSigns,
			"Initiator", tx.Initiator, "InitiatorSigns", tx.InitiatorSigns, "XuperSign", tx.XuperSign)
		ok, isRelyOnMarkedTx, err := uv.verifyMarked(tx)
		if isRelyOnMarkedTx {
			if !ok || err != nil {
				uv.log.Warn("tx verification failed because it is blocked tx", "err", err)
			} else {
				uv.log.Trace("blocked tx verification succeed")
			}
			return ok, err
		}
	}
	return isValid, err
}

//执行一个block的时候, 处理本地未确认交易
//返回：被确认的txid集合、err
func (uv *UtxoVM) processUnconfirmTxs(block *pb.InternalBlock, batch kvdb.Batch, needRepost bool) (map[string]bool, map[string]bool, error) {
	if !bytes.Equal(block.PreHash, uv.latestBlockid) {
		uv.log.Warn("play failed", "block.PreHash", fmt.Sprintf("%x", block.PreHash),
			"latestBlockid", fmt.Sprintf("%x", uv.latestBlockid))
		return nil, nil, ErrPreBlockMissMatch
	}
	txidsInBlock := map[string]bool{}    // block里面所有的txid
	UTXOKeysInBlock := map[string]bool{} // block里面所有的交易需要用掉的utxo
	keysVersionInBlock := map[string]string{}
	uv.mutex.Unlock()
	for _, tx := range block.Transactions {
		txidsInBlock[string(tx.Txid)] = true
		for _, txInput := range tx.TxInputs {
			utxoKey := genUtxoKey(txInput.FromAddr, txInput.RefTxid, txInput.RefOffset)
			if UTXOKeysInBlock[utxoKey] { //检查块内的utxo双花情况
				uv.log.Warn("found duplicated utxo in same block", "utxoKey", utxoKey, "txid", global.F(tx.Txid))
				uv.mutex.Lock()
				return nil, nil, ErrUTXODuplicated
			}
			UTXOKeysInBlock[utxoKey] = true
		}
		for txOutOffset, txOut := range tx.TxOutputsExt {
			valueVersion := xmodel.MakeVersion(tx.Txid, int32(txOutOffset))
			bucketAndKey := xmodel.MakeRawKey(txOut.Bucket, txOut.Key)
			keysVersionInBlock[string(bucketAndKey)] = valueVersion
		}
	}
	uv.mutex.Lock()
	// 下面开始处理unconfirmed的交易
	unconfirmTxMap, unconfirmTxGraph, delayedTxMap, loadErr := uv.sortUnconfirmedTx()
	if loadErr != nil {
		return nil, nil, loadErr
	}
	uv.log.Info("unconfirm table size", "unconfirmTxMap", uv.unconfirmTxAmount)
	undoDone := map[string]bool{}
	unconfirmToConfirm := map[string]bool{}
	for txid, unconfirmTx := range unconfirmTxMap {
		if _, exist := txidsInBlock[string(txid)]; exist {
			// 说明这个交易已经被确认
			batch.Delete(append([]byte(pb.UnconfirmedTablePrefix), []byte(txid)...))
			uv.log.Trace("  delete from unconfirmed", "txid", fmt.Sprintf("%x", txid))
			// 直接从unconfirm表删除, 大部分情况是这样的
			unconfirmToConfirm[txid] = true
			continue
		}
		hasConflict := false
		for _, unconfirmTxInput := range unconfirmTx.TxInputs {
			addr := unconfirmTxInput.FromAddr
			txid := unconfirmTxInput.RefTxid
			offset := unconfirmTxInput.RefOffset
			utxoKey := genUtxoKey(addr, txid, offset)
			if _, exist := UTXOKeysInBlock[utxoKey]; exist {
				// 说明此交易和block里面的交易存在双花冲突，需要回滚, 少数情况
				uv.log.Warn("conflict, refuse double spent", "key", utxoKey, "txid", global.F(unconfirmTx.Txid))
				hasConflict = true
				break
			}
		}
		for _, txInputExt := range unconfirmTx.TxInputsExt {
			bucketAndKey := xmodel.MakeRawKey(txInputExt.Bucket, txInputExt.Key)
			localVersion := xmodel.MakeVersion(txInputExt.RefTxid, txInputExt.RefOffset)
			remoteVersion := keysVersionInBlock[string(bucketAndKey)]
			if localVersion != remoteVersion && remoteVersion != "" {
				txidInVer := xmodel.GetTxidFromVersion(remoteVersion)
				if _, known := unconfirmTxMap[string(txidInVer)]; known {
					continue
				}
				uv.log.Warn("inputs version conflict", "key", bucketAndKey, "localVersion", localVersion, "remoteVersion", remoteVersion)
				hasConflict = true
				break
			}
		}
		for txOutOffset, txOut := range unconfirmTx.TxOutputsExt {
			bucketAndKey := xmodel.MakeRawKey(txOut.Bucket, txOut.Key)
			localVersion := xmodel.MakeVersion(unconfirmTx.Txid, int32(txOutOffset))
			remoteVersion := keysVersionInBlock[string(bucketAndKey)]
			if localVersion != remoteVersion && remoteVersion != "" {
				txidInVer := xmodel.GetTxidFromVersion(remoteVersion)
				if _, known := unconfirmTxMap[string(txidInVer)]; known {
					continue
				}
				uv.log.Warn("outputs version conflict", "key", bucketAndKey, "localVersion", localVersion, "remoteVersion", remoteVersion)
				hasConflict = true
				break
			}
		}
		tooDelayed := delayedTxMap[string(unconfirmTx.Txid)]
		if tooDelayed {
			uv.log.Warn("will undo tx because it is beyond confirmed delay", "txid", global.F(unconfirmTx.Txid))
		}
		if hasConflict || tooDelayed {
			undoErr := uv.undoUnconfirmedTx(unconfirmTx, unconfirmTxMap,
				unconfirmTxGraph, batch, undoDone, nil)
			if undoErr != nil {
				uv.log.Warn("fail to undo tx", "undoErr", undoErr)
				return nil, nil, undoErr
			}
		}
	}
	if needRepost {
		go func() {
			sortTxList, unexpectedCyclic, dagSizeList := TopSortDFS(unconfirmTxGraph)
			if unexpectedCyclic {
				uv.log.Warn("transaction conflicted", "unexpectedCyclic", unexpectedCyclic)
				return
			}
			dagNo := 0
			uv.log.Info("parallel group of reposting", "dagGroupEach", dagSizeList)
			for start := 0; start < len(sortTxList); {
				dagsize := dagSizeList[dagNo]
				batchTx := []*pb.Transaction{}
				for _, txid := range sortTxList[start : start+dagsize] {
					if txidsInBlock[txid] || undoDone[txid] {
						continue
					}
					offlineTx := unconfirmTxMap[txid]
					batchTx = append(batchTx, offlineTx)
				}
				uv.OfflineTxChan <- batchTx
				start += dagsize
				dagNo++
			}
		}()
	}
	return unconfirmToConfirm, undoDone, nil
}

// Play do play and repost block
func (uv *UtxoVM) Play(blockid []byte) error {
	return uv.PlayAndRepost(blockid, false, true)
}

// PlayAndRepost 执行一个新收到的block，要求block的pre_hash必须是当前vm的latest_block
// 执行后会更新latestBlockid
func (uv *UtxoVM) PlayAndRepost(blockid []byte, needRepost bool, isRootTx bool) error {
	batch := uv.ldb.NewBatch()
	block, blockErr := uv.ledger.QueryBlock(blockid)
	if blockErr != nil {
		return blockErr
	}
	uv.mutex.Lock()
	defer uv.mutex.Unlock()
	// 下面开始处理unconfirmed的交易
	unconfirmToConfirm, undoDone, err := uv.processUnconfirmTxs(block, batch, needRepost)
	if err != nil {
		return err
	}

	ctx := &contract.TxContext{UtxoBatch: batch, Block: block, LedgerObj: uv.ledger, UtxoMeta: uv} // 将batch赋值到合约机的上下文
	uv.smartContract.SetContext(ctx)
	autoGenTxList, genErr := uv.GetVATList(block.Height, -1, block.Timestamp)
	if genErr != nil {
		uv.log.Warn("get autogen tx list failed", "err", genErr)
		return genErr
	}
	// 进入正题，开始执行block里面的交易，预期不会有冲突了
	uv.log.Debug("autogen tx list size, before play block", "len", len(autoGenTxList))
	idx, length := 0, len(block.Transactions)

	// parallel verify
	verifyErr := uv.verifyBlockTxs(block, isRootTx, unconfirmToConfirm)
	if verifyErr != nil {
		uv.log.Warn("verifyBlockTx error ", "err", verifyErr)
		return verifyErr
	}

	for idx < length {
		tx := block.Transactions[idx]
		txid := string(tx.Txid)
		if unconfirmToConfirm[txid] == false { // 本地没预执行过的Tx, 从block中收到的，需要Play执行
			cacheFiller := &CacheFiller{}
			err := uv.doTxInternal(tx, batch, cacheFiller)
			if err != nil {
				uv.log.Warn("dotx failed when Play", "txid", fmt.Sprintf("%x", tx.Txid), "err", err)
				return err
			}
			cacheFiller.Commit()
		}
		feeErr := uv.payFee(tx, batch, block)
		if feeErr != nil {
			uv.log.Warn("payFee failed", "feeErr", feeErr)
			return feeErr
		}
		//如果不是矿工的话，需要执行操作
		//合约的结果校验，任何错误都可能是作恶
		var cErr error
		if idx, cErr = uv.TxOfRunningContractVerify(batch, block, tx, &autoGenTxList, idx); cErr != nil {
			uv.log.Warn("TxOfRunningContractVerify failed when playing", "error", cErr, "idx", idx)
			return cErr
		}
	}
	uv.log.Debug("autogen tx list size, after play block", "len", len(autoGenTxList))
	if err := uv.smartContract.Finalize(block.Blockid); err != nil {
		uv.log.Warn("smart contract.finalize failed", "blockid", fmt.Sprintf("%x", block.Blockid))
		// 合约执行失败，不影响签发块
		return err
	}
	// 更新不可逆区块高度
	curIrreversibleBlockHeight := uv.GetIrreversibleBlockHeight()
	curIrreversibleSlideWindow := uv.GetIrreversibleSlideWindow()
	updateErr := uv.updateNextIrreversibleBlockHeight(block.Height, curIrreversibleBlockHeight, curIrreversibleSlideWindow, batch)
	if updateErr != nil {
		return updateErr
	}
	//更新latestBlockid
	persistErr := uv.updateLatestBlockid(block.Blockid, batch, "failed to save block")
	if persistErr != nil {
		return persistErr
	}
	//写盘成功再删除unconfirm的内存镜像
	for txid := range unconfirmToConfirm {
		uv.unconfirmTxInMem.Delete(txid)
	}
	for txid := range undoDone {
		uv.unconfirmTxInMem.Delete(txid)
	}
	// 内存级别更新UtxoMeta信息
	uv.mutexMeta.Lock()
	defer uv.mutexMeta.Unlock()
	newMeta := proto.Clone(uv.metaTmp).(*pb.UtxoMeta)
	uv.meta = newMeta
	return nil
}

// PlayForMiner 进行合约预执行
func (uv *UtxoVM) PlayForMiner(blockid []byte, batch kvdb.Batch) error {
	block, blockErr := uv.ledger.QueryBlock(blockid)
	if blockErr != nil {
		return blockErr
	}
	if !bytes.Equal(block.PreHash, uv.latestBlockid) {
		uv.log.Warn("play for miner failed", "block.PreHash", fmt.Sprintf("%x", block.PreHash),
			"latestBlockid", fmt.Sprintf("%x", uv.latestBlockid))
		return ErrPreBlockMissMatch
	}
	uv.mutex.Lock()
	defer uv.mutex.Unlock() // lock guard
	var err error
	defer func() {
		if err != nil {
			uv.clearBalanceCache()
		}
	}()
	for _, tx := range block.Transactions {
		txid := string(tx.Txid)
		if tx.Coinbase {
			err = uv.doTxInternal(tx, batch, nil)
			if err != nil {
				uv.log.Warn("dotx failed when PlayForMiner", "txid", fmt.Sprintf("%x", tx.Txid), "err", err)
				return err
			}
		} else {
			batch.Delete(append([]byte(pb.UnconfirmedTablePrefix), []byte(txid)...))
		}
		err = uv.payFee(tx, batch, block)
		if err != nil {
			uv.log.Warn("payFee failed", "feeErr", err)
			return err
		}
	}
	//继续PrePlayForMiner的合约上下文
	if err = uv.smartContract.Finalize(block.Blockid); err != nil {
		uv.log.Warn("smart contract.finalize failed", "blockid", fmt.Sprintf("%x", block.Blockid))
		return err
	}
	// 更新不可逆区块高度
	curIrreversibleBlockHeight := uv.GetIrreversibleBlockHeight()
	curIrreversibleSlideWindow := uv.GetIrreversibleSlideWindow()
	updateErr := uv.updateNextIrreversibleBlockHeight(block.Height, curIrreversibleBlockHeight, curIrreversibleSlideWindow, batch)
	if updateErr != nil {
		return updateErr
	}
	//更新latestBlockid
	err = uv.updateLatestBlockid(block.Blockid, batch, "failed to save block")
	if err != nil {
		return err
	}
	//写盘成功再清理unconfirm内存镜像
	for _, tx := range block.Transactions {
		uv.unconfirmTxInMem.Delete(string(tx.Txid))
	}
	// 内存级别更新UtxoMeta信息
	uv.mutexMeta.Lock()
	defer uv.mutexMeta.Unlock()
	newMeta := proto.Clone(uv.metaTmp).(*pb.UtxoMeta)
	uv.meta = newMeta
	return nil
}

// verifyAutogenTx verify if a autogen tx is valid, return true if tx is valid.
func (uv *UtxoVM) verifyAutogenTx(tx *pb.Transaction) bool {
	if !tx.Autogen {
		// not autogen tx, just return true
		return true
	}

	if len(tx.TxInputs) > 0 || len(tx.TxOutputs) > 0 {
		// autogen tx must have no tx inputs/outputs
		return false
	}

	if len(tx.TxInputsExt) > 0 || len(tx.TxOutputsExt) > 0 {
		// autogen tx must have no tx inputs/outputs extend
		return false
	}

	return true
}

// Walk 从当前的latestBlockid 游走到 blockid, 会触发utxo状态的回滚。执行后会更新latestBlockid
func (uv *UtxoVM) Walk(blockid []byte, ledgerPrune bool) error {
	uv.log.Info("utxoVM start walk.", "dest_block", hex.EncodeToString(blockid),
		"latest_blockid", hex.EncodeToString(uv.latestBlockid))

	xTimer := global.NewXTimer()

	// 获取全局锁
	uv.mutex.Lock()
	defer uv.mutex.Unlock()
	xTimer.Mark("walk_get_lock")

	// 首先先把所有的unconfirm回滚，记录被回滚的交易，然后walk结束后恢复被回滚的合法未确认交易
	undoDone, undoList, err := uv.RollBackUnconfirmedTx()
	if err != nil {
		uv.log.Warn("walk fail,rollback unconfirm tx fail", "err", err)
		return fmt.Errorf("walk rollback unconfirm tx fail")
	}
	xTimer.Mark("walk_rollback_unconfirm_tx")

	// 清理cache
	uv.clearBalanceCache()

	// 寻找blockid和latestBlockid的最低公共祖先, 生成undoBlocks和todoBlocks
	undoBlocks, todoBlocks, err := uv.ledger.FindUndoAndTodoBlocks(uv.latestBlockid, blockid)
	if err != nil {
		uv.log.Warn("walk fail,find common parent block fail", "dest_block", hex.EncodeToString(blockid),
			"latest_block", hex.EncodeToString(uv.latestBlockid), "err", err)
		return fmt.Errorf("walk find common parent block fail")
	}
	xTimer.Mark("walk_find_undo_todo_block")

	// utxoVM回滚需要回滚区块
	err = uv.procUndoBlkForWalk(undoBlocks, undoDone, ledgerPrune)
	if err != nil {
		uv.log.Warn("walk fail,because undo block fail", "err", err)
		return fmt.Errorf("walk undo block fail")
	}
	xTimer.Mark("walk_undo_block")

	// utxoVM执行需要执行区块
	err = uv.procTodoBlkForWalk(todoBlocks)
	if err != nil {
		uv.log.Warn("walk fail,because todo block fail", "err", err)
		return fmt.Errorf("walk todo block fail")
	}
	xTimer.Mark("walk_todo_block")

	// 异步回放被回滚未确认交易
	go uv.recoverUnconfirmedTx(undoList)

	uv.log.Info("utxoVM walk finish", "dest_block", hex.EncodeToString(blockid),
		"latest_blockid", hex.EncodeToString(uv.latestBlockid), "costs", xTimer.Print())
	return nil
}

// utxoVM重放未确认交易，失败仅仅日志记录
func (uv *UtxoVM) recoverUnconfirmedTx(undoList TxLists) {
	xTimer := global.NewXTimer()
	uv.log.Info("start recover unconfirm tx", "tx_count", len(undoList))

	var tx *pb.Transaction
	var succCnt, verifyErrCnt, confirmCnt, doTxErrCnt int
	// 由于未确认交易也可能存在依赖顺序，需要按依赖顺序回放交易
	for i := len(undoList) - 1; i >= 0; i-- {
		tx = undoList[i]
		// 过滤挖矿奖励和自动生成交易，理论上挖矿奖励和自动生成交易不会进入未确认交易池
		if tx.Coinbase || tx.Autogen {
			continue
		}

		// 检查交易是否已经被确认（被其他节点打包倒区块并广播了过来）
		isConfirm, err := uv.ledger.HasTransaction(tx.Txid)
		if err != nil && isConfirm {
			confirmCnt++
			uv.log.Info("this tx has been confirmed,ignore recover", "txid", hex.EncodeToString(tx.Txid))
			continue
		}

		uv.log.Info("start recover unconfirm tx", "txid", hex.EncodeToString(tx.Txid))
		// 重新对交易鉴权，过掉冲突交易
		isValid, err := uv.ImmediateVerifyTx(tx, false)
		if err != nil || !isValid {
			verifyErrCnt++
			uv.log.Info("this tx immediate verify fail,ignore recover", "txid",
				hex.EncodeToString(tx.Txid), "is_valid", isValid, "err", err)
			continue
		}

		// 重新提交交易，可能交易已经被其他节点打包到区块广播过来，导致失败
		err = uv.doTxSync(tx)
		if err != nil {
			doTxErrCnt++
			uv.log.Info("dotx fail for recover unconfirm tx,ignore recover this tx",
				"txid", hex.EncodeToString(tx.Txid), "err", err)
			continue
		}

		succCnt++
		uv.log.Info("recover unconfirm tx succ", "txid", hex.EncodeToString(tx.Txid))
	}

	uv.log.Info("recover unconfirm tx done", "costs", xTimer.Print(), "tx_count", len(undoList),
		"succ_count", succCnt, "confirm_count", confirmCnt, "verify_err_count",
		verifyErrCnt, "dotx_err_cnt", doTxErrCnt)
}

// utxoVM批量回滚区块
func (uv *UtxoVM) procUndoBlkForWalk(undoBlocks []*pb.InternalBlock,
	undoDone map[string]bool, ledgerPrune bool) (err error) {
	var undoBlk *pb.InternalBlock
	var showBlkId string
	var tx *pb.Transaction
	var showTxId string

	// 依次回滚每个区块
	for _, undoBlk = range undoBlocks {
		showBlkId = hex.EncodeToString(undoBlk.Blockid)
		uv.log.Info("start undo block for walk", "blockid", showBlkId)

		// 加一个(共识)开关来判定是否需要采用不可逆
		// 不需要更新IrreversibleBlockHeight以及SlideWindow，因为共识层面的回滚不会回滚到
		// IrreversibleBlockHeight，只有账本裁剪才需要更新IrreversibleBlockHeight以及SlideWindow
		curIrreversibleBlockHeight := uv.GetIrreversibleBlockHeight()
		if !ledgerPrune && undoBlk.Height <= curIrreversibleBlockHeight {
			return fmt.Errorf("block to be undo is older than irreversibleBlockHeight."+
				"irreversible_height:%d,undo_block_height:%d", curIrreversibleBlockHeight, undoBlk.Height)
		}

		// 将batch赋值到合约机的上下文
		batch := uv.ldb.NewBatch()
		ctx := &contract.TxContext{
			UtxoBatch: batch,
			Block:     undoBlk,
			IsUndo:    true,
			LedgerObj: uv.ledger,
			UtxoMeta:  uv,
		}
		uv.smartContract.SetContext(ctx)

		// 倒序回滚交易
		for i := len(undoBlk.Transactions) - 1; i >= 0; i-- {
			tx = undoBlk.Transactions[i]
			showTxId = hex.EncodeToString(tx.Txid)

			// 回滚交易
			if !undoDone[string(tx.Txid)] {
				err = uv.undoTxInternal(tx, batch)
				if err != nil {
					return fmt.Errorf("undo tx fail.txid:%s,err:%v", showTxId, err)
				}
			}

			// 回滚小费，undoTxInternal不会滚小费
			err = uv.undoPayFee(tx, batch, undoBlk)
			if err != nil {
				return fmt.Errorf("undo fee fail.txid:%s,err:%v", showTxId, err)
			}

			// 二代合约回滚，回滚失败只是日志记录
			err = uv.RollbackContract(undoBlk.Blockid, tx)
			if err != nil {
				uv.log.Warn("failed to rollback contract, when undo block", "err", err)
			}
		}

		if err = uv.smartContract.Finalize(undoBlk.PreHash); err != nil {
			return fmt.Errorf("smart contract fianlize fail.blockid:%s,err:%v", showBlkId, err)
		}

		// 账本裁剪时，无视区块不可逆原则
		if ledgerPrune {
			curIrreversibleBlockHeight := uv.GetIrreversibleBlockHeight()
			curIrreversibleSlideWindow := uv.GetIrreversibleSlideWindow()
			err = uv.updateNextIrreversibleBlockHeightForPrune(undoBlk.Height,
				curIrreversibleBlockHeight, curIrreversibleSlideWindow, batch)
			if err != nil {
				return fmt.Errorf("update irreversible block height fail.err:%v", err)
			}
		}

		// 更新utxoVM LatestBlockid，这里是回滚，所以是更新为上一个区块
		err = uv.updateLatestBlockid(undoBlk.PreHash, batch, "error occurs when undo blocks")
		if err != nil {
			return fmt.Errorf("update latest blockid fail.latest_blockid:%s,err:%v",
				hex.EncodeToString(undoBlk.PreHash), err)
		}

		// 每回滚完一个块，内存级别更新UtxoMeta信息
		uv.mutexMeta.Lock()
		newMeta := proto.Clone(uv.metaTmp).(*pb.UtxoMeta)
		uv.meta = newMeta
		uv.mutexMeta.Unlock()

		uv.log.Info("finish undo this block", "blockid", showBlkId)
	}

	return nil
}

// utxoVM批量执行区块
func (uv *UtxoVM) procTodoBlkForWalk(todoBlocks []*pb.InternalBlock) (err error) {
	var todoBlk *pb.InternalBlock
	var showBlkId string
	var tx *pb.Transaction
	var showTxId string

	// 依次执行每个块的交易
	for i := len(todoBlocks) - 1; i >= 0; i-- {
		todoBlk = todoBlocks[i]
		showBlkId = hex.EncodeToString(todoBlk.Blockid)

		uv.log.Info("start do block for walk", "blockid", showBlkId)
		// 将batch赋值到合约机的上下文
		batch := uv.ldb.NewBatch()
		ctx := &contract.TxContext{UtxoBatch: batch, Block: todoBlk, LedgerObj: uv.ledger, UtxoMeta: uv}
		uv.smartContract.SetContext(ctx)
		autoGenTxList, err := uv.GetVATList(todoBlk.Height, -1, todoBlk.Timestamp)
		if err != nil {
			return fmt.Errorf("get autogen tx list failed.blockid:%s,err:%v", showBlkId, err)
		}

		// 执行区块里面的交易
		idx, length := 0, len(todoBlk.Transactions)
		for idx < length {
			tx = todoBlk.Transactions[idx]
			showTxId = hex.EncodeToString(tx.Txid)
			// 校验交易合法性
			if !tx.Autogen && !tx.Coinbase {
				if ok, err := uv.ImmediateVerifyTx(tx, false); !ok {
					return fmt.Errorf("immediate verify tx error.txid:%s,err:%v", showTxId, err)
				}
			}

			// 执行交易
			cacheFiller := &CacheFiller{}
			err = uv.doTxInternal(tx, batch, cacheFiller)
			if err != nil {
				return fmt.Errorf("todo tx fail.txid:%s,err:%v", showTxId, err)
			}
			cacheFiller.Commit()

			// 处理小费
			err = uv.payFee(tx, batch, todoBlk)
			if err != nil {
				return fmt.Errorf("pay fee fail.txid:%s,err:%v", showTxId, err)
			}

			// 执行二代合约
			idx, err = uv.TxOfRunningContractVerify(batch, todoBlk, tx, &autoGenTxList, idx)
			if err != nil {
				return fmt.Errorf("run tx contract fail.txid:%s,err:%v", showTxId, err)
			}
		}

		uv.log.Debug("Begin to Finalize", "blockid", showBlkId)
		if err = uv.smartContract.Finalize(todoBlk.Blockid); err != nil {
			return fmt.Errorf("smart contract fianlize fail.blockid:%s,err:%v", showBlkId, err)
		}

		// 更新不可逆区块高度
		curIrreversibleBlockHeight := uv.GetIrreversibleBlockHeight()
		curIrreversibleSlideWindow := uv.GetIrreversibleSlideWindow()
		err = uv.updateNextIrreversibleBlockHeight(todoBlk.Height, curIrreversibleBlockHeight,
			curIrreversibleSlideWindow, batch)
		if err != nil {
			return fmt.Errorf("update irreversible height fail.blockid:%s,err:%v", showBlkId, err)
		}
		// 每do一个block,是一个原子batch写
		err = uv.updateLatestBlockid(todoBlk.Blockid, batch, "error occurs when do blocks")
		if err != nil {
			return fmt.Errorf("update last blockid fail.blockid:%s,err:%v", showBlkId, err)
		}

		// 完成一个区块后，内存级别更新UtxoMeta信息
		uv.mutexMeta.Lock()
		newMeta := proto.Clone(uv.metaTmp).(*pb.UtxoMeta)
		uv.meta = newMeta
		uv.mutexMeta.Unlock()

		uv.log.Info("finish todo this block", "blockid", showBlkId)
	}

	return nil
}

//获得一个账号的余额，inLock表示在调用此函数时已经对uv.mutex加过锁了
func (uv *UtxoVM) getBalance(addr string) (*big.Int, error) {
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

// QueryTxFromForbiddenWithConfirmed query if the tx has been forbidden
func (uv *UtxoVM) QueryTxFromForbiddenWithConfirmed(txid []byte) (bool, bool, error) {
	return uv.queryTxFromForbiddenWithConfirmed(txid)
}

// GetBalance 查询Address的可用余额
func (uv *UtxoVM) GetBalance(addr string) (*big.Int, error) {
	return uv.getBalance(addr)
}

// Close 关闭utxo vm, 目前主要是关闭leveldb
func (uv *UtxoVM) Close() {
	uv.smartContract.Stop()
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
	uv.utxoCache.Remove(address, utxoKey)
}

// GetVATList return the registered VAT list
func (uv *UtxoVM) GetVATList(blockHeight int64, maxCount int, timestamp int64) ([]*pb.Transaction, error) {
	txs := []*pb.Transaction{}
	for i := 0; i < len(uv.vatHandler.HandlerList); i++ {
		name := uv.vatHandler.HandlerList[i]
		vats, err := uv.vatHandler.Handlers[name].GetVerifiableAutogenTx(blockHeight, maxCount, timestamp)
		if err != nil {
			uv.log.Warn("GetVATList error", "err", err)
			continue
		}
		if vats != nil {
			txs = append(txs, vats...)
		}
	}
	return txs, nil
}

// MustVAT must VAT
func (uv *UtxoVM) MustVAT(desc *contract.TxDesc) bool {
	if desc.Module == "" {
		return false //不是合约,跳过
	}
	return uv.vatHandler.MustVAT(desc.Module, desc.Method)
}

// NewBatch return batch instance
func (uv *UtxoVM) NewBatch() kvdb.Batch {
	return uv.ldb.NewBatch()
}

// GetXModel return the instance of XModel
func (uv *UtxoVM) GetXModel() *xmodel.XModel {
	return uv.model3
}

func (uv *UtxoVM) GetSnapShotWithBlock(blockId []byte) (xmodel.XMReader, error) {
	reader, err := uv.model3.CreateSnapshot(blockId)
	return reader, err
}

// SetMaxConfirmedDelay set the max value of tx confirm delay. If beyond, tx will be rollbacked
func (uv *UtxoVM) SetMaxConfirmedDelay(seconds uint32) {
	uv.maxConfirmedDelay = seconds
	uv.log.Info("set max confirmed delay of tx", "seconds", seconds)
}

// SetModifyBlockAddr set modified block addr
func (uv *UtxoVM) SetModifyBlockAddr(addr string) {
	uv.modifyBlockAddr = addr
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
