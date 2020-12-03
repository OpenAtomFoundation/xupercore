// 统一定义状态机对外暴露功能
package state

import (
	"bytes"
	"fmt"
	"math/big"
	"path/filepath"
	"time"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/pb"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/meta"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/xmodel"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

type XuperState struct {
	lctx *def.LedgerCtx
	log  logs.Logger
	ledger ledger.Ledger
	utxo utxo.Utxo //utxo表
	xmodel xmodel.XModel //xmodel数据表和历史表
	meta  meta.Meta //meta表
	tx  tx.Tx //未确认交易表
	ldb kvdb.Database
	latestBlockid []byte
}

func NewXuperState(lctx *def.LedgerCtx) (*XuperState, error) {
	if lctx == nil {
		return nil, fmt.Errrof("create state failed because context set error")
	}

	obj := &XuperState{
		lctx: lctx,
		log:  lctx.XLog,
	}

	var err error
	kvParam := &kvdb.KVParameter{
		//todo 暂定数据放state目录
		DBPath:                filepath.Join(lctx.LedgerCfg.StorePath, "state"),
		KVEngineType:          lctx.LedgerCfg.KVEngineType,
		MemCacheSize:          ledger.MemCacheSize,
		FileHandlersCacheSize: ledger.FileHandlersCacheSize,
		OtherPaths:            lctx.LedgerCfg.OtherPaths,
		StorageType:           lctx.LedgerCfg.StorageType,
	}
	obj.ldb, err = kvdb.CreateKVInstance(kvParam)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create ldb error:%s", err)
	}

	obj.ledger, err = ledger.NewLedger(lctx)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create ledger error:%s", err)
	}

	obj.utxo, err = utxo.NewUtxo(lctx, obj.ledger)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create utxo error:%s", err)
	}

	obj.xmodel, err = xmodel.NewXModel(obj.ledger, obj.ldb, lctx.XLog)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create xmodel error:%s", err)
	}

	obj.meta, err = meta.NewMeta(lctx, obj.ldb)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create meta error:%s", err)
	}

	obj.tx, err = meta.NewTx(lctx, obj.ldb)
	if err != nil {
		return nil, fmt.Errorf("create state failed because create tx error:%s", err)
	}

	return obj, nil
}

// 选择足够金额的utxo
func (t *XuperState) SelectUtxos(fromAddr string, totalNeed *big.Int, needLock, excludeUnconfirmed bool) ([]*pb.TxInput, [][]byte, *big.Int, error) {
	return t.utxo.SelectUtxos(fromAddr, totalNeed, needLock, excludeUnconfirmed)
}

// 获取一批未确认交易（用于矿工打包区块）
func (t *XuperState) GetUnconfirmedTx(dedup bool) ([]*pb.Transaction, error) {
	return t.tx.GetUnconfirmedTx(dedup)
}

func (t *XuperState) GetLatestBlockid() []byte {
	return t.latestBlockid
}

// HasTx 查询一笔交易是否在unconfirm表  这些可能是放在tx对外提供
func (t *XuperState) HasTx(txid []byte) (bool, error) {
	_, exist := t.tx.UnconfirmedTxInMem.Load(string(txid))
	return exist, nil
}

func (t *XuperState) QueryTxFromForbiddenWithConfirmed(txid []byte) (bool, bool, error) {
	// 如果配置文件配置了封禁tx监管合约，那么继续下面的执行。否则，直接返回.
	forbiddenContract := t.meta.GetForbiddenContract()
	if forbiddenContract == nil {
		return false, false, nil
	}

	// 这里不针对ModuleName/ContractName/MethodName做特殊化处理
	moduleNameForForbidden := forbiddenContract.ModuleName
	contractNameForForbidden := forbiddenContract.ContractName
	methodNameForForbidden := forbiddenContract.MethodName

	if moduleNameForForbidden == "" && contractNameForForbidden == "" && methodNameForForbidden == "" {
		return false, false, nil
	}

	request := &pb.InvokeRequest{
		ModuleName:   moduleNameForForbidden,
		ContractName: contractNameForForbidden,
		MethodName:   methodNameForForbidden,
		Args: map[string][]byte{
			"txid": []byte(fmt.Sprintf("%x", txid)),
		},
	}
	/*modelCache, err := xmodel.NewXModelCache(t.xmodel, t.utxo)
	if err != nil {
		return false, false, err
	}
	contextConfig := &contract.ContextConfig{
		XMCache:        modelCache,
		ResourceLimits: contract.MaxLimits,
	}
	moduleName := request.GetModuleName()
	vm, err := t.vmMgr3.GetVM(moduleName)
	if err != nil {
		return false, false, err
	}*/
	contextConfig.ContractName = request.GetContractName()
	ctx, err := vm.NewContext(contextConfig)
	if err != nil {
		return false, false, err
	}
	invokeRes, invokeErr := ctx.Invoke(request.GetMethodName(), request.GetArgs())
	if invokeErr != nil {
		ctx.Release()
		return false, false, invokeErr
	}
	ctx.Release()
	// 判断forbidden合约的结果
	if invokeRes.Status >= 400 {
		return false, false, nil
	}
	inputs, _, _ := modelCache.GetRWSets()
	versionData := &xmodel_pb.VersionedData{}
	if len(inputs) != 2 {
		return false, false, nil
	}
	versionData = inputs[1]
	confirmed, err := t.ledger.HasTransaction(versionData.RefTxid)
	if err != nil {
		return false, false, err
	}
	return true, confirmed, nil
}

func (t *XuperState) GetFrozenBalance(addr string) (*big.Int, error) {
	addrPrefix := fmt.Sprintf("%s%s_", pb.UTXOTablePrefix, addr)
	utxoFrozen := big.NewInt(0)
	curHeight := t.ledger.GetMeta().TrunkHeight
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
func (t *XuperState) GetBalanceDetail(addr string) ([]*pb.TokenFrozenDetail, error) {
	addrPrefix := fmt.Sprintf("%s%s_", pb.UTXOTablePrefix, addr)
	utxoFrozen := big.NewInt(0)
	utxoUnFrozen := big.NewInt(0)
	curHeight := t.ledger.GetMeta().TrunkHeight
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

	var tokenFrozenDetails []*pb.TokenFrozenDetail

	tokenFrozenDetail := &pb.TokenFrozenDetail{
		Balance:  utxoFrozen.String(),
		IsFrozen: true,
	}
	tokenFrozenDetails = append(tokenFrozenDetails, tokenFrozenDetail)

	tokenUnFrozenDetail := &pb.TokenFrozenDetail{
		Balance:  utxoUnFrozen.String(),
		IsFrozen: false,
	}
	tokenFrozenDetails = append(tokenFrozenDetails, tokenUnFrozenDetail)

	return tokenFrozenDetails, nil
}

// 校验交易
func (t *XuperState) VerifyTx() {

}

// 执行交易
func (t *XuperState) DoTx(tx *pb.Transaction) error {
	tx.ReceivedTimestamp = time.Now().UnixNano()
	if tx.Coinbase {
		t.log.Warn("coinbase tx can not be given by PostTx", "txid", fmt.Sprintf("%x", tx.Txid))
		return ErrUnexpected
	}
	if len(tx.Blockid) > 0 {
		t.log.Warn("tx from PostTx must not have blockid", "txid",  fmt.Sprintf("%x", tx.Txid))
		return ErrUnexpected
	}
	return t.doTxSync(tx)
}

// 创建快照
func (t *XuperState) GetSnapShotWithBlock(blockId []byte) (xmodel.XMReader, error) {
	reader, err := t.xmodel.CreateSnapshot(blockId)
	return reader, err
}

func (t *XuperState) BucketCacheDelete(bucket, version string) {
	t.xmodel.BucketCacheDelete(bucket, version)
}

// 执行区块
func (t *XuperState) Play(blockid []byte) error {
	return t.PlayAndRepost(blockid, false, true)
}

func (t *XuperState) PlayForMiner(blockid []byte) error {
	batch := t.NewBatch()
	return t.utxo.PlayForMiner(blockid, batch)
}

// 执行和发送区块
func (t *XuperState) PlayAndRepost(blockid []byte, needRepost bool, isRootTx bool) error {
	return t.utxo.PlayAndRepost(blockid, needRepost, isRootTx)
}

// 回滚全部未确认交易
func (t *XuperState) RollBackUnconfirmedTx() (map[string]bool, []*pb.Transaction, error) {
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

	// 回滚完成从未确认交易表删除
	for txid := range undoDone {
		t.tx.UnconfirmTxInMem.Delete(txid)
	}
	return undoDone, undoList, nil
}

func (t *XuperState) PreExec(req *pb.InvokeRPCRequest) (*pb.InvokeResponse, error) {
	// get reserved contracts from chain config
	reservedRequests, err := t.meta.GetReservedContractRequests(req.GetRequests(), true)
	if err != nil {
		t.log.Error("PreExec getReservedContractRequests error", "error", err)
		return nil, err
	}
	// contract request with reservedRequests
	req.Requests = append(reservedRequests, req.Requests...)
	t.log.Trace("PreExec requests after merge", "requests", req.Requests)

	// if no reserved request and user's request, return directly
	// the operation of xmodel.NewXModelCache costs some resources
	if len(req.Requests) == 0 {
		rsps := &pb.InvokeResponse{}
		return rsps, nil
	}

	// transfer in contract
	transContractName, transAmount, err := t.tx.ParseContractTransferRequest(req.Requests)
	if err != nil {
		return nil, err
	}
	// init modelCache
	modelCache, err := xmodel.NewXModelCache(t.xmodel)
	if err != nil {
		return nil, err
	}

	contextConfig := &contract.ContextConfig{
		//todo 由谁实现XMState  哪些字段是需要初始化的?
		//XMState:     modelCache,
		Initiator:   req.GetInitiator(),
		AuthRequire: req.GetAuthRequire(),
	}
	gasUesdTotal := int64(0)
	response := [][]byte{}

	gasPrice := t.meta.GetGasPrice()

	var requests []*pb.InvokeRequest
	var responses []*pb.ContractResponse
	// af is the flag of whether the contract already carries the amount parameter
	af := false
	for i, tmpReq := range req.Requests {
		if af {
			return nil, ErrInvokeReqParams
		}
		if tmpReq == nil {
			continue
		}
		if tmpReq.GetModuleName() == "" && tmpReq.GetContractName() == "" && tmpReq.GetMethodName() == "" {
			continue
		}

		if tmpReq.GetAmount() != "" {
			amount, ok := new(big.Int).SetString(tmpReq.GetAmount(), 10)
			if !ok {
				return nil, def.ErrInvokeReqParams
			}
			if amount.Cmp(new(big.Int).SetInt64(0)) == 1 {
				af = true
			}
		}
		moduleName := tmpReq.GetModuleName()
		vm, err := uv.vmMgr3.GetVM(moduleName)
		if err != nil {
			return nil, err
		}

		contextConfig.ContractName = tmpReq.GetContractName()
		if transContractName == tmpReq.GetContractName() {
			contextConfig.TransferAmount = transAmount.String()
		} else {
			contextConfig.TransferAmount = ""
		}
		//TODO vm从何而来？
		ctx, err := vm.NewContext(contextConfig)
		if err != nil {
			t.log.Error("PreExec NewContext error", "error", err,
				"contractName", tmpReq.GetContractName())
			if i < len(reservedRequests) && strings.HasSuffix(err.Error(), "not found") {
				requests = append(requests, tmpReq)
				continue
			}
			return nil, err
		}
		res, err := ctx.Invoke(tmpReq.GetMethodName(), tmpReq.GetArgs())
		if err != nil {
			ctx.Release()
			t.log.Error("PreExec Invoke error", "error", err,
				"contractName", tmpReq.GetContractName())
			return nil, err
		}
		if res.Status >= 400 && i < len(reservedRequests) {
			ctx.Release()
			t.log.Error("PreExec Invoke error", "status", res.Status, "contractName", tmpReq.GetContractName())
			return nil, errors.New(res.Message)
		}
		response = append(response, res.Body)
		responses = append(responses, contract.ToPBContractResponse(res))

		resourceUsed := ctx.ResourceUsed()
		if i >= len(reservedRequests) {
			gasUesdTotal += resourceUsed.TotalGas(gasPrice)
		}
		request := *tmpReq
		request.ResourceLimits = contract.ToPbLimits(resourceUsed)
		requests = append(requests, &request)
		ctx.Release()
	}

	utxoInputs, utxoOutputs := modelCache.GetUtxoRWSets()

	err = modelCache.WriteTransientBucket()
	if err != nil {
		return nil, err
	}

	inputs, outputs, err := modelCache.GetRWSets()
	if err != nil {
		return nil, err
	}
	rsps := &pb.InvokeResponse{
		Inputs:      xmodel.GetTxInputs(inputs),
		Outputs:     xmodel.GetTxOutputs(outputs),
		Response:    response,
		Requests:    requests,
		GasUsed:     gasUesdTotal,
		Responses:   responses,
		UtxoInputs:  utxoInputs,
		UtxoOutputs: utxoOutputs,
	}
	return rsps, nil
}

// 同步账本和状态机
func (t *XuperState) Walk() {

}

// 获取状态机tip block id
func (t *XuperState) GetTipBlockid() []byte {
	return t.GetMeta().GetTipBlockid()
}

// 查询交易
func (t *XuperState) QueryTx(txid []byte) (*pb.Transaction, bool, error) {
	return t.xmodel.QueryTx(txid)
}

// 查询账余额
func (t *XuperState) GetBalance(addr string) (*big.Int, error){
	return t.utxo.GetBalance(addr)
}

// 查找状态机meta信息
func (t *XuperState) GetMeta() *pb.UtxoMeta {
	return t.meta.GetMeta()
}

func (t *XuperState) doTxSync(tx *pb.Transaction) error {
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
		t.log.Info("failed to lock", "txid", global.F(tx.Txid))
		return ErrDoubleSpent
	}
	waitTime := time.Now().Unix() - recvTime
	if waitTime > TxWaitTimeout {
		t.log.Warn("dotx wait too long!", "waitTime", waitTime, "txid", fmt.Sprintf("%x", tx.Txid))
	}
	_, exist := t.tx.UnconfirmTxInMem.Load(string(tx.Txid))
	if exist {
		t.log.Debug("this tx already in unconfirm table, when DoTx", "txid", fmt.Sprintf("%x", tx.Txid))
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
	t.log.Debug("print tx size when DoTx", "tx_size", batch.ValueSize(), "txid", fmt.Sprintf("%x", tx.Txid))
	writeErr := batch.Write()
	if writeErr != nil {
		t.utxo.ClearCache()
		t.log.Warn("fail to save to ldb", "writeErr", writeErr)
		return writeErr
	}
	t.tx.UnconfirmTxInMem.Store(string(tx.Txid), tx)
	cacheFiller.Commit()
	return nil
}

func (t *XuperState) doTxInternal(tx *pb.Transaction, batch kvdb.Batch, cacheFiller *CacheFiller) error {
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
		utxoKey := GenUtxoKeyWithPrefix(addr, txid, offset)
		batch.Delete([]byte(utxoKey)) // 删除用掉的utxo
		t.utxo.UtxoCache.Remove(string(addr), utxoKey)
		t.utxo.SubBalance(addr, big.NewInt(0).SetBytes(txInput.Amount))
	}
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		utxoKey := GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
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

//TODO how to judge
func (t *XuperState) isSmartContract(desc []byte) (*contract.TxDesc, bool) {
	if bytes.HasPrefix(desc, []byte("{")) {
		descObj, err := contract.Parse(string(desc))
		if err != nil {
			t.log.Warn("parse contract failed", "desc", fmt.Sprintf("%s", desc))
			return nil, false
		}
		if descObj.Module == "" || descObj.Method == "" {
			return nil, false
		}
		// 判断合约是不是被注册
		allowedModules := t.smartContract.GetAll()
		if _, ok := allowedModules[descObj.Module]; !ok {
			return nil, false
		}
		return descObj, err == nil
	}
	return nil, false
}

func (t *XuperState) NewBatch() kvdb.Batch {
	return t.ldb.NewBatch()
}

func (t *XuperState) ClearCache() {
	t.utxo.utxoCache = NewUtxoCache(t.cacheSize)
	t.prevFoundKeyCache = common.NewLRUCache(uv.cacheSize)
	t.clearBalanceCache()
	t.xmodel.CleanCache()
	t.log.Warn("clear utxo cache")
}

func (t *XuperState) clearBalanceCache() {
	t.log.Warn("clear balance cache")
	t.balanceCache = cache.NewLRUCache(t.cacheSize) //清空balanceCache
	t.balanceViewDirty = map[string]int{}             //清空cache dirty flag表
	t.xmodel.CleanCache()
}

func (t *XuperState) undoUnconfirmedTx(tx *pb.Transaction, txMap map[string]*pb.Transaction, txGraph tx.TxGraph,
	batch kvdb.Batch, undoDone map[string]bool, pundoList *[]pb.Transaction) error {
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
func (t *XuperState) undoTxInternal(tx *pb.Transaction, batch kvdb.Batch) error {
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
		t.UnlockKey([]byte(utxoKey))
		t.AddBalance(addr, uItem.Amount)
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
		t.UtxoCache.Remove(string(addr), utxoKey)
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


// GetContractStatus get contract status of a contract
func (t *XuperState) GetContractStatus(contractName string) (*pb.ContractStatus, error) {
	res := &pb.ContractStatus{}
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


// queryContractBannedStatus query where the contract is bannded
// FIXME zq: need to use a grace manner to get the bannded contract name
func (t *XuperState) queryContractBannedStatus(contractName string) (bool, error) {
	request := &pb.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: "unified_check",
		MethodName:   "banned_check",
		Args: map[string][]byte{
			"contract": []byte(contractName),
		},
	}

	/*modelCache, err := xmodel.NewXModelCache(t.xmodel, t.utxo)
	if err != nil {
		t.log.Warn("queryContractBannedStatus new model cache error", "error", err)
		return false, err
	}*/
	moduleName := request.GetModuleName()
	vm, err := t.vmMgr3.GetVM(moduleName)
	if err != nil {
		t.log.Warn("queryContractBannedStatus get VM error", "error", err)
		return false, err
	}

	contextConfig := &contract.ContextConfig{
		//todo
		//XMState:        modelCache,
		ResourceLimits: contract.MaxLimits,
		ContractName:   request.GetContractName(),
	}
	ctx, err := vm.NewContext(contextConfig)
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


//todo Where to implement this function
func GenWriteKeyWithPrefix() {}
