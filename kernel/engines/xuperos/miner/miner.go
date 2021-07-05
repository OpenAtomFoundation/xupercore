package miner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/agent"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/xpb"
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
	// 矿工处理区块的队列
	minerQueue int64
	// 记录同步中任务目标区块高度
	inSyncHeight       int64
	inSyncTargetHeight int64
	// 记录同步中任务目标区块id
	inSyncTargetBlockId []byte
	// 标记是否退出运行
	isExit bool
	// 用户等待退出
	exitWG sync.WaitGroup
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
		return common.ErrParameter
	}

	// 1.检查区块有效性和高度，忽略无效或者比当前同步高度低的区块
	blockSize := int64(proto.Size(block))
	maxBlockSize := t.ctx.State.GetMaxBlockSize()
	if blockSize > maxBlockSize {
		ctx.GetLog().Warn("forbidden proc block because block is too large",
			"blockSize", blockSize, "maxBlockSize", maxBlockSize)
		return common.ErrForbidden.More("block is too large")
	}

	if block.GetHeight() < t.inSyncTargetHeight || bytes.Equal(block.GetBlockid(), t.inSyncTargetBlockId) {
		ctx.GetLog().Trace("forbidden proc block because recv block height lower than in sync height",
			"recvHeight", block.GetHeight(), "recvBlockId", utils.F(block.GetBlockid()),
			"inSyncTargetHeight", t.inSyncTargetHeight, "inSyncTargetBlockId",
			utils.F(t.inSyncTargetBlockId))
		return common.ErrForbidden.More("%s", "recv block height lower than in sync height")
	}

	for id, tx := range block.Transactions {
		if !t.ctx.Ledger.IsValidTx(id, tx, block) {
			ctx.GetLog().Warn("forbidden proc block because invalid tx got from the block",
				"txid", utils.F(tx.Txid), "blockId", utils.F(block.Blockid))
			return common.ErrForbidden.More("%s", "invalid tx got from the block")
		}
	}

	atomic.AddInt64(&t.minerQueue, 1)
	defer atomic.AddInt64(&t.minerQueue, -1)
	if t.minerQueue >= t.ctx.EngCtx.EngCfg.MaxBlockQueueSize {
		ctx.GetLog().Warn("forbidden proc block because miner queue full", "minerQueue", t.minerQueue,
			"recvHeight", block.GetHeight(), "recvBlockId", utils.F(block.GetBlockid()),
			"inSyncTargetHeight", t.inSyncTargetHeight, "inSyncTargetBlockId", utils.F(t.inSyncTargetBlockId))
		return common.ErrForbidden.More("miner queue full")
	}

	ctx.GetLog().Debug("recv new block,try sync block", "recvHeight", block.Height,
		"recvBlockId", utils.F(block.Blockid), "inSyncTargetHeight", t.inSyncTargetHeight,
		"inSyncTargetBlockId", utils.F(t.inSyncTargetBlockId))

	// 尝试同步到该高度，如果小于账本高度会被直接忽略
	return t.trySyncBlock(ctx, block)
}

// 启动矿工，周期检查矿工身份
// 同一时间，矿工状态是唯一的。0:休眠中 1:同步区块中 2:打包区块中
func (t *Miner) Start() {
	// 用于监测退出
	t.exitWG.Add(1)
	defer t.exitWG.Done()

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
			ctx.GetLog().Trace("compete master result", "height", ledgerTipHeight+1, "isMiner", isMiner, "isSync", isSync, "err", err)
		}
		// 3.如需要同步，尝试同步网络最新区块
		if err == nil && isMiner && isSync {
			err = t.trySyncBlock(ctx, nil)
		}
		// 4.如果是矿工，出块
		if err == nil && isMiner {
			err = t.mining(ctx)
		}
		// 5.如果出错，休眠3s后重试，防止cpu被打满
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

	t.log.Trace("miner exited", "ledgerTipHeight", ledgerTipHeight,
		"ledgerTipId", utils.F(ledgerTipId), "stateTipId", utils.F(stateTipId))
}

// 停止矿工
func (t *Miner) Stop() {
	t.isExit = true
	t.exitWG.Wait()
}

func (t *Miner) IsExit() bool {
	return t.isExit
}

// 挖矿生产区块
func (t *Miner) mining(ctx xctx.XContext) error {
	ctx.GetLog().Debug("mining start.")
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
		stateTipId = t.ctx.State.GetLatestBlockid()
	}

	// 3.共识挖矿前处理
	height := t.ctx.Ledger.GetMeta().TrunkHeight + 1
	now := time.Now()
	truncateTarget, extData, err := t.ctx.Consensus.ProcessBeforeMiner(now.UnixNano())
	if err != nil {
		ctx.GetLog().Warn("consensus process before miner failed", "err", err)
		return fmt.Errorf("consensus process before miner failed")
	}
	ctx.GetLog().Debug("consensus before miner succ", "truncateTarget", truncateTarget, "extData", string(extData))
	if truncateTarget != nil {
		// 裁剪掉账本目标区块，裁掉的交易判断冲突重新回放，裁剪完后继续出块操作
		if err := t.truncateForMiner(ctx, truncateTarget); err != nil {
			return err
		}
		// 重置高度
		height = t.ctx.Ledger.GetMeta().TrunkHeight + 1
	}

	// 4.打包区块
	block, err := t.packBlock(ctx, height, now, extData)
	if err != nil {
		ctx.GetLog().Warn("pack block error", "err", err)
		return err
	}
	ctx.GetLog().Debug("pack block succ", "height", height, "blockId", utils.F(block.GetBlockid()))

	// 5.账本&状态机&共识确认新区块
	err = t.confirmBlockForMiner(ctx, block)
	if err != nil {
		ctx.GetLog().Warn("confirm block for miner failed", "err", err,
			"blockId", utils.F(block.GetBlockid()))
		return err
	}

	// 6.异步广播新生成的区块
	go t.broadcastBlock(ctx, block)

	ctx.GetLog().Trace("complete new block generation", "blockId", utils.F(block.GetBlockid()),
		"height", height, "costs", ctx.GetTimer().Print())
	return nil
}

// 裁剪掉账本最新的区块
func (t *Miner) truncateForMiner(ctx xctx.XContext, target []byte) error {
	_, err := t.ctx.Ledger.QueryBlockHeader(target)
	if err != nil {
		ctx.GetLog().Warn("truncate failed because query target error", "err", err)
		return err
	}

	// 状态机回滚到目标状态
	err = t.ctx.State.Walk(target, false)
	if err != nil {
		ctx.GetLog().Warn("truncate failed because state walk error", "ledgerTipId", utils.F(t.ctx.Ledger.GetMeta().TipBlockid),
			"walkTargetBlockId", utils.F(target))
		return err
	}

	// 账本裁剪到这个区块
	err = t.ctx.Ledger.Truncate(target)
	if err != nil {
		ctx.GetLog().Warn("truncate failed because ledger truncate error", "err", err)
		return err
	}

	return nil
}

func (t *Miner) packBlock(ctx xctx.XContext, height int64,
	now time.Time, consData []byte) (*lpb.InternalBlock, error) {
	// 区块大小限制
	sizeLimit, err := t.ctx.State.MaxTxSizePerBlock()
	if err != nil {
		return nil, err
	}
	ctx.GetLog().Debug("pack block get max size succ", "sizeLimit", sizeLimit)

	// 1.生成timer交易
	autoTx, err := t.getTimerTx(height)
	if err != nil {
		return nil, err
	}
	if len(autoTx.TxOutputsExt) > 0 {
		sizeLimit -= proto.Size(autoTx)
	}

	ctx.GetLog().Debug("pack block get timer tx succ", "auto tx", autoTx)

	// 2.选择本次要打包的tx
	generalTxList, err := t.getUnconfirmedTx(sizeLimit)
	if err != nil {
		return nil, err
	}
	ctx.GetLog().Debug("pack block get general tx succ", "txCount", len(generalTxList))

	// 3.获取矿工奖励交易
	awardTx, err := t.getAwardTx(height)
	if err != nil {
		return nil, err
	}
	ctx.GetLog().Debug("pack block get award tx succ", "txid", utils.F(awardTx.GetTxid()))

	txList := make([]*lpb.Transaction, 0)
	txList = append(txList, awardTx)
	if len(autoTx.TxOutputsExt) > 0 {
		txList = append(txList, autoTx)
	}
	if len(generalTxList) > 0 {
		txList = append(txList, generalTxList...)
	}

	// 4.打包区块
	consInfo, err := t.convertConsData(consData)
	if err != nil {
		ctx.GetLog().Warn("convert consensus data failed", "err", err, "consData", string(consData))
		return nil, fmt.Errorf("convert consensus data failed")
	}
	block, err := t.ctx.Ledger.FormatMinerBlock(txList, []byte(t.ctx.Address.Address),
		t.ctx.Address.PrivateKey, now.UnixNano(), consInfo.CurTerm, consInfo.CurBlockNum,
		t.ctx.State.GetLatestBlockid(), consInfo.TargetBits, t.ctx.State.GetTotal(),
		consInfo.Justify, nil, height)
	if err != nil {
		ctx.GetLog().Warn("format block error", "err", err)
		return nil, err
	}

	return block, nil
}

func (t *Miner) convertConsData(data []byte) (*agent.ConsensusStorage, error) {
	var consInfo agent.ConsensusStorage
	if len(data) < 1 {
		return &consInfo, nil
	}

	err := json.Unmarshal(data, &consInfo)
	if err != nil {
		return nil, err
	}

	return &consInfo, nil
}

func (t *Miner) getTimerTx(height int64) (*lpb.Transaction, error) {
	autoTx, err := t.ctx.State.GetTimerTx(height)
	if err != nil {
		t.log.Error("Get timer tx error", "error", err)
		return nil, common.ErrGenerateTimerTxFailed
	}

	return autoTx, nil
}

func (t *Miner) getUnconfirmedTx(sizeLimit int) ([]*lpb.Transaction, error) {
	unconfirmedTxs, err := t.ctx.State.GetUnconfirmedTx(false)
	if err != nil {
		return nil, err
	}

	txList := make([]*lpb.Transaction, 0)
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

	awardTx, err := tx.GenerateAwardTx(t.ctx.Address.Address, amount.String(), []byte("award"))
	if err != nil {
		return nil, err
	}

	return awardTx, nil
}

func (t *Miner) confirmBlockForMiner(ctx xctx.XContext, block *lpb.InternalBlock) error {
	// 需要转化下，为了共识做一些变更（比如pow）
	origBlkId := block.Blockid
	blkAgent := agent.NewBlockAgent(block)
	err := t.ctx.Consensus.CalculateBlock(blkAgent)
	if err != nil {
		ctx.GetLog().Warn("consensus calculate block failed", "err", err,
			"blockId", utils.F(block.Blockid))
		return fmt.Errorf("consensus calculate block failed")
	}
	ctx.GetLog().Trace("start confirm block for miner", "originalBlockId", utils.F(origBlkId),
		"newBlockId", utils.F(block.Blockid))

	// 账本确认区块
	confirmStatus := t.ctx.Ledger.ConfirmBlock(block, false)
	if confirmStatus.Succ {
		if confirmStatus.Orphan {
			ctx.GetLog().Trace("the mined blocked was attached to branch,no need to play",
				"blockId", utils.F(block.Blockid))
			return nil
		}
		ctx.GetLog().Trace("ledger confirm block success", "height", block.Height,
			"blockId", utils.F(block.Blockid))
	} else {
		ctx.GetLog().Warn("ledger confirm block failed", "err", confirmStatus.Error,
			"blockId", utils.F(block.Blockid))
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
		ctx.GetLog().Warn("consensus confirm block error", "err", err,
			"blockId", utils.F(block.Blockid))
		return err
	}

	ctx.GetLog().Trace("confirm block for miner succ", "blockId", utils.F(block.Blockid))
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
	ctx.GetLog().Debug("Miner::trySyncBlock", "targetBlockId", utils.F(targetBlock.GetBlockid()), "targetHeight", targetBlock.GetHeight(),
		"inSyncTargetBlockId", utils.F(t.inSyncTargetBlockId), "inSyncTargetHeight", t.inSyncTargetHeight)

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
		ctx.GetLog().Trace("ignore block because target block height lower than in sync height",
			"targetBlockHeight", targetBlock.GetHeight(), "targetBlockBlockId",
			utils.F(targetBlock.GetBlockid()), "inSyncTargetHeight", t.inSyncTargetHeight,
			"inSyncTargetBlockId", utils.F(t.inSyncTargetBlockId))
		return nil
	}
	// 检查同步目标是否已经在账本中，忽略已经在账本中任务
	if t.ctx.Ledger.ExistBlock(targetBlock.GetBlockid()) {
		ctx.GetLog().Trace("ignore block because target block has in ledger", "targetBlockId",
			utils.F(targetBlock.GetBlockid()))
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
			ctx.GetLog().Warn("try sync block walk failed", "error", err,
				"ledgerTipId", utils.F(ledgerTipId), "stateTipId", utils.F(stateTipId))
			return fmt.Errorf("try sync block walk failed")
		}
	}

	// 5.启动同步区块到目标高度
	err = t.syncBlock(ctx, targetBlock)
	if err != nil {
		ctx.GetLog().Warn("try sync block failed", "err", err, "targetBlock",
			utils.F(targetBlock.GetBlockid()))
		return fmt.Errorf("try sync block failed")
	}

	ctx.GetLog().Trace("try sync block succ", "targetBlock", utils.F(targetBlock.GetBlockid()))
	return nil
}

// 同步本地账本到指定的目标高度
func (t *Miner) syncBlock(ctx xctx.XContext, targetBlock *lpb.InternalBlock) error {
	// 1.判断账本当前高度，忽略小于账本高度或者等于tip block任务
	if targetBlock.GetHeight() < t.ctx.Ledger.GetMeta().GetTrunkHeight() ||
		bytes.Equal(targetBlock.GetBlockid(), t.ctx.Ledger.GetMeta().GetTipBlockid()) {
		return nil
	}

	// 2.从临近节点拉取本地缺失的区块
	// 可优化为并发拉取，可以优化为批处理，方便查看同步进度
	blkIds, err := t.downloadMissBlock(ctx, targetBlock)
	if err != nil {
		ctx.GetLog().Warn("download miss block failed", "err", err)
		return fmt.Errorf("download miss block failed")
	}

	// 4.如果账本发生变更，触发同步账本和状态机
	defer func() {
		ledgerTipId := t.ctx.Ledger.GetMeta().GetTipBlockid()
		stateTipId := t.ctx.State.GetLatestBlockid()
		if !bytes.Equal(ledgerTipId, stateTipId) {
			ledgerTipId := t.ctx.Ledger.GetMeta().TipBlockid
			// Walk相比PalyAndReport代价更高，后面可以优化下
			err := t.ctx.State.Walk(ledgerTipId, false)
			if err != nil {
				ctx.GetLog().Warn("sync block walk failed", "ledgerTipId", utils.F(ledgerTipId),
					"stateTipId", utils.F(stateTipId), "err", err)
				return
			}
		}
	}()

	// 3.将拉取到的区块加入账本
	ctx.GetLog().Debug("batch confirm block", "blockCount", len(blkIds))
	err = t.batchConfirmBlock(ctx, blkIds)
	if err != nil {
		ctx.GetLog().Warn("batch confirm block to ledger failed", "err", err, "blockCount", len(blkIds))
		return fmt.Errorf("batch confirm block to ledger failed")
	}

	return nil
}

// 从临近节点下载区块保存到临时账本（可以优化为并发下载）
func (t *Miner) downloadMissBlock(ctx xctx.XContext,
	targetBlock *lpb.InternalBlock) ([][]byte, error) {
	// 记录下载到的区块id
	blkIds := make([][]byte, 0)

	// 先把targetBlock存入缓存栈
	ledger := t.ctx.Ledger
	err := ledger.SavePendingBlock(targetBlock)
	if err != nil {
		ctx.GetLog().Warn("save pending block error", "blockId", targetBlock.Blockid, "err", err)
		return blkIds, err
	}
	blkIds = append(blkIds, targetBlock.GetBlockid())

	beginBlock := targetBlock
	for !ledger.ExistBlock(beginBlock.PreHash) {
		if len(beginBlock.PreHash) <= 0 || beginBlock.Height == 0 {
			ctx.GetLog().Error("the genesis block is different",
				"genesisBlockId", utils.F(ledger.GetMeta().RootBlockid),
				"syncGenesisBlockId", utils.F(beginBlock.Blockid))
			return nil, common.ErrGenesisBlockDiff
		}

		block, _ := ledger.GetPendingBlock(beginBlock.PreHash)
		if block != nil {
			beginBlock = block
			blkIds = append(blkIds, block.GetBlockid())
			continue
		}

		// 从临近节点下载区块
		block, err := t.getBlock(ctx, beginBlock.PreHash)
		if err != nil {
			ctx.GetLog().Warn("get block error", "err", err)
			return blkIds, err
		}
		// 保存区块到本地栈中
		err = ledger.SavePendingBlock(block)
		if err != nil {
			ctx.GetLog().Warn("save pending block error", "err", err)
			return blkIds, err
		}
		beginBlock = block
		blkIds = append(blkIds, block.GetBlockid())
	}

	return blkIds, nil
}

func (t *Miner) getBlock(ctx xctx.XContext, blockId []byte) (*lpb.InternalBlock, error) {
	input := &xpb.BlockID{
		Bcname:      t.ctx.BCName,
		Blockid:     blockId,
		NeedContent: true,
	}

	opts := []p2p.MessageOption{
		p2p.WithBCName(t.ctx.BCName),
		// p2p.WithLogId(ctx.GetLog().GetLogId()),
	}
	msg := p2p.NewMessage(protos.XuperMessage_GET_BLOCK, input, opts...)
	responses, err := t.ctx.EngCtx.Net.SendMessageWithResponse(ctx, msg)
	if err != nil {
		ctx.GetLog().Warn("confirm block chain status error", "err", err)
		return nil, err
	}

	for _, response := range responses {
		if response.GetHeader().GetErrorType() != protos.XuperMessage_SUCCESS {
			continue
		}

		var block xpb.BlockInfo
		err = p2p.Unmarshal(response, &block)
		if err != nil {
			ctx.GetLog().Warn("unmarshal block error", "err", err)
			continue
		}

		if block.Block == nil {
			ctx.GetLog().Warn("block is nil", "blockId", utils.F(blockId),
				"from", response.GetHeader().GetFrom())
			continue
		}

		ctx.GetLog().Trace("download block succ", "height", block.Block.Height,
			"blockId", utils.F(block.Block.Blockid), "msg_log_id", msg.Header.Logid)
		return block.Block, nil
	}

	return nil, errors.New("no response")
}

// 追加区块到账本中
func (t *Miner) batchConfirmBlock(ctx xctx.XContext, blkIds [][]byte) error {
	if len(blkIds) < 1 {
		return nil
	}

	for index := len(blkIds) - 1; index >= 0; index-- {
		block, err := t.ctx.Ledger.GetPendingBlock(blkIds[index])
		if err != nil {
			ctx.GetLog().Warn("ledger get pending block error",
				"blockId", utils.F(blkIds[index]), "err", err)
			return fmt.Errorf("get pending block failed from ledger")
		}

		valid, err := t.ctx.Ledger.VerifyBlock(block, ctx.GetLog().GetLogId())
		if !valid {
			ctx.GetLog().Warn("the verification of block failed.",
				"blockId", utils.F(blkIds[index]))
			return fmt.Errorf("the verification of block failed from ledger.")
		}
		blockAgent := agent.NewBlockAgent(block)
		isMatch, err := t.ctx.Consensus.CheckMinerMatch(ctx, blockAgent)
		if !isMatch {
			ctx.GetLog().Warn("consensus check miner match failed",
				"blockId", utils.F(blkIds[index]), "err", err)
			return errors.New("consensus check miner match failed")
		}

		status := t.ctx.Ledger.ConfirmBlock(block, false)
		if !status.Succ {
			ctx.GetLog().Warn("ledger confirm block failed",
				"blockId", utils.F(blkIds[index]), "err", status.Error)
			return errors.New("ledger confirm block failed")
		}

		err = t.ctx.Consensus.ProcessConfirmBlock(blockAgent)
		if err != nil {
			ctx.GetLog().Warn("consensus process confirm block failed",
				"blockId", utils.F(blkIds[index]), "err", err)
			return errors.New("consensus process confirm block failed")
		}
	}

	ctx.GetLog().Trace("batch confirm block to ledger succ", "blockCount", len(blkIds))
	return nil
}

// syncConfirm 向周围节点询问块是否可以被接受
func (t *Miner) isConfirmed(ctx xctx.XContext, bcs *xpb.ChainStatus) bool {
	input := &lpb.InternalBlock{Blockid: bcs.Block.Blockid}
	opts := []p2p.MessageOption{
		p2p.WithBCName(t.ctx.BCName),
		p2p.WithLogId(ctx.GetLog().GetLogId()),
	}
	msg := p2p.NewMessage(protos.XuperMessage_CONFIRM_BLOCKCHAINSTATUS, input, opts...)
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
		var bts xpb.TipStatus
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
	opts := []p2p.MessageOption{
		p2p.WithBCName(t.ctx.BCName),
		p2p.WithLogId(ctx.GetLog().GetLogId()),
	}
	var msg *protos.XuperMessage
	if engCtx.EngCfg.BlockBroadcastMode == common.InteractiveBroadCastMode {
		blockID := &lpb.InternalBlock{
			Blockid: block.Blockid,
		}
		msg = p2p.NewMessage(protos.XuperMessage_NEW_BLOCKID, blockID, opts...)
	} else {
		msg = p2p.NewMessage(protos.XuperMessage_SENDBLOCK, block, opts...)
	}

	err := engCtx.Net.SendMessage(t.ctx, msg)
	if err != nil {
		ctx.GetLog().Warn("broadcast block error", "p2pLogId", msg.GetHeader().GetLogid(), "err", err,
			"blockId", utils.F(block.GetBlockid()))
		return
	}

	ctx.GetLog().Trace("broadcast block succ", "p2pLogId", msg.GetHeader().GetLogid(),
		"blockId", utils.F(block.GetBlockid()))
	return
}

func (t *Miner) getWholeNetLongestBlock(ctx xctx.XContext) (*lpb.InternalBlock, error) {
	opt := []p2p.MessageOption{
		p2p.WithBCName(t.ctx.BCName),
		p2p.WithLogId(ctx.GetLog().GetLogId()),
	}
	msg := p2p.NewMessage(protos.XuperMessage_GET_BLOCKCHAINSTATUS, nil, opt...)
	responses, err := t.ctx.EngCtx.Net.SendMessageWithResponse(t.ctx, msg)
	if err != nil {
		ctx.GetLog().Warn("get block chain status error", "err", err)
		return nil, err
	}

	bcStatus := make([]*xpb.ChainStatus, 0, len(responses))
	for _, response := range responses {
		var status xpb.ChainStatus
		err = p2p.Unmarshal(response, &status)
		if err != nil {
			ctx.GetLog().Warn("unmarshal block chain status error", "err", err)
			continue
		}

		bcStatus = append(bcStatus, &status)
	}

	sort.Sort(BCSByHeight(bcStatus))
	for _, bcs := range bcStatus {
		if t.isConfirmed(ctx, bcs) {
			return bcs.Block, nil
		}
	}

	return nil, errors.New("not found longest block")
}

type BCSByHeight []*xpb.ChainStatus

func (s BCSByHeight) Len() int {
	return len(s)
}
func (s BCSByHeight) Less(i, j int) bool {
	return s[i].LedgerMeta.TrunkHeight > s[j].LedgerMeta.TrunkHeight
}
func (s BCSByHeight) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
