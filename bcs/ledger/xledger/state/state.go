// 统一定义状态机对外暴露功能
package state

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/meta"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/xmodel"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	governToken "github.com/xuperchain/xupercore/kernel/contract/proposal/govern_token"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/propose"
	timerTask "github.com/xuperchain/xupercore/kernel/contract/proposal/timer"
	kledger "github.com/xuperchain/xupercore/kernel/ledger"
	aclBase "github.com/xuperchain/xupercore/kernel/permission/acl/base"
	"github.com/xuperchain/xupercore/lib/cache"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

var (
	ErrDoubleSpent          = errors.New("utxo can not be spent more than once")
	ErrAlreadyInUnconfirmed = errors.New("this transaction is in unconfirmed state")
	ErrPreBlockMissMatch    = errors.New("play block failed because pre-hash != latest_block")
	ErrUnexpected           = errors.New("this is a unexpected error")
	ErrInvalidAutogenTx     = errors.New("found invalid autogen-tx")
	ErrUTXODuplicated       = errors.New("found duplicated utxo in same tx")
	ErrRWSetInvalid         = errors.New("RWSet of transaction invalid")
	ErrACLNotEnough         = errors.New("ACL not enough")
	ErrInvalidSignature     = errors.New("the signature is invalid or not match the address")

	ErrGasNotEnough   = errors.New("Gas not enough")
	ErrVersionInvalid = errors.New("Invalid tx version")
	ErrInvalidAccount = errors.New("Invalid account")
	ErrInvalidTxExt   = errors.New("Invalid tx ext")
	ErrTxTooLarge     = errors.New("Tx size is too large")

	ErrParseContractUtxos   = errors.New("Parse contract utxos error")
	ErrContractTxAmout      = errors.New("Contract transfer amount error")
	ErrGetReservedContracts = errors.New("Get reserved contracts error")
)

const (
	LatestBlockKey = "pointer"

	// BetaTxVersion 为当前代码支持的最高交易版本
	BetaTxVersion  = 3
	RootTxVersion  = 0
	FeePlaceholder = "$"
	// TxSizePercent max percent of txs' size in one block
	TxSizePercent = 0.8

	TxWaitTimeout = 5
)

type State struct {
	// 状态机运行环境上下文
	sctx          *context.StateCtx
	log           logs.Logger
	utxo          *utxo.UtxoVM   //utxo表
	xmodel        *xmodel.XModel //xmodel数据表和历史表
	meta          *meta.Meta     //meta表
	tx            *tx.Tx         //未确认交易表
	ldb           kvdb.Database
	latestBlockid []byte

	// 最新区块高度通知装置
	heightNotifier *BlockHeightNotifier
}

func NewState(sctx *context.StateCtx) (*State, error) {
	if sctx == nil {
		return nil, fmt.Errorf("create state failed because context set error")
	}

	obj := &State{
		sctx: sctx,
		log:  sctx.XLog,
	}

	var err error
	storePath := sctx.EnvCfg.GenDataAbsPath(sctx.EnvCfg.ChainDir)
	storePath = filepath.Join(storePath, sctx.BCName)
	stateDBPath := filepath.Join(storePath, def.StateStrgDirName)
	kvParam := &kvdb.KVParameter{
		DBPath:                stateDBPath,
		KVEngineType:          sctx.LedgerCfg.KVEngineType,
		MemCacheSize:          ledger.MemCacheSize,
		FileHandlersCacheSize: ledger.FileHandlersCacheSize,
		OtherPaths:            sctx.LedgerCfg.OtherPaths,
		StorageType:           sctx.LedgerCfg.StorageType,
	}
	obj.ldb, err = kvdb.CreateKVInstance(kvParam)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create ldb error:%s", err)
	}

	obj.xmodel, err = xmodel.NewXModel(sctx, obj.ldb)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create xmodel error:%s", err)
	}

	obj.meta, err = meta.NewMeta(sctx, obj.ldb)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create meta error:%s", err)
	}

	obj.utxo, err = utxo.MakeUtxo(sctx, obj.meta, sctx.LedgerCfg.Utxo.CacheSize,
		sctx.LedgerCfg.Utxo.TmpLockSeconds, obj.ldb)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create utxo error:%s", err)
	}

	obj.tx, err = tx.NewTx(sctx, obj.ldb)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create tx error:%s", err)
	}

	latestBlockid, findErr := obj.meta.MetaTable.Get([]byte(LatestBlockKey))
	if findErr == nil {
		obj.latestBlockid = latestBlockid
	} else {
		if def.NormalizedKVError(findErr) != def.ErrKVNotFound {
			return nil, findErr
		}
	}

	loadErr := obj.tx.LoadUnconfirmedTxFromDisk()
	if loadErr != nil {
		return nil, loadErr
	}

	obj.heightNotifier = NewBlockHeightNotifier()

	return obj, nil
}

func (t *State) SetAclMG(aclMgr aclBase.AclManager) {
	t.sctx.SetAclMG(aclMgr)
}

func (t *State) SetContractMG(contractMgr contract.Manager) {
	t.sctx.SetContractMG(contractMgr)
}

func (t *State) SetGovernTokenMG(governTokenMgr governToken.GovManager) {
	t.sctx.SetGovernTokenMG(governTokenMgr)
}

func (t *State) SetProposalMG(proposalMgr propose.ProposeManager) {
	t.sctx.SetProposalMG(proposalMgr)
}

func (t *State) SetTimerTaskMG(timerTaskMgr timerTask.TimerManager) {
	t.sctx.SetTimerTaskMG(timerTaskMgr)
}

// 选择足够金额的utxo
func (t *State) SelectUtxos(fromAddr string, totalNeed *big.Int, needLock, excludeUnconfirmed bool) ([]*protos.TxInput, [][]byte, *big.Int, error) {
	return t.utxo.SelectUtxos(fromAddr, totalNeed, needLock, excludeUnconfirmed)
}

// 获取一批未确认交易（用于矿工打包区块）
func (t *State) GetUnconfirmedTx(dedup bool) ([]*pb.Transaction, error) {
	return t.tx.GetUnconfirmedTx(dedup)
}

func (t *State) GetLatestBlockid() []byte {
	return t.latestBlockid
}

func (t *State) QueryUtxoRecord(accountName string, displayCount int64) (*pb.UtxoRecordDetail, error) {
	return t.utxo.QueryUtxoRecord(accountName, displayCount)
}

func (t *State) SelectUtxosBySize(fromAddr string, needLock, excludeUnconfirmed bool) ([]*protos.TxInput, [][]byte, *big.Int, error) {
	return t.utxo.SelectUtxosBySize(fromAddr, needLock, excludeUnconfirmed)
}

func (t *State) QueryContractStatData() (*protos.ContractStatData, error) {
	return t.utxo.QueryContractStatData()
}

func (t *State) GetAccountContracts(account string) ([]string, error) {
	return t.utxo.GetAccountContracts(account)
}

// 查询合约状态
func (t *State) GetContractStatus(contractName string) (*protos.ContractStatus, error) {
	res := &protos.ContractStatus{}
	res.ContractName = contractName
	verdata, err := t.xmodel.Get("contract", bridge.ContractCodeDescKey(contractName))
	if err != nil {
		t.log.Warn("GetContractStatus get version data error", "error", err.Error())
		return nil, err
	}
	txid := verdata.GetRefTxid()
	res.Txid = fmt.Sprintf("%x", txid)
	tx, _, err := t.xmodel.QueryTx(txid)
	if err != nil {
		t.log.Warn("GetContractStatus query tx error", "error", err.Error())
		return nil, err
	}
	res.Desc = tx.GetDesc()
	res.Timestamp = tx.GetReceivedTimestamp()
	// query if contract is bannded
	res.IsBanned, err = t.queryContractBannedStatus(contractName)
	return res, nil
}

func (t *State) QueryAccountACL(accountName string) (*protos.Acl, error) {
	return t.sctx.AclMgr.GetAccountACL(accountName)
}

func (t *State) QueryContractMethodACL(contractName string, methodName string) (*protos.Acl, error) {
	return t.sctx.AclMgr.GetContractMethodACL(contractName, methodName)
}

func (t *State) QueryAccountContainAK(address string) ([]string, error) {
	return t.utxo.QueryAccountContainAK(address)
}

func (t *State) QueryAccountGovernTokenBalance(accountName string) (*protos.GovernTokenBalance, error) {
	return t.sctx.GovernTokenMgr.GetGovTokenBalance(accountName)
}

// HasTx 查询一笔交易是否在unconfirm表  这些可能是放在tx对外提供
func (t *State) HasTx(txid []byte) (bool, error) {
	_, exist := t.tx.UnconfirmTxInMem.Load(string(txid))
	return exist, nil
}

func (t *State) GetFrozenBalance(addr string) (*big.Int, error) {
	addrPrefix := fmt.Sprintf("%s%s_", pb.UTXOTablePrefix, addr)
	utxoFrozen := big.NewInt(0)
	curHeight := t.sctx.Ledger.GetMeta().TrunkHeight
	it := t.ldb.NewIteratorWithPrefix([]byte(addrPrefix))
	defer it.Release()
	for it.Next() {
		uBinary := it.Value()
		uItem := &utxo.UtxoItem{}
		uErr := uItem.Loads(uBinary)
		if uErr != nil {
			return nil, uErr
		}
		if uItem.FrozenHeight <= curHeight && uItem.FrozenHeight != -1 {
			continue
		}
		utxoFrozen.Add(utxoFrozen, uItem.Amount) // utxo累加
	}
	if it.Error() != nil {
		return nil, it.Error()
	}
	return utxoFrozen, nil
}

// GetFrozenBalance 查询Address的被冻结的余额 / 未冻结的余额
func (t *State) GetBalanceDetail(addr string) ([]*pb.BalanceDetailInfo, error) {
	addrPrefix := fmt.Sprintf("%s%s_", pb.UTXOTablePrefix, addr)
	utxoFrozen := big.NewInt(0)
	utxoUnFrozen := big.NewInt(0)
	curHeight := t.sctx.Ledger.GetMeta().TrunkHeight
	it := t.ldb.NewIteratorWithPrefix([]byte(addrPrefix))
	defer it.Release()
	for it.Next() {
		uBinary := it.Value()
		uItem := &utxo.UtxoItem{}
		uErr := uItem.Loads(uBinary)
		if uErr != nil {
			return nil, uErr
		}
		if uItem.FrozenHeight <= curHeight && uItem.FrozenHeight != -1 {
			utxoUnFrozen.Add(utxoUnFrozen, uItem.Amount) // utxo累加
			continue
		}
		utxoFrozen.Add(utxoFrozen, uItem.Amount) // utxo累加
	}
	if it.Error() != nil {
		return nil, it.Error()
	}

	var tokenFrozenDetails []*pb.BalanceDetailInfo

	tokenFrozenDetail := &pb.BalanceDetailInfo{
		Balance:  utxoFrozen.String(),
		IsFrozen: true,
	}
	tokenFrozenDetails = append(tokenFrozenDetails, tokenFrozenDetail)

	tokenUnFrozenDetail := &pb.BalanceDetailInfo{
		Balance:  utxoUnFrozen.String(),
		IsFrozen: false,
	}
	tokenFrozenDetails = append(tokenFrozenDetails, tokenUnFrozenDetail)

	return tokenFrozenDetails, nil
}

// 校验交易
// VerifyTx check the tx signature and permission
func (t *State) VerifyTx(tx *pb.Transaction) (bool, error) {
	isValid, err := t.ImmediateVerifyTx(tx, false)
	if err != nil || !isValid {
		t.log.Warn("ImmediateVerifyTx failed", "error", err,
			"AuthRequire ", tx.AuthRequire, "AuthRequireSigns ", tx.AuthRequireSigns,
			"Initiator", tx.Initiator, "InitiatorSigns", tx.InitiatorSigns, "XuperSign", tx.XuperSign)
		ok, isRelyOnMarkedTx, err := t.verifyMarked(tx)
		if isRelyOnMarkedTx {
			if !ok || err != nil {
				t.log.Warn("tx verification failed because it is blocked tx", "err", err)
			} else {
				t.log.Trace("blocked tx verification succeed")
			}
			return ok, err
		}
	}
	return isValid, err
}

// 执行交易
func (t *State) DoTx(tx *pb.Transaction) error {
	tx.ReceivedTimestamp = time.Now().UnixNano()
	if tx.Coinbase {
		t.log.Warn("coinbase tx can not be given by PostTx", "txid", utils.F(tx.Txid))
		return ErrUnexpected
	}
	if len(tx.Blockid) > 0 {
		t.log.Warn("tx from PostTx must not have blockid", "txid", utils.F(tx.Txid))
		return ErrUnexpected
	}
	return t.doTxSync(tx)
}

// 创建获取最新状态数据XMReader
func (t *State) CreateXMReader() kledger.XMReader {
	return t.xmodel
}

// 根据指定blockid创建快照（Select方法不可用）
func (t *State) CreateSnapshot(blkId []byte) (kledger.XMReader, error) {
	return t.xmodel.CreateSnapshot(blkId)
}

// 获取最新确认高度快照（Select方法不可用）
func (t *State) GetTipSnapshot() (kledger.XMReader, error) {
	return t.CreateSnapshot(t.latestBlockid)
}

// 根据指定blockid创建快照（相比XMReader，只有Get方法，直接返回[]byte）
func (t *State) CreateXMSnapshotReader(blkId []byte) (kledger.XMSnapshotReader, error) {
	return t.xmodel.CreateXMSnapshotReader(blkId)
}

// 获取状态机最新确认高度快照（相比XMReader，只有Get方法，直接返回[]byte）
func (t *State) GetTipXMSnapshotReader() (kledger.XMSnapshotReader, error) {
	return t.CreateXMSnapshotReader(t.latestBlockid)
}

func (t *State) BucketCacheDelete(bucket, version string) {
	t.xmodel.BucketCacheDelete(bucket, version)
}

// 执行区块
func (t *State) Play(blockid []byte) error {
	return t.PlayAndRepost(blockid, false, true)
}

func (t *State) PlayForMiner(blockid []byte) error {
	batch := t.NewBatch()
	block, blockErr := t.sctx.Ledger.QueryBlock(blockid)
	if blockErr != nil {
		return blockErr
	}
	if !bytes.Equal(block.PreHash, t.latestBlockid) {
		t.log.Warn("play for miner failed", "block.PreHash", utils.F(block.PreHash),
			"latestBlockid", fmt.Sprintf("%x", t.latestBlockid))
		return ErrPreBlockMissMatch
	}
	t.utxo.Mutex.Lock()
	defer t.utxo.Mutex.Unlock() // lock guard
	var err error
	defer func() {
		if err != nil {
			t.clearBalanceCache()
		}
	}()
	for _, tx := range block.Transactions {
		txid := string(tx.Txid)
		if tx.Coinbase || tx.Autogen {
			err = t.doTxInternal(tx, batch, nil)
			if err != nil {
				t.log.Warn("dotx failed when PlayForMiner", "txid", utils.F(tx.Txid), "err", err)
				return err
			}
		} else {
			batch.Delete(append([]byte(pb.UnconfirmedTablePrefix), []byte(txid)...))
		}
		err = t.payFee(tx, batch, block)
		if err != nil {
			t.log.Warn("payFee failed", "feeErr", err)
			return err
		}
	}
	// 更新不可逆区块高度
	curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
	curIrreversibleSlideWindow := t.meta.GetIrreversibleSlideWindow()
	updateErr := t.meta.UpdateNextIrreversibleBlockHeight(block.Height, curIrreversibleBlockHeight, curIrreversibleSlideWindow, batch)
	if updateErr != nil {
		return updateErr
	}
	//更新latestBlockid
	err = t.updateLatestBlockid(block.Blockid, batch, "failed to save block")
	if err != nil {
		return err
	}
	//写盘成功再清理unconfirm内存镜像
	for _, tx := range block.Transactions {
		t.tx.UnconfirmTxInMem.Delete(string(tx.Txid))
	}
	// 内存级别更新UtxoMeta信息
	t.meta.MutexMeta.Lock()
	defer t.meta.MutexMeta.Unlock()
	newMeta := proto.Clone(t.meta.MetaTmp).(*pb.UtxoMeta)
	t.meta.Meta = newMeta
	return nil
}

// 执行和发送区块
// PlayAndRepost 执行一个新收到的block，要求block的pre_hash必须是当前vm的latest_block
// 执行后会更新latestBlockid
func (t *State) PlayAndRepost(blockid []byte, needRepost bool, isRootTx bool) error {
	batch := t.ldb.NewBatch()
	block, blockErr := t.sctx.Ledger.QueryBlock(blockid)
	if blockErr != nil {
		return blockErr
	}
	t.utxo.Mutex.Lock()
	defer t.utxo.Mutex.Unlock()
	// 下面开始处理unconfirmed的交易
	unconfirmToConfirm, undoDone, err := t.processUnconfirmTxs(block, batch, needRepost)
	if err != nil {
		return err
	}

	// parallel verify
	verifyErr := t.verifyBlockTxs(block, isRootTx, unconfirmToConfirm)
	if verifyErr != nil {
		t.log.Warn("verifyBlockTx error ", "err", verifyErr)
		return verifyErr
	}
	t.log.Debug("play and repost verify block tx succ")

	for idx := 0; idx < len(block.Transactions); idx++ {
		tx := block.Transactions[idx]
		txid := string(tx.Txid)
		if unconfirmToConfirm[txid] == false { // 本地没预执行过的Tx, 从block中收到的，需要Play执行
			cacheFiller := &utxo.CacheFiller{}
			err := t.doTxInternal(tx, batch, cacheFiller)
			if err != nil {
				t.log.Warn("dotx failed when Play", "txid", utils.F(tx.Txid), "err", err)
				return err
			}
			cacheFiller.Commit()
		}
		feeErr := t.payFee(tx, batch, block)
		if feeErr != nil {
			t.log.Warn("payFee failed", "feeErr", feeErr)
			return feeErr
		}
	}
	// 更新不可逆区块高度
	curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
	curIrreversibleSlideWindow := t.meta.GetIrreversibleSlideWindow()
	updateErr := t.meta.UpdateNextIrreversibleBlockHeight(block.Height, curIrreversibleBlockHeight, curIrreversibleSlideWindow, batch)
	if updateErr != nil {
		return updateErr
	}
	//更新latestBlockid
	persistErr := t.updateLatestBlockid(block.Blockid, batch, "failed to save block")
	if persistErr != nil {
		return persistErr
	}
	//写盘成功再删除unconfirm的内存镜像
	for txid := range unconfirmToConfirm {
		t.tx.UnconfirmTxInMem.Delete(txid)
	}
	for txid := range undoDone {
		t.tx.UnconfirmTxInMem.Delete(txid)
	}
	t.log.Debug("write to state succ")

	// 内存级别更新UtxoMeta信息
	t.meta.MutexMeta.Lock()
	newMeta := proto.Clone(t.meta.MetaTmp).(*pb.UtxoMeta)
	t.meta.Meta = newMeta
	t.meta.MutexMeta.Unlock()

	t.log.Debug("paly and repost succ", "blockId", utils.F(block.Blockid))

	return nil
}

func (t *State) GetTimerTx(blockHeight int64) (*pb.Transaction, error) {
	stateConfig := &contract.SandboxConfig{
		XMReader: t.CreateXMReader(),
	}
	if !t.sctx.IsInit() {
		return nil, nil
	}
	t.log.Info("GetTimerTx", "blockHeight", blockHeight)
	sandBox, err := t.sctx.ContractMgr.NewStateSandbox(stateConfig)
	if err != nil {
		t.log.Error("PreExec new state sandbox error", "error", err)
		return nil, err
	}

	contextConfig := &contract.ContextConfig{
		State:       sandBox,
		Initiator:   "",
		AuthRequire: nil,
	}

	args := make(map[string][]byte)
	args["block_height"] = []byte(strconv.FormatInt(blockHeight, 10))
	req := protos.InvokeRequest{
		ModuleName:   "xkernel",
		ContractName: "$timer_task",
		MethodName:   "Do",
		Args:         args,
	}

	contextConfig.ResourceLimits = contract.MaxLimits
	contextConfig.Module = req.ModuleName
	contextConfig.ContractName = req.GetContractName()

	ctx, err := t.sctx.ContractMgr.NewContext(contextConfig)
	if err != nil {
		t.log.Error("GetTimerTx NewContext error", "err", err, "contractName", req.GetContractName())
		return nil, err
	}

	ctxResponse, ctxErr := ctx.Invoke(req.MethodName, req.Args)
	if ctxErr != nil {
		ctx.Release()
		t.log.Error("GetTimerTx Invoke error", "error", ctxErr, "contractName", req.GetContractName())
		return nil, ctxErr
	}
	// 判断合约调用的返回码
	if ctxResponse.Status >= 400 {
		ctx.Release()
		t.log.Error("GetTimerTx Invoke error", "status", ctxResponse.Status, "contractName", req.GetContractName())
		return nil, errors.New(ctxResponse.Message)
	}

	ctx.Release()

	rwSet := sandBox.RWSet()
	if rwSet == nil {
		return nil, nil
	}
	inputs := xmodel.GetTxInputs(rwSet.RSet)
	outputs := xmodel.GetTxOutputs(rwSet.WSet)

	autoTx, err := tx.GenerateAutoTxWithRWSets(inputs, outputs)
	if err != nil {
		return nil, err
	}

	t.log.Trace("GetTimerTx", "readSet", rwSet.RSet, "writeSet", rwSet.WSet)

	return autoTx, nil
}

// 回滚全部未确认交易
func (t *State) RollBackUnconfirmedTx() (map[string]bool, []*pb.Transaction, error) {
	// 分析依赖关系
	batch := t.NewBatch()
	unconfirmTxMap, unconfirmTxGraph, _, loadErr := t.tx.SortUnconfirmedTx()
	if loadErr != nil {
		return nil, nil, loadErr
	}

	// 回滚未确认交易
	undoDone := make(map[string]bool)
	undoList := make([]*pb.Transaction, 0)
	for txid, unconfirmTx := range unconfirmTxMap {
		undoErr := t.undoUnconfirmedTx(unconfirmTx, unconfirmTxMap, unconfirmTxGraph,
			batch, undoDone, &undoList)
		if undoErr != nil {
			t.log.Warn("fail to undo tx", "undoErr", undoErr, "txid", fmt.Sprintf("%x", txid))
			return nil, nil, undoErr
		}
	}

	// 原子写
	writeErr := batch.Write()
	if writeErr != nil {
		t.ClearCache()
		t.log.Warn("failed to clean unconfirmed tx", "writeErr", writeErr)
		return nil, nil, writeErr
	}

	// 由于这里操作不是原子操作，需要保持按回滚顺序delete
	for _, tx := range undoList {
		t.tx.UnconfirmTxInMem.Delete(string(tx.Txid))
		t.log.Trace("delete from unconfirm tx memory", "txid", utils.F(tx.Txid))
	}
	return undoDone, undoList, nil
}

// 同步账本和状态机
func (t *State) Walk(blockid []byte, ledgerPrune bool) error {
	t.log.Info("utxoVM start walk.", "dest_block", hex.EncodeToString(blockid),
		"latest_blockid", hex.EncodeToString(t.latestBlockid))

	xTimer := timer.NewXTimer()

	// 获取全局锁
	t.utxo.Mutex.Lock()
	defer t.utxo.Mutex.Unlock()
	xTimer.Mark("walk_get_lock")

	// 首先先把所有的unconfirm回滚，记录被回滚的交易，然后walk结束后恢复被回滚的合法未确认交易
	undoDone, undoList, err := t.RollBackUnconfirmedTx()
	if err != nil {
		t.log.Warn("walk fail,rollback unconfirm tx fail", "err", err)
		return fmt.Errorf("walk rollback unconfirm tx fail")
	}
	xTimer.Mark("walk_rollback_unconfirm_tx")

	// 清理cache
	t.clearBalanceCache()

	// 寻找blockid和latestBlockid的最低公共祖先, 生成undoBlocks和todoBlocks
	undoBlocks, todoBlocks, err := t.sctx.Ledger.FindUndoAndTodoBlocks(t.latestBlockid, blockid)
	if err != nil {
		t.log.Warn("walk fail,find common parent block fail", "dest_block", hex.EncodeToString(blockid),
			"latest_block", hex.EncodeToString(t.latestBlockid), "err", err)
		return fmt.Errorf("walk find common parent block fail")
	}
	xTimer.Mark("walk_find_undo_todo_block")

	// utxoVM回滚需要回滚区块
	err = t.procUndoBlkForWalk(undoBlocks, undoDone, ledgerPrune)
	if err != nil {
		t.log.Warn("walk fail,because undo block fail", "err", err)
		return fmt.Errorf("walk undo block fail")
	}
	xTimer.Mark("walk_undo_block")

	// utxoVM执行需要执行区块
	err = t.procTodoBlkForWalk(todoBlocks)
	if err != nil {
		t.log.Warn("walk fail,because todo block fail", "err", err)
		return fmt.Errorf("walk todo block fail")
	}
	xTimer.Mark("walk_todo_block")

	// 异步回放被回滚未确认交易
	go t.recoverUnconfirmedTx(undoList)

	t.log.Info("utxo walk finish", "dest_block", hex.EncodeToString(blockid),
		"latest_blockid", hex.EncodeToString(t.latestBlockid), "costs", xTimer.Print())
	return nil
}

// 查询交易
func (t *State) QueryTx(txid []byte) (*pb.Transaction, bool, error) {
	return t.xmodel.QueryTx(txid)
}

// 查询账余额
func (t *State) GetBalance(addr string) (*big.Int, error) {
	return t.utxo.GetBalance(addr)
}

// GetTotal 返回当前总资产
func (t *State) GetTotal() *big.Int {
	return t.utxo.GetTotal()
}

// 查找状态机meta信息
func (t *State) GetMeta() *pb.UtxoMeta {
	meta := &pb.UtxoMeta{}
	meta.LatestBlockid = t.latestBlockid
	meta.UtxoTotal = t.utxo.GetTotal().String() // pb没有bigint，所以转换为字符串
	meta.AvgDelay = t.tx.AvgDelay
	meta.UnconfirmTxAmount = t.tx.UnconfirmTxAmount
	meta.MaxBlockSize = t.meta.GetMaxBlockSize()
	meta.ReservedContracts = t.meta.GetReservedContracts()
	meta.ForbiddenContract = t.meta.GetForbiddenContract()
	meta.NewAccountResourceAmount = t.meta.GetNewAccountResourceAmount()
	meta.IrreversibleBlockHeight = t.meta.GetIrreversibleBlockHeight()
	meta.IrreversibleSlideWindow = t.meta.GetIrreversibleSlideWindow()
	meta.GasPrice = t.meta.GetGasPrice()
	meta.GroupChainContract = t.meta.GetGroupChainContract()
	return meta
}

func (t *State) doTxSync(tx *pb.Transaction) error {
	pbTxBuf, pbErr := proto.Marshal(tx)
	if pbErr != nil {
		t.log.Warn("    fail to marshal tx", "pbErr", pbErr)
		return pbErr
	}
	recvTime := time.Now().Unix()
	t.utxo.Mutex.RLock()
	defer t.utxo.Mutex.RUnlock() //lock guard
	spLockKeys := t.utxo.SpLock.ExtractLockKeys(tx)
	succLockKeys, lockOK := t.utxo.SpLock.TryLock(spLockKeys)
	defer t.utxo.SpLock.Unlock(succLockKeys)
	if !lockOK {
		t.log.Info("failed to lock", "txid", utils.F(tx.Txid))
		return ErrDoubleSpent
	}
	waitTime := time.Now().Unix() - recvTime
	if waitTime > TxWaitTimeout {
		t.log.Warn("dotx wait too long!", "waitTime", waitTime, "txid", utils.F(tx.Txid))
	}
	_, exist := t.tx.UnconfirmTxInMem.Load(string(tx.Txid))
	if exist {
		t.log.Debug("this tx already in unconfirm table, when DoTx", "txid", utils.F(tx.Txid))
		return ErrAlreadyInUnconfirmed
	}
	batch := t.ldb.NewBatch()
	cacheFiller := &utxo.CacheFiller{}
	doErr := t.doTxInternal(tx, batch, cacheFiller)
	if doErr != nil {
		t.log.Info("doTxInternal failed, when DoTx", "doErr", doErr)
		return doErr
	}
	batch.Put(append([]byte(pb.UnconfirmedTablePrefix), tx.Txid...), pbTxBuf)
	t.log.Debug("print tx size when DoTx", "tx_size", batch.ValueSize(), "txid", utils.F(tx.Txid))
	writeErr := batch.Write()
	if writeErr != nil {
		t.ClearCache()
		t.log.Warn("fail to save to ldb", "writeErr", writeErr)
		return writeErr
	}
	t.tx.UnconfirmTxInMem.Store(string(tx.Txid), tx)
	cacheFiller.Commit()
	return nil
}

func (t *State) doTxInternal(tx *pb.Transaction, batch kvdb.Batch, cacheFiller *utxo.CacheFiller) error {
	if tx.GetModifyBlock() == nil || (tx.GetModifyBlock() != nil && !tx.ModifyBlock.Marked) {
		if err := t.utxo.CheckInputEqualOutput(tx); err != nil {
			return err
		}
	}

	err := t.xmodel.DoTx(tx, batch)
	if err != nil {
		t.log.Warn("xmodel DoTx failed", "err", err)
		return ErrRWSetInvalid
	}
	for _, txInput := range tx.TxInputs {
		addr := txInput.FromAddr
		txid := txInput.RefTxid
		offset := txInput.RefOffset
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, txid, offset)
		batch.Delete([]byte(utxoKey)) // 删除用掉的utxo
		t.utxo.UtxoCache.Remove(string(addr), utxoKey)
		t.utxo.SubBalance(addr, big.NewInt(0).SetBytes(txInput.Amount))
	}
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		uItem := &utxo.UtxoItem{}
		uItem.Amount = big.NewInt(0)
		uItem.Amount.SetBytes(txOutput.Amount)
		// 输出是0,忽略
		if uItem.Amount.Cmp(big.NewInt(0)) == 0 {
			continue
		}
		uItem.FrozenHeight = txOutput.FrozenHeight
		uItemBinary, uErr := uItem.Dumps()
		if uErr != nil {
			return uErr
		}
		batch.Put([]byte(utxoKey), uItemBinary) // 插入本交易产生的utxo
		if cacheFiller != nil {
			cacheFiller.Add(func() {
				t.utxo.UtxoCache.Insert(string(addr), utxoKey, uItem)
			})
		} else {
			t.utxo.UtxoCache.Insert(string(addr), utxoKey, uItem)
		}
		t.utxo.AddBalance(addr, uItem.Amount)
		if tx.Coinbase {
			// coinbase交易（包括创始块和挖矿奖励)会增加系统的总资产
			t.utxo.UpdateUtxoTotal(uItem.Amount, batch, true)
		}
	}
	return nil
}

func (t *State) NewBatch() kvdb.Batch {
	return t.ldb.NewBatch()
}

func (t *State) GetLDB() kvdb.Database {
	return t.ldb
}

func (t *State) ClearCache() {
	t.utxo.UtxoCache = utxo.NewUtxoCache(t.utxo.CacheSize)
	t.utxo.PrevFoundKeyCache = cache.NewLRUCache(t.utxo.CacheSize)
	t.clearBalanceCache()
	t.xmodel.CleanCache()
	t.log.Info("clear utxo cache")
}

func (t *State) clearBalanceCache() {
	t.log.Info("clear balance cache")
	t.utxo.BalanceCache = cache.NewLRUCache(t.utxo.CacheSize) //清空balanceCache
	t.utxo.BalanceViewDirty = map[string]int{}                //清空cache dirty flag表
	t.xmodel.CleanCache()
}

func (t *State) undoUnconfirmedTx(tx *pb.Transaction, txMap map[string]*pb.Transaction, txGraph tx.TxGraph,
	batch kvdb.Batch, undoDone map[string]bool, pundoList *[]*pb.Transaction) error {
	if undoDone[string(tx.Txid)] == true {
		return nil
	}

	t.log.Info("start to undo transaction", "txid", fmt.Sprintf("%s", hex.EncodeToString(tx.Txid)))
	childrenTxids, exist := txGraph[string(tx.Txid)]
	if exist {
		for _, childTxid := range childrenTxids {
			childTx := txMap[childTxid]
			// 先递归回滚依赖“我”的交易
			t.undoUnconfirmedTx(childTx, txMap, txGraph, batch, undoDone, pundoList)
		}
	}

	// 下面开始回滚自身
	undoErr := t.undoTxInternal(tx, batch)
	if undoErr != nil {
		return undoErr
	}
	batch.Delete(append([]byte(pb.UnconfirmedTablePrefix), tx.Txid...))

	// 记录回滚交易，用于重放
	undoDone[string(tx.Txid)] = true
	if pundoList != nil {
		// 需要保持回滚顺序
		*pundoList = append(*pundoList, tx)
	}
	return nil
}

// undoTxInternal 交易回滚的核心逻辑
// @tx: 要执行的transaction
// @batch: 对数据的变更写入到batch对象
// @tx_in_block:  true说明这个tx是来自区块, false说明是回滚unconfirm表的交易
func (t *State) undoTxInternal(tx *pb.Transaction, batch kvdb.Batch) error {
	err := t.xmodel.UndoTx(tx, batch)
	if err != nil {
		t.log.Warn("xmodel.UndoTx failed", "err", err)
		return ErrRWSetInvalid
	}

	for _, txInput := range tx.TxInputs {
		addr := txInput.FromAddr
		txid := txInput.RefTxid
		offset := txInput.RefOffset
		amount := txInput.Amount
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, txid, offset)
		uItem := &utxo.UtxoItem{}
		uItem.Amount = big.NewInt(0)
		uItem.Amount.SetBytes(amount)
		uItem.FrozenHeight = txInput.FrozenHeight
		t.utxo.UtxoCache.Insert(string(addr), utxoKey, uItem)
		uBinary, uErr := uItem.Dumps()
		if uErr != nil {
			return uErr
		}
		// 退还用掉的UTXO
		batch.Put([]byte(utxoKey), uBinary)
		t.utxo.UnlockKey([]byte(utxoKey))
		t.utxo.AddBalance(addr, uItem.Amount)
		t.log.Trace("undo insert utxo key", "utxoKey", utxoKey)
	}

	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		txOutputAmount := big.NewInt(0).SetBytes(txOutput.Amount)
		if txOutputAmount.Cmp(big.NewInt(0)) == 0 {
			continue
		}
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		// 删除产生的UTXO
		batch.Delete([]byte(utxoKey))
		t.utxo.UtxoCache.Remove(string(addr), utxoKey)
		t.utxo.SubBalance(addr, txOutputAmount)
		t.log.Trace("undo delete utxo key", "utxoKey", utxoKey)
		if tx.Coinbase {
			// coinbase交易（包括创始块和挖矿奖励), 回滚会导致系统总资产缩水
			delta := big.NewInt(0)
			delta.SetBytes(txOutput.Amount)
			t.utxo.UpdateUtxoTotal(delta, batch, false)
		}
	}

	return nil
}

func (t *State) procUndoBlkForWalk(undoBlocks []*pb.InternalBlock,
	undoDone map[string]bool, ledgerPrune bool) (err error) {
	var undoBlk *pb.InternalBlock
	var showBlkId string
	var tx *pb.Transaction
	var showTxId string

	// 依次回滚每个区块
	for _, undoBlk = range undoBlocks {
		showBlkId = hex.EncodeToString(undoBlk.Blockid)
		t.log.Info("start undo block for walk", "blockid", showBlkId)

		// 加一个(共识)开关来判定是否需要采用不可逆
		// 不需要更新IrreversibleBlockHeight以及SlideWindow，因为共识层面的回滚不会回滚到
		// IrreversibleBlockHeight，只有账本裁剪才需要更新IrreversibleBlockHeight以及SlideWindow
		curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
		if !ledgerPrune && undoBlk.Height <= curIrreversibleBlockHeight {
			return fmt.Errorf("block to be undo is older than irreversibleBlockHeight."+
				"irreversible_height:%d,undo_block_height:%d", curIrreversibleBlockHeight, undoBlk.Height)
		}

		// 将batch赋值到合约机的上下文
		batch := t.ldb.NewBatch()
		// 倒序回滚交易
		for i := len(undoBlk.Transactions) - 1; i >= 0; i-- {
			tx = undoBlk.Transactions[i]
			showTxId = hex.EncodeToString(tx.Txid)

			// 回滚交易
			if !undoDone[string(tx.Txid)] {
				err = t.undoTxInternal(tx, batch)
				if err != nil {
					return fmt.Errorf("undo tx fail.txid:%s,err:%v", showTxId, err)
				}
			}

			// 回滚小费，undoTxInternal不会滚小费
			err = t.undoPayFee(tx, batch, undoBlk)
			if err != nil {
				return fmt.Errorf("undo fee fail.txid:%s,err:%v", showTxId, err)
			}
		}

		// 账本裁剪时，无视区块不可逆原则
		if ledgerPrune {
			curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
			curIrreversibleSlideWindow := t.meta.GetIrreversibleSlideWindow()
			err = t.meta.UpdateNextIrreversibleBlockHeightForPrune(undoBlk.Height,
				curIrreversibleBlockHeight, curIrreversibleSlideWindow, batch)
			if err != nil {
				return fmt.Errorf("update irreversible block height fail.err:%v", err)
			}
		}

		// 更新utxoVM LatestBlockid，这里是回滚，所以是更新为上一个区块
		err = t.updateLatestBlockid(undoBlk.PreHash, batch, "error occurs when undo blocks")
		if err != nil {
			return fmt.Errorf("update latest blockid fail.latest_blockid:%s,err:%v",
				hex.EncodeToString(undoBlk.PreHash), err)
		}

		// 每回滚完一个块，内存级别更新UtxoMeta信息
		t.meta.MutexMeta.Lock()
		newMeta := proto.Clone(t.meta.MetaTmp).(*pb.UtxoMeta)
		t.meta.Meta = newMeta
		t.meta.MutexMeta.Unlock()

		t.log.Info("finish undo this block", "blockid", showBlkId)
	}

	return nil
}

func (t *State) updateLatestBlockid(newBlockid []byte, batch kvdb.Batch, reason string) error {
	// FIXME: 如果在高频的更新场景中可能有性能问题，需要账本加上cache
	blk, err := t.sctx.Ledger.QueryBlockHeader(newBlockid)
	if err != nil {
		return err
	}
	batch.Put(append([]byte(pb.MetaTablePrefix), []byte(utxo.LatestBlockKey)...), newBlockid)
	writeErr := batch.Write()
	if writeErr != nil {
		t.ClearCache()
		t.log.Warn(reason, "writeErr", writeErr)
		return writeErr
	}
	t.latestBlockid = newBlockid
	t.heightNotifier.UpdateHeight(blk.GetHeight())
	return nil
}

func (t *State) undoPayFee(tx *pb.Transaction, batch kvdb.Batch, block *pb.InternalBlock) error {
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if !bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		addr = block.Proposer
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		// 删除产生的UTXO
		batch.Delete([]byte(utxoKey))
		t.utxo.UtxoCache.Remove(string(addr), utxoKey)
		t.utxo.SubBalance(addr, big.NewInt(0).SetBytes(txOutput.Amount))
		t.log.Info("undo delete fee utxo key", "utxoKey", utxoKey)
	}
	return nil
}

//批量执行区块
func (t *State) procTodoBlkForWalk(todoBlocks []*pb.InternalBlock) (err error) {
	var todoBlk *pb.InternalBlock
	var showBlkId string
	var tx *pb.Transaction
	var showTxId string

	// 依次执行每个块的交易
	for i := len(todoBlocks) - 1; i >= 0; i-- {
		todoBlk = todoBlocks[i]
		showBlkId = hex.EncodeToString(todoBlk.Blockid)

		t.log.Info("start do block for walk", "blockid", showBlkId)
		// 将batch赋值到合约机的上下文
		batch := t.ldb.NewBatch()

		// 执行区块里面的交易
		idx, length := 0, len(todoBlk.Transactions)
		for idx < length {
			tx = todoBlk.Transactions[idx]
			showTxId = hex.EncodeToString(tx.Txid)
			t.log.Debug("procTodoBlkForWalk", "txid", showTxId, "autogen", t.verifyAutogenTxValid(tx), "coinbase", tx.Coinbase)
			// 校验定时交易合法性
			if t.verifyAutogenTxValid(tx) && !tx.Coinbase {
				// 校验auto tx
				if ok, err := t.ImmediateVerifyAutoTx(todoBlk.Height, tx, false); !ok {
					return fmt.Errorf("immediate verify auto tx error.txid:%s,err:%v", showTxId, err)
				}
			}

			// 校验普通交易合法性
			if !tx.Autogen && !tx.Coinbase {
				if ok, err := t.ImmediateVerifyTx(tx, false); !ok {
					return fmt.Errorf("immediate verify tx error.txid:%s,err:%v", showTxId, err)
				}
			}

			// 执行交易
			cacheFiller := &utxo.CacheFiller{}
			err = t.doTxInternal(tx, batch, cacheFiller)
			if err != nil {
				return fmt.Errorf("todo tx fail.txid:%s,err:%v", showTxId, err)
			}
			cacheFiller.Commit()

			// 处理小费
			err = t.payFee(tx, batch, todoBlk)
			if err != nil {
				return fmt.Errorf("pay fee fail.txid:%s,err:%v", showTxId, err)
			}
			idx++
		}

		t.log.Debug("Begin to Finalize", "blockid", showBlkId)

		// 更新不可逆区块高度
		curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
		curIrreversibleSlideWindow := t.meta.GetIrreversibleSlideWindow()
		err = t.meta.UpdateNextIrreversibleBlockHeight(todoBlk.Height, curIrreversibleBlockHeight,
			curIrreversibleSlideWindow, batch)
		if err != nil {
			return fmt.Errorf("update irreversible height fail.blockid:%s,err:%v", showBlkId, err)
		}
		// 每do一个block,是一个原子batch写
		err = t.updateLatestBlockid(todoBlk.Blockid, batch, "error occurs when do blocks")
		if err != nil {
			return fmt.Errorf("update last blockid fail.blockid:%s,err:%v", showBlkId, err)
		}

		// 完成一个区块后，内存级别更新UtxoMeta信息
		t.meta.MutexMeta.Lock()
		newMeta := proto.Clone(t.meta.MetaTmp).(*pb.UtxoMeta)
		t.meta.Meta = newMeta
		t.meta.MutexMeta.Unlock()

		t.log.Info("finish todo this block", "blockid", showBlkId)
	}

	return nil
}

func (t *State) payFee(tx *pb.Transaction, batch kvdb.Batch, block *pb.InternalBlock) error {
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if !bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		addr = block.Proposer // 占位符替换为矿工
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		uItem := &utxo.UtxoItem{}
		uItem.Amount = big.NewInt(0)
		uItem.Amount.SetBytes(txOutput.Amount)
		uItemBinary, uErr := uItem.Dumps()
		if uErr != nil {
			return uErr
		}
		batch.Put([]byte(utxoKey), uItemBinary) // 插入本交易产生的utxo
		t.utxo.AddBalance(addr, uItem.Amount)
		t.utxo.UtxoCache.Insert(string(addr), utxoKey, uItem)
		t.log.Trace("    insert fee utxo key", "utxoKey", utxoKey, "amount", uItem.Amount.String())
	}
	return nil
}

func (t *State) recoverUnconfirmedTx(undoList []*pb.Transaction) {
	xTimer := timer.NewXTimer()
	t.log.Info("start recover unconfirm tx", "tx_count", len(undoList))

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
		isConfirm, err := t.sctx.Ledger.HasTransaction(tx.Txid)
		if err != nil && isConfirm {
			confirmCnt++
			t.log.Info("this tx has been confirmed,ignore recover", "txid", hex.EncodeToString(tx.Txid))
			continue
		}

		t.log.Info("start recover unconfirm tx", "txid", hex.EncodeToString(tx.Txid))
		// 重新对交易鉴权，过掉冲突交易
		isValid, err := t.ImmediateVerifyTx(tx, false)
		if err != nil || !isValid {
			verifyErrCnt++
			t.log.Info("this tx immediate verify fail,ignore recover", "txid",
				hex.EncodeToString(tx.Txid), "is_valid", isValid, "err", err)
			continue
		}

		// 重新提交交易，可能交易已经被其他节点打包到区块广播过来，导致失败
		err = t.doTxSync(tx)
		if err != nil {
			doTxErrCnt++
			t.log.Info("dotx fail for recover unconfirm tx,ignore recover this tx",
				"txid", hex.EncodeToString(tx.Txid), "err", err)
			continue
		}

		succCnt++
		t.log.Info("recover unconfirm tx succ", "txid", hex.EncodeToString(tx.Txid))
	}

	t.log.Info("recover unconfirm tx done", "costs", xTimer.Print(), "tx_count", len(undoList),
		"succ_count", succCnt, "confirm_count", confirmCnt, "verify_err_count",
		verifyErrCnt, "dotx_err_cnt", doTxErrCnt)
}

//执行一个block的时候, 处理本地未确认交易
//返回：被确认的txid集合、err
func (t *State) processUnconfirmTxs(block *pb.InternalBlock, batch kvdb.Batch, needRepost bool) (map[string]bool, map[string]bool, error) {
	if !bytes.Equal(block.PreHash, t.latestBlockid) {
		t.log.Warn("play failed", "block.PreHash", utils.F(block.PreHash),
			"latestBlockid", utils.F(t.latestBlockid))
		return nil, nil, ErrPreBlockMissMatch
	}
	txidsInBlock := map[string]bool{}    // block里面所有的txid
	UTXOKeysInBlock := map[string]bool{} // block里面所有的交易需要用掉的utxo
	keysVersionInBlock := map[string]string{}
	for _, tx := range block.Transactions {
		txidsInBlock[string(tx.Txid)] = true
		for _, txInput := range tx.TxInputs {
			utxoKey := utxo.GenUtxoKey(txInput.FromAddr, txInput.RefTxid, txInput.RefOffset)
			if UTXOKeysInBlock[utxoKey] { //检查块内的utxo双花情况
				t.log.Warn("found duplicated utxo in same block", "utxoKey", utxoKey, "txid", utils.F(tx.Txid))
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

	// 下面开始处理unconfirmed的交易
	unconfirmTxMap, unconfirmTxGraph, delayedTxMap, loadErr := t.tx.SortUnconfirmedTx()
	if loadErr != nil {
		return nil, nil, loadErr
	}
	t.log.Info("unconfirm table size", "unconfirmTxCount", t.tx.UnconfirmTxAmount)
	undoDone := map[string]bool{}
	unconfirmToConfirm := map[string]bool{}
	for txid, unconfirmTx := range unconfirmTxMap {
		if _, exist := txidsInBlock[string(txid)]; exist {
			// 说明这个交易已经被确认
			batch.Delete(append([]byte(pb.UnconfirmedTablePrefix), []byte(txid)...))
			t.log.Trace("  delete from unconfirmed", "txid", fmt.Sprintf("%x", txid))
			// 直接从unconfirm表删除, 大部分情况是这样的
			unconfirmToConfirm[txid] = true
			continue
		}
		hasConflict := false
		for _, unconfirmTxInput := range unconfirmTx.TxInputs {
			addr := unconfirmTxInput.FromAddr
			txid := unconfirmTxInput.RefTxid
			offset := unconfirmTxInput.RefOffset
			utxoKey := utxo.GenUtxoKey(addr, txid, offset)
			if _, exist := UTXOKeysInBlock[utxoKey]; exist {
				// 说明此交易和block里面的交易存在双花冲突，需要回滚, 少数情况
				t.log.Warn("conflict, refuse double spent", "key", utxoKey, "txid", utils.F(unconfirmTx.Txid))
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
				t.log.Warn("inputs version conflict", "key", bucketAndKey, "localVersion", localVersion, "remoteVersion", remoteVersion)
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
				t.log.Warn("outputs version conflict", "key", bucketAndKey, "localVersion", localVersion, "remoteVersion", remoteVersion)
				hasConflict = true
				break
			}
		}
		tooDelayed := delayedTxMap[string(unconfirmTx.Txid)]
		if tooDelayed {
			t.log.Warn("will undo tx because it is beyond confirmed delay", "txid", utils.F(unconfirmTx.Txid))
		}
		if hasConflict || tooDelayed {
			undoErr := t.undoUnconfirmedTx(unconfirmTx, unconfirmTxMap,
				unconfirmTxGraph, batch, undoDone, nil)
			if undoErr != nil {
				t.log.Warn("fail to undo tx", "undoErr", undoErr)
				return nil, nil, undoErr
			}
		}
	}
	if needRepost {
		go func() {
			sortTxList, unexpectedCyclic, dagSizeList := tx.TopSortDFS(unconfirmTxGraph)
			if unexpectedCyclic {
				t.log.Warn("transaction conflicted", "unexpectedCyclic", unexpectedCyclic)
				return
			}
			dagNo := 0
			t.log.Info("parallel group of reposting", "dagGroupEach", dagSizeList)
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
				t.utxo.OfflineTxChan <- batchTx
				start += dagsize
				dagNo++
			}
		}()
	}
	return unconfirmToConfirm, undoDone, nil
}

func (t *State) Close() {
	t.ldb.Close()
}

func (t *State) queryContractBannedStatus(contractName string) (bool, error) {
	request := &protos.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: "unified_check",
		MethodName:   "banned_check",
		Args: map[string][]byte{
			"contract": []byte(contractName),
		},
	}

	xmReader := t.CreateXMReader()
	sandBoxCfg := &contract.SandboxConfig{
		XMReader: xmReader,
	}
	sandBox, err := t.sctx.ContractMgr.NewStateSandbox(sandBoxCfg)
	if err != nil {
		return false, err
	}

	contextConfig := &contract.ContextConfig{
		State:          sandBox,
		ResourceLimits: contract.MaxLimits,
		ContractName:   request.GetContractName(),
	}
	ctx, err := t.sctx.ContractMgr.NewContext(contextConfig)
	if err != nil {
		t.log.Warn("queryContractBannedStatus new context error", "error", err)
		return false, err
	}
	_, err = ctx.Invoke(request.GetMethodName(), request.GetArgs())
	if err != nil && err.Error() == "contract has been banned" {
		ctx.Release()
		t.log.Warn("queryContractBannedStatus error", "error", err)
		return true, err
	}
	ctx.Release()
	return false, nil
}

// WaitBlockHeight wait util the height of current block >= target
func (t *State) WaitBlockHeight(target int64) int64 {
	return t.heightNotifier.WaitHeight(target)
}
