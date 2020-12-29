package xuperos

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xuperchain/core/common"
	"github.com/xuperchain/xuperchain/core/common/config"
	"github.com/xuperchain/xuperchain/core/global"
	"github.com/xuperchain/xuperchain/core/pb"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/lib/timer"
)

// 负责生产和同步区块
type Miner struct {
	ctx *def.ChainCtx
	log logs.Logger
	// 矿工锁，用来确保矿工出块和同步操作串行进行
	minerMutex sync.Mutex
	// 记录同步中任务目标区块高度
	inSyncTargetHeight int
	// 记录同步中任务目标区块id
	inSyncTargetBlockId int
	// 标记是否退出运行
	isExit bool
}

func NewMiner(ctx *def.ChainCtx) *Miner {
	obj := &miner{
		ctx: ctx,
		log: ctx.GetLog(),
	}

	return obj
}

// 处理P2P网络中接收到的区块
func (t *miner) ProcBlock(bctx xctx.XContext, block *lpb.InternalBlock) error {
	if bctx == nil || block == nil {
		return fmt.Errorf("param error")
	}
	log := bctx.GetLog()

	// 1.检查区块有效性和高度，忽略无效或者比当前同步高度低的区块
	blockSize := int64(proto.Size(block))
	maxBlockSize := t.ctx.State.GetMaxBlockSize()
	if blockSize > maxBlockSize {
		log.Warn("refused proc block because block is too large",
			"blockSize", blockSize, "maxBlockSize", maxBlockSize)
		return fmt.Errorf("refused proc block")
	}

	inSyncTargetHeight := t.inSyncTargetHeight
	inSyncTargetBlockId := t.inSyncTargetBlockId
	if block.GetHeight() < inSyncTargetHeight || bytes.Equal(block.GetBlockid(), inSyncTargetBlockId) {
		log.Trace("recv block height lower than in sync height,ignore", "recvHeight",
			block.GetHeight(), "inSyncTargetHeight", inSyncTargetHeight,
			"inSyncTargetBlockId", utils.F(inSyncTargetHeight))
		return nil
	}

	for id, tx := range block.Transactions {
		if !t.ctx.Ledger.IsValidTx(id, tx, block) {
			log.Warn("invalid tx got from the block", "txid", utils.F(tx.Txid),
				"blockId", utils.F(block.Blockid))
			return fmt.Errorf("invalid tx got from the block")
		}
	}

	// 尝试同步到该高度，如果小于账本高度会被直接忽略
	return t.trySyncBlock(block, log)
}

// 启动矿工，周期检查矿工身份
// 同一时间，矿工状态是唯一的。0:休眠中 1:同步区块中 2:打包区块中
func (t *miner) Start() {
	// 启动矿工循环
	err := nil
	isMiner := false
	isSync := false
	ledgerTipId := t.ctx.Ledger.GetMeta().TipBlockid
	ledgerTipHeight := t.ctx.Ledger.GetMeta().TrunkHeight
	stateTipId := t.ctx.State.GetLatestBlockid()
	for !t.isExit() {
		t.log.Trace("miner running", "ledgerTipHeight", ledgerTipHeight, "ledgerTipId",
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
			err = t.trySyncBlock(nil, t.log)
		}
		// 4.如果是矿工，出块
		if err == nil && isMiner {
			err = t.mining()
		}
		// 5.如果出错，休眠3s后重试
		if !t.isExit() && err != nil {
			t.log.Warn("miner run occurred error,sleep 3s try", "err", err)
			time.Sleep(3 * time.Second)
		}
		// 6.更新状态
		if !isExit {
			err = nil
			ledgerTipId = t.ctx.Ledger.GetMeta().TipBlockid
			ledgerTipHeight = t.ctx.Ledger.GetMeta().TrunkHeight
			stateTipId = t.ctx.State.GetLatestBlockid()
		}
	}

	t.log.Trace("miner exited", "ledgerTipHeight", ledgerTipHeight)
}

// 停止矿工
func (t *miner) Stop() {
	t.isExit = true
}

func (t *miner) isExit() bool {
	return t.isExit
}

// 挖矿生产区块
func (t *miner) mining() error {
	log, _ := logs.NewLogger("", "miner")
	tmr := timer.NewXTimer()

	// 1.获取矿工互斥锁，矿工行为完全串行
	t.minerMutex.Lock()
	defer t.minerMutex.Unlock()

	// 2.状态机walk，确保状态机和账本一致
	ledgerTipId := t.ctx.Ledger.GetMeta().TipBlockid
	stateTipId := t.ctx.State.GetLatestBlockid()
	if !bytes.Equal(ledgerTipId, stateTipId) {
		err = t.ctx.State.Walk(ledgerTipId, false)
		if err != nil {
			log.Warn("mining walk failed", "ledgerTipId", utils.F(ledgerTipId),
				"stateTipId", utils.F(stateTipId))
			return fmt.Errorf("mining walk failed")
		}
		stateTipId = ledgerTipId
	}

	// 3.共识挖矿前处理
	outBlockHeight := t.ctx.Ledger.GetMeta().TrunkHeight + 1
	start := time.Now()
	isTruncate, extData, err := t.ctx.Consensus.ProcessBeforeMiner(start.UnixNano())
	if err != nil {
		log.Warn("consensus process before miner failed", "err", err)
		return fmt.Errorf("consensus process before miner failed")
	}
	if isTruncate {
		// 裁剪掉账本最高区块，裁掉的交易判断冲突重新回放，裁剪完后结束本次出块操作
		return t.truncateForMiner(ledgerTipId)
	}
	// 适配转化共识附加字段
	consData := t.convertConsData(extData)

	// 4.查询timer异步交易
	timerTxList, err := t.getTimerTx(outBlockHeight)

	// 5.选择本次要打包的tx
	generalTxList, err := t.selectGeneralTx(sizeLimit)

	// 6.获取矿工奖励交易
	awardTx, err := t.getAwardTx(outBlockHeight)

	// 7.打包新区块
	txList := make([]*pb.Transaction, 0)
	if len(timerTxList) > 0 {
		txList = append(txList, timerTxList...)
	}
	if len(generalTxList) > 0 {
		txList = append(txList, generalTxList...)
	}
	txList = append(txList, awardTx)
	newBlock, err := t.packMinerBlock(outBlockHeight, txList, consData)

	// 8.确认新区块到账本&状态机
	err = t.confirmMinerBlock(newBlock)

	// 9.异步广播新生成的区块
	go t.broadcastMinerBlock(newBlock)

	log.Trace("complete new block generation", "blockId", utils.F(newBlock.GetBlockid()),
		"height", outBlockHeight, "costs", tmr.Print())
	return nil
}

// 尝试检查同步节点账本到目标区块
// 如果不指定目标区块，则从临近节点查询获取网络状态
func (t *miner) trySyncBlock(targetBlock *lpb.InternalBlock, log logs.Logger) error {
	// 1.获取到同步目标高度
	err := nil
	if targetBlock == nil {
		// 广播查询获取网络最新区块
		targetBlock, err = t.getWholeNetLongestBlock(log)
		if err != nil {
			log.Warn("get whole network longest block failed,sync block exit", "err", err)
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
		log.Trace("try sync block height lower than in sync height,ignore", "targetHeight",
			targetBlock.GetHeight(), "insyncHeight", inSyncHeight, "inSyncTargetHeight",
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
			log.Warn("try sync block walk failed", "ledgerTipId", utils.F(ledgerTipId),
				"stateTipId", utils.F(stateTipId))
			return fmt.Errorf("try sync block walk failed")
		}
	}

	// 5.启动同步区块到目标高度
	err = t.syncBlock(targetBlock, log)
	if err != nil {
		log.Warn("try sync block failed", "err", err, "targetBlock", utils.F(targetBlock.GetBlockid()))
		return fmt.Errorf("try sync block failed")
	}

	log.Trace("try sync block succ", "targetBlock", utils.F(targetBlock.GetBlockid()))
	return nil
}

func (t *miner) syncBlock(targetBlock *lpb.InternalBlock, log logs.Logger) error {
	// 1.判断账本当前高度，忽略小于账本高度或者等于tip block任务
	if targetBlock.GetHeight() < t.ctx.Ledger.GetMeta().GetTrunkHeight() ||
		bytes.Equal(targetBlock.GetBlockid(), t.ctx.Ledger.GetMeta().GetTipBlockid()) {
		return nil
	}

	// 2.从临近节点拉取缺失区块(可优化为并发拉取，如果上个块)
	blkIds, err := t.downloadMissBlock(targetBlock, log)
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
				log.Warn("sync block walk failed", "ledgerTipId", utils.F(ledgerTipId),
					"stateTipId", utils.F(stateTipId), "err", err)
				return
			}
			log.Trace("sync block succ", "targetBlockId", utils.F(targetBlock.GetBlockid()))
		}
	}()

	// 3.将拉取到的区块加入账本
	err = t.batchConfirmBlock(blkIds)
	if err != nil {
		return err
	}

	return nil
}

// 从临近节点下载区块保存到临时账本（可以优化为并发下载）
func (t *miner) downloadMissBlock(targetBlock *lpb.InternalBlock, log logs.Logger) ([][]byte, error) {

}

// 批量追加区块到账本中
func (t *miner) batchConfirmBlock(blkIds [][]byte) error {

}

// countGetBlockChainStatus 对p2p网络返回的结果进行统计
func countGetBlockChainStatus(hbcs []*xuper_p2p.XuperMessage) *pb.BCStatus {
	p := hbcs[0]
	maxCount := 0
	countHeight := make(map[int64]int)
	for i := 1; i < len(hbcs); i++ {
		bcStatus := &pb.BCStatus{}
		err := proto.Unmarshal(p.GetData().GetMsgInfo(), bcStatus)
		if err != nil {
			continue
		}
		countHeight[bcStatus.GetMeta().GetTrunkHeight()]++
		if countHeight[bcStatus.GetMeta().GetTrunkHeight()] >= maxCount {
			p = hbcs[i]
			maxCount = countHeight[bcStatus.GetMeta().GetTrunkHeight()]
		}
	}
	res := &pb.BCStatus{}
	err := proto.Unmarshal(p.GetData().GetMsgInfo(), res)
	if err != nil {
		return nil
	}
	return res
}

// syncConfirm 向周围节点询问块是否可以被接受
func (xc *XChainCore) syncConfirm(bcs *pb.BCStatus) bool {
	bcsBuf, err := proto.Marshal(bcs)
	msg, err := p2p_base.NewXuperMessage(p2p_base.XuperMsgVersion2, bcs.GetBcname(), "", xuper_p2p.XuperMessage_CONFIRM_BLOCKCHAINSTATUS, bcsBuf, xuper_p2p.XuperMessage_NONE)
	filters := []p2p_base.FilterStrategy{p2p_base.NearestBucketStrategy}
	whiteList := xc.groupChain.GetAllowedPeersWithBcname(xc.bcname)
	opts := []p2p_base.MessageOption{
		p2p_base.WithFilters(filters),
		p2p_base.WithBcName(xc.bcname),
		p2p_base.WithWhiteList(whiteList),
	}
	res, err := xc.P2pSvr.SendMessageWithResponse(context.Background(), msg, opts...)
	if err != nil {
		return false
	}

	return countConfirmBlockRes(res)
}

// countConfirmBlockRes 对p2p网络返回的确认区块的结果进行统计
func countConfirmBlockRes(res []*xuper_p2p.XuperMessage) bool {
	// 统计邻近节点的返回信息
	agreeCnt := 0
	disAgresCnt := 0
	for i := 0; i < len(res); i++ {
		bts := &pb.BCTipStatus{}
		err := proto.Unmarshal(res[i].GetData().GetMsgInfo(), bts)
		if err != nil {
			continue
		}
		if bts.GetIsTrunkTip() {
			agreeCnt++
		} else {
			disAgresCnt++
		}
	}
	// 支持的节点需要大于反对的节点，并且支持的节点个数需要大于res的1/3
	return agreeCnt >= disAgresCnt && agreeCnt >= len(res)/3
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
func (t *miner) broadcastBlock(freshBlock *pb.InternalBlock) {
	state := t.ctx.Net.P2PState()
	block := &pb.Block{
		Header: &pb.Header{
			Logid:    t.logID,
			FromNode: state.PeerId,
		},
		Bcname:  t.ctx.BCName,
		Blockid: freshBlock.Blockid,
	}

	var msg *netPB.XuperMessage
	opts := []p2p.MessageOption{
		p2p.WithBCName(t.ctx.BCName),
		p2p.WithLogId(t.logID),
	}
	if t.ctx.EngCfg.BlockBroadcastMode == def.InteractiveBroadCastMode {
		msg = p2p.NewMessage(netPB.XuperMessage_NEW_BLOCKID, block, opts...)
	} else {
		block.Block = freshBlock
		msg = p2p.NewMessage(netPB.XuperMessage_SENDBLOCK, block, opts...)
	}

	err := t.ctx.Net.SendMessage(context.Background(), msg)
	if err != nil {
		t.log.Error("broadcast block error", "logid", t.logID, "error", err)
	}

	return
}
