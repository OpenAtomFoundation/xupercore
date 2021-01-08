package miner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xuperchain/core/pb"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/agent"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

// 负责生产和同步区块
type Miner struct {
	ctx *common.ChainCtx
	log logs.Logger
	// 矿工锁，用来确保矿工出块和同步操作串行进行
	minerMutex sync.Mutex
	// 记录同步中任务目标区块高度
	inSyncHeight       int64
	inSyncTargetHeight int64
	// 记录同步中任务目标区块id
	inSyncTargetBlockId []byte
	// 标记是否退出运行
	isExit bool
}

func NewMiner(ctx *common.ChainCtx) *Miner {
	obj := &Miner{
		ctx: ctx,
		log: ctx.GetLog(),
	}

	return obj
}

// 处理P2P网络中接收到的区块
func (t *Miner) ProcBlock(ctx xctx.XContext, block *lpb.InternalBlock) error {
	if ctx == nil || block == nil {
		return fmt.Errorf("param error")
	}

	// 1.检查区块有效性和高度，忽略无效或者比当前同步高度低的区块
	blockSize := int64(proto.Size(block))
	maxBlockSize := t.ctx.State.GetMaxBlockSize()
	if blockSize > maxBlockSize {
		ctx.GetLog().Warn("refused proc block because block is too large",
			"blockSize", blockSize, "maxBlockSize", maxBlockSize)
		return fmt.Errorf("refused proc block")
	}

	inSyncTargetHeight := t.inSyncTargetHeight
	inSyncTargetBlockId := t.inSyncTargetBlockId
	if block.GetHeight() < inSyncTargetHeight || bytes.Equal(block.GetBlockid(), inSyncTargetBlockId) {
		ctx.GetLog().Trace("ignore block because recv block height lower than in sync height", "recvHeight",
			block.GetHeight(), "inSyncTargetHeight", inSyncTargetHeight,
			"inSyncTargetBlockId", utils.F(inSyncTargetBlockId))
		return nil
	}

	for id, tx := range block.Transactions {
		if !t.ctx.Ledger.IsValidTx(id, tx, block) {
			ctx.GetLog().Warn("invalid tx got from the block", "txid", utils.F(tx.Txid),
				"blockId", utils.F(block.Blockid))
			return fmt.Errorf("invalid tx got from the block")
		}
	}

	// 尝试同步到该高度，如果小于账本高度会被直接忽略
	return t.trySyncBlock(ctx, block)
}

// 启动矿工，周期检查矿工身份
// 同一时间，矿工状态是唯一的。0:休眠中 1:同步区块中 2:打包区块中
func (t *Miner) Start() {
	// 启动矿工循环
	var err error
	isMiner := false
	isSync := false
	ledgerTipId := t.ctx.Ledger.GetMeta().TipBlockid
	ledgerTipHeight := t.ctx.Ledger.GetMeta().TrunkHeight
	stateTipId := t.ctx.State.GetLatestBlockid()
	for !t.IsExit() {
		log, _ := logs.NewLogger("", "miner")
		ctx := &xctx.BaseCtx{
			XLog:  log,
			Timer: timer.NewXTimer(),
		}

		ctx.GetLog().Trace("miner running", "ledgerTipHeight", ledgerTipHeight, "ledgerTipId",
			utils.F(ledgerTipId), "stateTipId", utils.F(stateTipId), "err", err)

		// 1.状态机walk，确保状态机和账本一致
		if !bytes.Equal(ledgerTipId, stateTipId) {
			err = t.ctx.State.Walk(ledgerTipId, false)
		}
		// 2.通过共识检查矿工身份
		if err == nil {
			isMiner, isSync, err = t.ctx.Consensus.CompeteMaster(ledgerTipHeight + 1)
		}
		// 3.如需要同步，尝试同步网络最新区块
		if err == nil && isSync {
			err = t.trySyncBlock(ctx, nil)
		}
		// 4.如果是矿工，出块
		if err == nil && isMiner {
			err = t.mining(ctx)
		}
		// 5.如果出错，休眠3s后重试
		if err != nil && !t.IsExit() {
			ctx.GetLog().Warn("miner run occurred error,sleep 3s try", "err", err)
			time.Sleep(3 * time.Second)
		}
		// 6.更新状态
		if !t.IsExit() {
			err = nil
			ledgerTipId = t.ctx.Ledger.GetMeta().TipBlockid
			ledgerTipHeight = t.ctx.Ledger.GetMeta().TrunkHeight
			stateTipId = t.ctx.State.GetLatestBlockid()
		}
	}

	t.log.Trace("miner exited", "ledgerTipHeight", ledgerTipHeight)
}

// 停止矿工
func (t *Miner) Stop() {
	t.isExit = true
}

func (t *Miner) IsExit() bool {
	return t.isExit
}

// 挖矿生产区块
func (t *Miner) mining(ctx xctx.XContext) error {
	// 1.获取矿工互斥锁，矿工行为完全串行
	t.minerMutex.Lock()
	defer t.minerMutex.Unlock()

	// 2.状态机walk，确保状态机和账本一致
	ledgerTipId := t.ctx.Ledger.GetMeta().TipBlockid
	stateTipId := t.ctx.State.GetLatestBlockid()
	if !bytes.Equal(ledgerTipId, stateTipId) {
		err := t.ctx.State.Walk(ledgerTipId, false)
		if err != nil {
			ctx.GetLog().Warn("mining walk failed", "ledgerTipId", utils.F(ledgerTipId),
				"stateTipId", utils.F(stateTipId))
			return fmt.Errorf("mining walk failed")
		}
		stateTipId = ledgerTipId
	}

	// 3.共识挖矿前处理
	height := t.ctx.Ledger.GetMeta().TrunkHeight + 1
	now := time.Now()
	isTruncate, extData, err := t.ctx.Consensus.ProcessBeforeMiner(now.UnixNano())
	if err != nil {
		ctx.GetLog().Warn("consensus process before miner failed", "err", err)
		return fmt.Errorf("consensus process before miner failed")
	}
	if isTruncate {
		// 裁剪掉账本最高区块，裁掉的交易判断冲突重新回放，裁剪完后结束本次出块操作
		return t.truncateForMiner(ctx, ledgerTipId)
	}

	// 4.打包区块
	block, err := t.packBlock(ctx, height, now, extData)
	if err != nil {
		ctx.GetLog().Warn("pack block error", "err", err)
		return err
	}

	// 5.账本&状态机&共识确认新区块
	err = t.confirmBlock(ctx, block)
	if err != nil {
		ctx.GetLog().Warn("confirm block error", "err", err)
		return err
	}

	// 6.异步广播新生成的区块
	go t.broadcastBlock(ctx, block)

	ctx.GetLog().Trace("complete new block generation", "blockId", utils.F(block.GetBlockid()),
		"height", height, "costs", ctx.GetTimer().Print())
	return nil
}

func (t *Miner) truncateForMiner(ctx xctx.XContext, ledgerTipId []byte) error {
	block, err := t.ctx.Ledger.QueryBlock(ledgerTipId)
	if err != nil {
		ctx.GetLog().Warn("truncate failed because query tip block error", "err", err)
		return err
	}

	err = t.ctx.State.Walk(block.PreHash, false)
	if err != nil {
		ctx.GetLog().Warn("truncate failed because state walk error", "ledgerTipId", utils.F(ledgerTipId),
			"stateTipId", utils.F(block.PreHash))
		return err
	}

	stateTipId := t.ctx.State.GetTipBlockid()
	err = t.ctx.Ledger.Truncate(stateTipId)
	if err != nil {
		ctx.GetLog().Warn("truncate failed because ledger truncate error", "err", err)
		return err
	}

	return nil
}

func (t *Miner) packBlock(ctx xctx.XContext, height int64, now time.Time, consData []byte) (*lpb.InternalBlock, error) {
	// 区块大小限制
	sizeLimit, err := t.ctx.State.MaxTxSizePerBlock()
	if err != nil {
		return nil, err
	}

	// 1.查询timer异步交易
	timerTxList, err := t.getTimerTx(height)
	for _, tx := range timerTxList {
		sizeLimit -= proto.Size(tx)
	}

	// 2.选择本次要打包的tx
	generalTxList, err := t.getUnconfirmedTx(sizeLimit)

	// 3.获取矿工奖励交易
	awardTx, err := t.getAwardTx(height)

	txList := make([]*lpb.Transaction, 0)
	if len(timerTxList) > 0 {
		txList = append(txList, timerTxList...)
	}
	if len(generalTxList) > 0 {
		txList = append(txList, generalTxList...)
	}

	//3. 统一在最后插入矿工奖励
	txList = append(txList, awardTx)
	block, err := t.ctx.Ledger.FormatMinerBlock(height, txList, t.ctx.Address, now.UnixNano(),
		t.ctx.State.GetLatestBlockid(), t.ctx.State.GetTotal(), consData)
	if err != nil {
		ctx.GetLog().Warn("format block error", "err", err)
		return nil, err
	}

	return block, nil
}

func (t *Miner) convertConsData(data []byte) *protos.ConsensusInfo {
	if data == nil {
		return nil
	}

	var consInfo *protos.ConsensusInfo
	err := json.Unmarshal(data, consInfo)
	if err != nil {
		return nil
	}

	return consInfo
}

func (t *Miner) getTimerTx(height int64) ([]*lpb.Transaction, error) {
	return nil, nil
}

func (t *Miner) getUnconfirmedTx(sizeLimit int) ([]*lpb.Transaction, error) {
	unconfirmedTxs, err := t.ctx.State.GetUnconfirmedTx(false)
	if err != nil {
		return nil, err
	}

	txList := make([]*lpb.Transaction, 0, len(unconfirmedTxs))
	for _, tx := range unconfirmedTxs {
		size := proto.Size(tx)
		if size > sizeLimit {
			break
		}
		sizeLimit -= size
		txList = append(txList, tx)
	}

	return txList, nil
}

func (t *Miner) getAwardTx(height int64) (*lpb.Transaction, error) {
	amount := t.ctx.Ledger.GenesisBlock.CalcAward(height)
	if amount.Cmp(big.NewInt(0)) < 0 {
		return nil, errors.New("amount in transaction can not be negative number")
	}

	txOutput := &protos.TxOutput{}
	txOutput.ToAddr = []byte(t.ctx.Address.Address)
	txOutput.Amount = amount.Bytes()
	awardTx := &lpb.Transaction{Version: state.RootTxVersion}
	awardTx.TxOutputs = append(awardTx.TxOutputs, txOutput)
	awardTx.Desc = []byte{'1'}
	awardTx.Coinbase = true
	awardTx.Timestamp = time.Now().UnixNano()
	awardTx.Txid, _ = txhash.MakeTransactionID(awardTx)
	return nil, nil
}

func (t *Miner) confirmBlock(ctx xctx.XContext, block *lpb.InternalBlock) error {
	// 需要转化下，为了共识做一些变更（比如pow）
	blkAgent := agent.NewBlockAgent(block)
	err := t.ctx.Consensus.CalculateBlock(blkAgent)

	// 账本确认区块
	confirmStatus := t.ctx.Ledger.ConfirmBlock(block, false)
	if confirmStatus.Succ {
		if confirmStatus.Orphan {
			ctx.GetLog().Warn("the mined blocked was attached to branch, no need to play")
			return errors.New("the mined blocked was attached to branch")
		}
		ctx.GetLog().Trace("ledger confirm block success", "height", block.Height, "blockId", utils.F(block.Blockid))
	} else {
		ctx.GetLog().Warn("ledger confirm block error", "confirm_status", confirmStatus)
		return errors.New("ledger confirm block error")
	}

	// 状态机确认区块
	err = t.ctx.State.PlayForMiner(block.Blockid)
	if err != nil {
		ctx.GetLog().Warn("state play error ", "error", err, "blockId", utils.F(block.Blockid))
		return err
	}

	// 共识确认区块
	err = t.ctx.Consensus.ProcessConfirmBlock(blkAgent)
	if err != nil {
		ctx.GetLog().Warn("consensus confirm block error", "err", err)
		return err
	}

	return nil
}

// 尝试检查同步节点账本到目标区块
// 如果不指定目标区块，则从临近节点查询获取网络状态
func (t *Miner) trySyncBlock(ctx xctx.XContext, targetBlock *lpb.InternalBlock) error {
	// 1.获取到同步目标高度
	var err error
	if targetBlock == nil {
		// 广播查询获取网络最新区块
		targetBlock, err = t.getWholeNetLongestBlock(ctx)
		if err != nil {
			ctx.GetLog().Warn("get whole network longest block failed,sync block exit", "err", err)
			return fmt.Errorf("try sync block get whole network longest block failed")
		}
	}

	// 2.获取矿工互斥锁，矿工行为完全串行
	t.minerMutex.Lock()
	defer func() {
		if err != nil {
			// 如果同步出错，更新到当前账本主干高度
			t.inSyncTargetHeight = t.ctx.Ledger.GetMeta().GetTrunkHeight()
			t.inSyncTargetBlockId = t.ctx.Ledger.GetMeta().GetTipBlockid()
		}
		// 释放矿工锁
		t.minerMutex.Unlock()
	}()

	// 3.检查同步目标，忽略目标高度小于正在同步高度的任务
	if targetBlock.GetHeight() < t.inSyncTargetHeight ||
		bytes.Equal(targetBlock.GetBlockid(), t.inSyncTargetBlockId) {
		ctx.GetLog().Trace("try sync block height lower than in sync height,ignore", "targetHeight",
			targetBlock.GetHeight(), "inSyncHeight", t.inSyncHeight, "inSyncTargetHeight",
			utils.F(t.inSyncTargetBlockId))
		return nil
	}

	// 4.更新同步中区块高度
	t.inSyncTargetHeight = targetBlock.GetHeight()
	t.inSyncTargetBlockId = targetBlock.GetBlockid()

	// 4.状态机walk，确保状态机和账本一致
	ledgerTipId := t.ctx.Ledger.GetMeta().GetTipBlockid()
	stateTipId := t.ctx.State.GetLatestBlockid()
	if !bytes.Equal(ledgerTipId, stateTipId) {
		err = t.ctx.State.Walk(ledgerTipId, false)
		if err != nil {
			ctx.GetLog().Warn("try sync block walk failed", "error", err, "ledgerTipId", utils.F(ledgerTipId),
				"stateTipId", utils.F(stateTipId))
			return fmt.Errorf("try sync block walk failed")
		}
	}

	// 5.启动同步区块到目标高度
	err = t.syncBlock(ctx, targetBlock)
	if err != nil {
		ctx.GetLog().Warn("try sync block failed", "err", err, "targetBlock", utils.F(targetBlock.GetBlockid()))
		return fmt.Errorf("try sync block failed")
	}

	ctx.GetLog().Trace("try sync block succ", "targetBlock", utils.F(targetBlock.GetBlockid()))
	return nil
}

func (t *Miner) getWholeNetLongestBlock(ctx xctx.XContext) (*lpb.InternalBlock, error) {
	input := &pb.BCStatus{Bcname: t.ctx.BCName}
	opt := []p2p.MessageOption{
		p2p.WithBCName(t.ctx.BCName),
		p2p.WithLogId(ctx.GetLog().GetLogId()),
	}
	msg := p2p.NewMessage(protos.XuperMessage_GET_BLOCKCHAINSTATUS, input, opt...)
	response, err := t.ctx.EngCtx.Net.SendMessageWithResponse(t.ctx, msg)
	if err != nil {
		ctx.GetLog().Warn("get block chain status error", "err", err)
		return nil, err
	}

	bcStatus := make([]*pb.BCStatus, 0, len(response))
	for _, resp := range response {
		var status pb.BCStatus
		err = p2p.Unmarshal(resp, &status)
		if err != nil {
			ctx.GetLog().Warn("unmarshal block chain status error", "err", err)
			continue
		}

		bcStatus = append(bcStatus, &status)
	}

	sort.Sort(BCSByHeight(bcStatus))
	for _, bcs := range bcStatus {
		if t.isConfirmed(bcs) {
			return bcs.Block, nil
		}
	}

	return nil, errors.New("not found longest block")
}

type BCSByHeight []*pb.BCStatus

func (s BCSByHeight) Len() int           { return len(s) }
func (s BCSByHeight) Less(i, j int) bool { return s[i].Meta.TrunkHeight > s[j].Meta.TrunkHeight }
func (s BCSByHeight) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (t *Miner) syncBlock(ctx xctx.XContext, targetBlock *lpb.InternalBlock) error {
	// 1.判断账本当前高度，忽略小于账本高度或者等于tip block任务
	if targetBlock.GetHeight() < t.ctx.Ledger.GetMeta().GetTrunkHeight() ||
		bytes.Equal(targetBlock.GetBlockid(), t.ctx.Ledger.GetMeta().GetTipBlockid()) {
		return nil
	}

	// 2.从临近节点拉取缺失区块(可优化为并发拉取，如果上个块)
	beginBlock, err := t.downloadMissBlock(ctx, targetBlock)
	if err != nil {
		return err
	}

	// 4.如果账本发生变更，触发同步账本和状态机
	defer func() {
		ledgerTipId := t.ctx.Ledger.GetMeta().GetTipBlockid()
		stateTipId := t.ctx.State.GetLatestBlockid()
		if !bytes.Equal(ledgerTipId, stateTipId) {
			ledgerTipId := t.ctx.Ledger.GetMeta().TipBlockid
			err := t.ctx.State.Walk(ledgerTipId, false)
			if err != nil {
				ctx.GetLog().Warn("sync block walk failed", "ledgerTipId", utils.F(ledgerTipId),
					"stateTipId", utils.F(stateTipId), "err", err)
				return
			}
			ctx.GetLog().Trace("sync block succ", "targetBlockId", utils.F(targetBlock.GetBlockid()))
		}
	}()

	// 3.将拉取到的区块加入账本
	err = t.batchConfirmBlock(ctx, beginBlock, targetBlock)
	if err != nil {
		return err
	}

	return nil
}

// 从临近节点下载区块保存到临时账本（可以优化为并发下载）
//
// case1: A => B => C(tipBlock) => D(beginBlock=targetBlock)
// case2: A => B => C(tipBlock) => D(beginBlock) => ... => E(targetBlock)
// case3: A => ... => B => ... => C(tipBlock)
//                    |
//                    | => D(beginBlock) => ... => E(targetBlock)
//
// 通过高度可以并发下载
func (t *Miner) downloadMissBlock(ctx xctx.XContext, targetBlock *lpb.InternalBlock) (*lpb.InternalBlock, error) {
	ledger := t.ctx.Ledger
	err := ledger.SavePendingBlock(targetBlock)
	if err != nil {
		ctx.GetLog().Warn("save pending block error", "blockId", targetBlock.Blockid, "err", err)
		return nil, err
	}

	beginBlock := targetBlock
	preHash := targetBlock.PreHash
	for !ledger.ExistBlock(preHash) {
		block, _ := ledger.GetPendingBlock(preHash)
		if block != nil {
			continue
		}

		input := &pb.BlockID{
			Bcname:      t.ctx.BCName,
			Blockid:     preHash,
			NeedContent: true,
		}
		block, err := t.GetBlock(ctx, input)
		if err != nil {
			ctx.GetLog().Warn("get block error", "err", err)
			return nil, err
		}

		err = ledger.SavePendingBlock(block)
		if err != nil {
			ctx.GetLog().Warn("save pending block error", "err", err)
			return nil, err
		}

		// TODO: 确定ledger接口返回值
		beginBlock = block.Block
		preHash = block.Block.PreHash
	}

	return beginBlock, nil
}

func (t *Miner) GetBlock(ctx xctx.XContext, input *pb.BlockID) (*pb.Block, error) {
	msg := p2p.NewMessage(protos.XuperMessage_GET_BLOCK, input, p2p.WithBCName(t.ctx.BCName))
	response, err := t.ctx.EngCtx.Net.SendMessageWithResponse(t.ctx, msg)
	if err != nil {
		ctx.GetLog().Warn("confirm block chain status error", "err", err)
		return nil, err
	}

	for _, msg := range response {
		if msg.GetHeader().GetErrorType() != protos.XuperMessage_SUCCESS {
			continue
		}

		var block *pb.Block
		err = p2p.Unmarshal(msg, block)
		if err != nil {
			ctx.GetLog().Warn("get block error", "err", err)
			continue
		}

		return block, nil
	}

	return nil, errors.New("no response")
}

// 批量追加区块到账本中
func (t *Miner) batchConfirmBlock(ctx xctx.XContext, beginBlock, endBlock *lpb.InternalBlock) error {
	block := beginBlock
	for blockId := block.Blockid; !bytes.Equal(blockId, endBlock.Blockid); blockId = block.NextHash {
		var err error
		block, err = t.ctx.Ledger.GetPendingBlock(blockId)
		if err != nil {
			ctx.GetLog().Warn("ledger get pending block error", "blockId", blockId, "err", err)
			return err
		}

		blockAgent := agent.NewBlockAgent(block)
		isMatch, err := t.ctx.Consensus.CheckMinerMatch(ctx, blockAgent)
		if !isMatch {
			ctx.GetLog().Warn("consensus check block error", "err", err)
			return errors.New("consensus check block error")
		}

		status := t.ctx.Ledger.ConfirmBlock(block, false)
		if !status.Succ {
			ctx.GetLog().Warn("ledger confirm block error", "err", status.Error)
			return errors.New("ledger confirm block error")
		}
	}

	blkAgent := agent.NewBlockAgent(endBlock)
	err := t.ctx.Consensus.ProcessConfirmBlock(blkAgent)
	if err != nil {
		ctx.GetLog().Warn("consensus confirm block error")
		return errors.New("consensus confirm block error")
	}

	return nil
}

// syncConfirm 向周围节点询问块是否可以被接受
func (t *Miner) isConfirmed(ctx xctx.XContext, bcs *pb.BCStatus) bool {
	input := &pb.BCStatus{Bcname: t.ctx.BCName, Block: &pb.InternalBlock{Blockid: bcs.Block.Blockid}}
	opt := []p2p.MessageOption{
		p2p.WithBCName(t.ctx.BCName),
		//p2p.WithLogId(ctx.GetLog().GetLogId()),
	}
	msg := p2p.NewMessage(protos.XuperMessage_CONFIRM_BLOCKCHAINSTATUS, input, opt...)
	response, err := t.ctx.EngCtx.Net.SendMessageWithResponse(t.ctx, msg)
	if err != nil {
		ctx.GetLog().Warn("confirm block chain status error", "err", err)
		return false
	}

	return countConfirmBlock(response)
}

// countConfirmBlockRes 对p2p网络返回的确认区块的结果进行统计
// 统计邻近节点的返回信息
func countConfirmBlock(messages []*protos.XuperMessage) bool {
	agreeCnt := 0
	disagreeCnt := 0
	for _, msg := range messages {
		var bts pb.BCTipStatus
		err := p2p.Unmarshal(msg, &bts)
		if err != nil {
			continue
		}

		if bts.GetIsTrunkTip() {
			agreeCnt++
		} else {
			disagreeCnt++
		}
	}

	// 支持的节点需要大于反对的节点，并且支持的节点个数需要大于res的1/3
	return agreeCnt >= disagreeCnt && agreeCnt >= len(messages)/3
}

// 广播新区块
// 三种块传播模式：
//  1. 一种是完全块广播模式(Full_BroadCast_Mode)，即直接广播原始块给所有相邻节点，
//     适用于出块矿工在知道周围节点都不具备该块的情况下；
//  2. 一种是问询式块广播模式(Interactive_BroadCast_Mode)，即先广播新块的头部给相邻节点，
//     相邻节点在没有相同块的情况下通过GetBlock主动获取块数据。
//  3. Mixed_BroadCast_Mode是指出块节点将新块用Full_BroadCast_Mode模式广播，
//     其他节点使用Interactive_BroadCast_Mode
// broadcast block in Full_BroadCast_Mode since it's the original miner
func (t *Miner) broadcastBlock(ctx xctx.XContext, block *lpb.InternalBlock) {
	engCtx := t.ctx.EngCtx
	var msg *protos.XuperMessage
	if engCtx.EngCfg.BlockBroadcastMode == common.InteractiveBroadCastMode {
		blockID := &lpb.InternalBlock{
			Blockid: block.Blockid,
		}
		msg = p2p.NewMessage(protos.XuperMessage_NEW_BLOCKID, blockID, p2p.WithBCName(t.ctx.BCName))
	} else {
		msg = p2p.NewMessage(protos.XuperMessage_SENDBLOCK, block, p2p.WithBCName(t.ctx.BCName))
	}

	err := engCtx.Net.SendMessage(t.ctx, msg)
	if err != nil {
		ctx.GetLog().Error("broadcast block error", "logid", msg.GetHeader().GetLogid(), "error", err)
	}

	return
}
