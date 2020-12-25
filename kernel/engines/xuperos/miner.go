package xuperos

import (
	"bytes"
	"context"
	"fmt"
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
	"time"

	netPB "github.com/xuperchain/xupercore/kernel/network/pb"
)

// 封装矿工这个角色的所有行为
type miner struct {
	ctx *def.ChainCtx
	log logs.Logger

	keeper *LedgerKeeper

	logID string // 每次出块拥有唯一logID
	timer *timer.XTimer

	exitCh chan struct{}
}

func NewMiner(ctx *def.ChainCtx, keeper *LedgerKeeper) *miner {
	ctx.Status = def.MinerSafeModel
	obj := &miner{
		ctx:    ctx,
		log:    ctx.XLog,
		keeper: keeper,
		exitCh: make(chan struct{}),
	}

	return obj
}

// 处理广播区块
func (t *miner) ProcBlock() error {
	hd := &global.XContext{Timer: global.NewXTimer()}
	if t.ctx.Ledger.ExistBlock(in.GetBlock().GetBlockid()) {
		t.log.Debug("block is exist", "logid", in.Header.Logid, "cost", hd.Timer.Print())
		return nil
	}

	if bytes.Equal(t.ctx.State.GetLatestBlockId(), in.GetBlock().GetPreHash()) {
		t.log.Trace("appending block in SendBlock", "time", time.Now().UnixNano(), "bcName", t.ctx.BCName, "tipID", global.F(t.ctx.State.GetLatestBlockId()))
		ctx := CreateLedgerTaskCtx([]*SimpleBlock{
			&SimpleBlock{
				internalBlock: in.GetBlock(),
				logid:         in.GetHeader().GetLogid() + "_" + in.GetHeader().GetFromNode()},
		}, nil, hd)
		t.keeper.PutTask(ctx, Appending, -1)
		return nil
	}

	t.log.Trace("sync blocks in SendBlock", "time", time.Now().UnixNano(), "bcName", t.ctx.BCName, "tipID", global.F(t.ctx.State.GetLatestBlockId()))
	ctx := CreateLedgerTaskCtx(nil, []string{in.GetHeader().GetFromNode()}, hd)
	t.keeper.PutTask(ctx, Syncing, -1)
	return nil
}

// 启动矿工
func (t *miner) start() {
	// 1 强制walk到最新状态
	ledgerLastID := t.ctx.Ledger.GetMeta().GetTipBlockid()
	stateLastID := t.ctx.State.GetLatestBlockId()
	if !bytes.Equal(ledgerLastID, stateLastID) {
		t.log.Warn("ledger last blockID is not equal state last blockID", "ledgerLastID", ledgerLastID, "stateLastID", stateLastID)
		err := t.ctx.State.Walk(ledgerLastID, false)
		if err != nil {
			t.log.Error("state walk error", "error", err)
			return
		}
	}

	// 2 FAST_SYNC模式下需要回滚掉本地所有的未确认交易
	if t.ctx.EngCfg.NodeMode == config.NodeModeFastSync {
		if _, _, err := t.ctx.State.RollBackUnconfirmedTx(); err != nil {
			t.log.Warn("state RollBackUnconfirmedTx error", "error", err, "mode", "FastSync")
		}
	}

	// 3 开始同步
	t.ctx.Status = def.MinerNormal
	t.checkSyncBlock()
	for {
		select {
		case <-t.exitCh:
			break
		default:
		}

		t.logID = global.Glogid()
		t.timer = timer.NewXTimer()
		height := t.ctx.Ledger.GetMeta().GetTrunkHeight() + 1

		// 重要: 首次出块前一定要同步到最新的状态
		t.timer.Mark("MiningStart")
		t.log.Info("mining start", "logid", t.logID, "height", height, "consensus", t.ctx.Consensus.Status().GetConsensusName())

		// TODO: 账本裁剪

		master, needSync, err := t.ctx.Consensus.CompeteMaster(height)
		if err != nil {
			t.log.Error("consensus CompeteMaster error", "logid", t.logID, "error", err)
			continue
		}
		t.timer.Mark("CompeteMaster")
		t.log.Trace("CompeteMaster", "blockchain", t.ctx.BCName, "master", master, "needSync", needSync, "compete height", height)
		if master {
			if needSync {
				t.checkSyncBlock()
				t.timer.Mark("SyncBlock")
			}

			t.miner()
		}
		t.timer.Mark("MiningEnd")

		meta := t.ctx.Ledger.GetMeta()
		t.log.Info("mining end", "logid", t.logID,
			"height", meta.TrunkHeight,
			"ledgerLastID", fmt.Sprintf("%x", meta.TipBlockid),
			"stateLastID", fmt.Sprintf("%x", t.ctx.State.GetLatestBlockId()),
			"cost", t.timer.Print())
	}

	t.log.Info("miner exist")
}

// 停止矿工
func (t *miner) stop() {
	close(t.exitCh)
}

// 实现矿工行为(同步区块或者生产区块)
func (t *miner) miner() {
	t.keeper.CoreLock()
	lockHold := true
	t.timer.Mark("Lock")
	defer func() {
		if lockHold {
			t.keeper.CoreUnlock()
		}
	}()

	// 检查账本和状态机的数据一致性
	stateLastID := t.ctx.State.GetLatestBlockId()
	ledgerLastID := t.ctx.Ledger.GetMeta().GetTipBlockid()
	if !bytes.Equal(ledgerLastID, stateLastID) {
		t.log.Warn("ledger and state data inconsistent", "logid", t.logID, "ledgerLastID", global.F(ledgerLastID), "stateLastID", global.F(stateLastID))
		err := t.ctx.State.Walk(ledgerLastID, false)
		t.timer.Mark("StateWalk")
		if err != nil {
			if !t.ctx.EngCfg.FailSkip {
				t.log.Error("state walk error", "logid", t.logID, "ledgerLastID", ledgerLastID, "stateLastID", stateLastID, "error", err)
				return
			} else {
				err := t.keeper.DoTruncateTask(stateLastID)
				t.timer.Mark("Truncate")
				if err != nil {
					t.log.Error("ledger keeper truncate error", "logid", t.logID, "ledgerLastID", ledgerLastID, "stateLastID", stateLastID, "error", err)
					return
				}
			}
		}

		stateLastID = t.ctx.State.GetLatestBlockId()
		ledgerLastID = t.ctx.Ledger.GetMeta().GetTipBlockid()
	}

	batch := t.ctx.State.NewBatch()
	block, err := t.produceBlock(batch)
	t.timer.Mark("ProduceBlock")
	if err != nil {
		t.log.Error("produce block error", "logid", t.logID, "error", err)
		return
	}
	t.log.Trace("[mining] produce block succeeded", "logid", t.logID, "blockId", global.F(block.Blockid))

	// 账本确认区块
	confirmStatus := t.ctx.Ledger.ConfirmBlock(block, false)
	t.timer.Mark("ConfirmBlock")
	if confirmStatus.Succ {
		if confirmStatus.Orphan {
			t.log.Error("[mining] the mined blocked was attached to branch, no need to play", "logid", t.logID)
			return
		}
	} else {
		t.log.Error("[mining] ConfirmBlock Fail", "logid", t.logID, "confirm_status", confirmStatus)
		err := t.ctx.State.Walk(t.ctx.State.GetLatestBlockId(), false)
		if err != nil {
			t.log.Error("[mining] failed to walk when confirming block has error", "error", err)
		}
		return
	}
	t.log.Trace("[mining] ledger confirm block succeeded", "logid", t.logID)

	// 释放锁
	t.keeper.CoreUnlock()
	lockHold = false
	t.timer.Mark("Unlock")

	// TODO: 解耦
	t.ctx.State.SetBlockGenEvent()
	defer t.ctx.State.NotifyFinishBlockGen()

	// 更新状态机
	err = t.ctx.State.PlayForMiner(block.Blockid, batch)
	t.timer.Mark("PlayForMiner")
	if err != nil {
		t.log.Error("[mining] state play error", "logid", t.logID, "blockId", global.F(block.Blockid), "error", err)
		return
	}
	t.log.Trace("[mining] state play succeeded", "logid", t.logID)

	// 共识确认区块
	err = t.ctx.Consensus.ProcessConfirmBlock(block)
	if err != nil {
		t.log.Error("[mining] consensus confirm block error", "logid", t.logID, "blockId", global.F(block.Blockid), "error", err)
		return
	}
	t.timer.Mark("ProcessConfirmBlock")
	t.log.Trace("[mining] consensus confirm block succeeded", "logid", t.logID)

	// 广播区块
	go t.broadcastBlock(block)
	t.timer.Mark("BroadcastBlock")
	t.log.Trace("[mining] broadcast block succeeded", "logid", t.logID)
}

// 生产区块
func (t *miner) produceBlock(batch kvdb.Batch) (*pb.InternalBlock, error) {
	tm := time.Now()
	// 挖矿前共识的预处理
	// TODO: needTruncate, storage, err := t.ctx.Consensus.ProcessBeforeMiner(tm.UnixNano())
	// needTruncate: 回退到上一个区块
	_, storage, err := t.ctx.Consensus.ProcessBeforeMiner(tm.UnixNano())
	t.timer.Mark("ProcessBeforeMiner")
	if err != nil {
		t.log.Error("[mining] consensus ProcessBeforeMiner error", "logid", t.logID, "error", err)
		return nil, err
	}
	t.log.Trace("[mining] consensus process before miner succeeded", "logid", t.logID)

	txSize := 0
	txSizeLimit := t.ctx.State.MaxTxSizePerBlock()
	var txs []*pb.Transaction

	// 自动生成的交易
	nextHeight := t.ctx.Ledger.GetMeta().GetTrunkHeight() + 1
	vatTxs, err := t.ctx.State.GetVATList(nextHeight, -1, tm.UnixNano())
	t.timer.Mark("GetVatTX")
	if err != nil {
		t.log.Warn("[mining] fail to get vat tx list", "logid", t.logID)
		return nil, err
	}

	txs = append(txs, vatTxs...)
	for _, tx := range txs {
		txSize += proto.Size(tx)
	}
	t.log.Trace("[mining] get vat tx succeeded", "logid", t.logID, "txCount", len(vatTxs))

	// 未确认的交易
	unconfirmedTxs, err := t.ctx.State.GetUnconfirmedTx(false)
	t.timer.Mark("GetUnconfirmedTx")
	if err != nil {
		t.log.Warn("[mining] get unconfirmedTx failed", "logid", t.logID, "error", err)
		return nil, err
	}
	for _, tx := range unconfirmedTxs {
		txSize += proto.Size(tx)
		if txSize > txSizeLimit {
			t.log.Warn("already got enough tx to produce block", "logid", t.logID, "size", txSize, "limit", txSizeLimit)
			break
		}
		txs = append(txs, tx)
	}
	t.log.Trace("[mining] get unconfirmed tx succeeded", "logid", t.logID, "txCount", len(txs))

	// fake区块
	preHash := t.ctx.State.GetLatestBlockId()
	fakeBlock, err := t.ctx.Ledger.FormatFakeBlock(txs, t.ctx.Address, nextHeight, preHash, tm.UnixNano(), storage)
	if err != nil {
		t.log.Warn("[mining] format fake block error", "logid", t.logID, "error", err)
		return nil, err
	}

	// 预执行合约
	txs, _, err = t.ctx.State.TxOfRunningContractGenerate(txs, fakeBlock, batch, true)
	t.timer.Mark("PreExecContract")
	if err != nil {
		if err.Error() != common.ErrContractExecutionTimeout.Error() {
			t.log.Warn("[mining] PrePlay fake block failed", "logid", t.logID, "error", err)
			return nil, err
		}
	}
	t.log.Trace("[mining] pre exec contract succeeded", "logid", t.logID, "txCount", len(txs))

	// 矿工奖励交易
	blockAward := t.ctx.Ledger.GetGenesisBlock().CalcAward(nextHeight)
	awardTx, err := t.ctx.State.GenerateAwardTx(t.ctx.Address.Address, blockAward.String(), []byte{'1'})
	txs = append(txs, awardTx)
	t.timer.Mark("GenAwardTx")

	// 打包区块
	block, err := t.ctx.Ledger.FormatMinerBlock(txs, fakeBlock.GetFailedTxs(), t.ctx.Address, nextHeight, preHash, tm.UnixNano(), storage)
	t.timer.Mark("FormatMinerBlock")
	if err != nil {
		t.log.Warn("[mining] format block error", "logid", t.logID, "error", err)
		return nil, err
	}
	t.log.Trace("[mining] format block succeeded", "logid", t.logID, "txCount", len(txs), "failedTx", len(fakeBlock.GetFailedTxs()))

	return block, nil
}

// 检查同步区块
func (t *miner) checkSyncBlock() {
	hd := &global.XContext{Timer: global.NewXTimer()}
	t.log.Trace("sync blocks in SyncBatchBlocks", "time", time.Now().UnixNano(), "bcName", t.ctx.BCName, "tipID", global.F(t.ctx.Ledger.GetMeta().GetTipBlockid()))
	ctx := CreateLedgerTaskCtx(nil, nil, hd)
	t.keeper.PutTask(ctx, Syncing, -1)
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
