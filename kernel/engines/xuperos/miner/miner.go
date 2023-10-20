package miner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/patrickmn/go-cache"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/metrics"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/lib/utils"
)

const (
	tickOnCalcBlock           = time.Second
	syncOnStatusChangeTimeout = 1 * time.Minute

	statusFollowing = 0
	statusMining    = 1
	ConsensusSingle = "single"
	ConsensusTDPoS  = "tdpos"
)

const (
	// 缓存故障节点peerId的有效期时间
	faultPeerIdCacheExpired = 10 * time.Second

	// 故障节点出错次数阈值
	faultPeerIdCacheCount = 2

	// 缓存错误区块blockId的有效期时间
	faultBlockIdCacheExpired = 60 * time.Second

	// 故障节点与错误区块cache GC 周期（s）
	faultCacheGCInterval = 180 * time.Second
)

var (
	errCalculateBlockInterrupt = errors.New("calculate block interrupted")
)

// Miner 负责生产和同步区块
type Miner struct {
	ctx *common.ChainCtx
	log logs.Logger

	// 当前节点状态，矿工或者同步节点
	// 值得注意的是节点同一时刻只能处于一种角色，并严格执行相应的动作。
	// 即：如果是矿工则只出块，并且不会向其他节点同步新区块（pow除外），如果是非矿工则定时同步区块。
	status int

	// cache用于在同步到错误区块时缓存blockId和对应节点的peerId
	faultPeerIdCache  *cache.Cache // key:peerId, val:count(累计出现错误次数)
	faultBlockIdCache *cache.Cache // key:blockId, val:peerId

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

	obj.faultPeerIdCache = cache.New(faultPeerIdCacheExpired, faultCacheGCInterval)
	obj.faultBlockIdCache = cache.New(faultBlockIdCacheExpired, faultCacheGCInterval)

	return obj
}

// Deprecated: 使用新的同步方案，这个函数仅用来兼容
// 处理P2P网络中接收到的区块
func (m *Miner) ProcBlock(_ xctx.XContext, _ *lpb.InternalBlock) error {
	return nil
}

// Start
// 启动矿工，周期检查矿工身份
// 同一时间，矿工状态是唯一的
// 0:休眠中 1:同步区块中 2:打包区块中
func (m *Miner) Start() {
	var err error

	// 用于监测退出
	m.exitWG.Add(1)
	defer m.exitWG.Done()

	// 节点初始状态为同步节点
	m.status = statusFollowing

	// 开启挖矿前先同步区块
	ctx := &xctx.BaseCtx{
		XLog:  m.log,
		Timer: timer.NewXTimer(),
	}
	_ = m.syncWithNeighbors(ctx)

	// 启动矿工循环
	for !m.IsExit() {
		err = m.step()

		// 如果出错，休眠1s后重试，防止cpu被打满
		if err != nil {
			m.log.Warn("miner run occurred error,sleep 1s try", "err", err)
			time.Sleep(time.Second)
		}
	}
}

// Stop 停止矿工
func (m *Miner) Stop() {
	m.isExit = true
	m.exitWG.Wait()
}

func (m *Miner) IsExit() bool {
	return m.isExit
}

func traceMiner() func(string) {
	last := time.Now()
	return func(action string) {
		metrics.CallMethodHistogram.WithLabelValues("miner", action).Observe(time.Since(last).Seconds())
		last = time.Now()
	}
}

// step 用于推动节点循环进行一次动作，可以是一次出块动作(矿工角色)，也可以是一次区块同步（非矿工）
// 在此期间可能会发生节点角色变更。
func (m *Miner) step() error {
	ledgerTipId := m.ctx.Ledger.GetMeta().TipBlockid
	ledgerTipHeight := m.ctx.Ledger.GetMeta().TrunkHeight
	stateTipId := m.ctx.State.GetLatestBlockid()

	log, _ := logs.NewLogger("", "miner")
	ctx := &xctx.BaseCtx{
		XLog:  log,
		Timer: timer.NewXTimer(),
	}

	// 账本和状态机最新区块id不一致，需要进行一次同步
	if !bytes.Equal(ledgerTipId, stateTipId) {
		err := m.ctx.State.Walk(ledgerTipId, false)
		if err != nil {
			return err
		}
	}

	trace := traceMiner()

	ctx.GetLog().Trace("miner step", "ledgerTipHeight", ledgerTipHeight, "ledgerTipId",
		utils.F(ledgerTipId), "stateTipId", utils.F(stateTipId))

	// 如果上次角色是非矿工，则尝试同步网络最新区块
	// 注意：这里出现错误也要继续执行，防止恶意节点错误出块导致流程无法继续执行
	if m.status == statusFollowing {
		err := m.syncWithValidators(ctx, syncOnStatusChangeTimeout)
		ctx.GetLog().Trace("miner syncWithValidators before CompeteMaster", "originTipHeight", ledgerTipHeight,
			"currentLedgerHeight", m.ctx.Ledger.GetMeta().TrunkHeight, "err", err)
		trace("syncUpValidators")
	}

	// 通过共识检查矿工身份
	isMiner, isSync, err := m.ctx.Consensus.CompeteMaster(ledgerTipHeight + 1)
	trace("competeMaster")
	ctx.GetLog().Trace("compete master result", "height", ledgerTipHeight+1, "isMiner", isMiner, "isSync", isSync, "err", err)
	if err != nil {
		return err
	}

	// 如果是矿工，出块
	if isMiner {
		if m.status == statusFollowing || isSync {
			ctx.GetLog().Info("miner change follow=>miner",
				"miner", m.ctx.Address.Address,
				"height", m.ctx.Ledger.GetMeta().GetTrunkHeight(),
			)

			// 在由非矿工向矿工切换的这次"边沿触发"，主动向所有的验证集合的最长链进行一次区块同步
			err = m.syncWithValidators(ctx, syncOnStatusChangeTimeout)
			if err != nil {
				ctx.GetLog().Error("miner change follow=>miner syncWithValidators failed", "err", err)
				return err
			}

			// 由于同步了最长链，所以这里需要检查链是否增长
			// 由于pos和poa类共识依赖账本高度来判断状态，如果链发生变化则表明CompeteMaster的结果需要重新根据当前最新高度计算
			if ledgerTipHeight != m.ctx.Ledger.GetMeta().TrunkHeight {
				ctx.GetLog().Trace("miner change follow=>miner", "originTipHeight", ledgerTipHeight, "currentLedgerHeight",
					m.ctx.Ledger.GetMeta().TrunkHeight, "isMiner", isMiner, "isSync", isSync)
				return nil
			}
			trace("syncUpValidators")
		}
		m.status = statusMining
		if m.ctx.EngCtx.EngCfg.DisableEmptyBlocks && !m.ctx.State.HasUnconfirmTx() {
			consensusStatus, err := m.ctx.Consensus.GetConsensusStatus()
			if err != nil {
				return err
			}
			// 目前不出空块配置只在 single 共识下生效
			if consensusStatus.GetConsensusName() == ConsensusTDPoS {
				return nil
			}
		}
		// 开始挖矿
		err = m.mining(ctx)
		if err != nil {
			return err
		}
		trace("mining")
		return nil
	}

	// 非miner，向邻居同步区块
	if m.status == statusMining {
		ctx.GetLog().Info("miner change miner=>following",
			"miner", m.ctx.Address.Address,
			"height", m.ctx.Ledger.GetMeta().GetTrunkHeight(),
		)
	}
	m.status = statusFollowing
	err = m.syncWithNeighbors(ctx)
	if err != nil {
		return err
	}
	trace("syncPeers")
	return nil
}

// mining 挖矿生产区块
func (m *Miner) mining(ctx xctx.XContext) error {
	ctx.GetLog().Debug("mining start.")

	// 1.共识挖矿前处理
	height := m.ctx.Ledger.GetMeta().TrunkHeight + 1
	now := time.Now()
	truncateTarget, extData, err := m.ctx.Consensus.ProcessBeforeMiner(height, now.UnixNano())
	ctx.GetTimer().Mark("ProcessBeforeMiner")
	if err != nil {
		ctx.GetLog().Warn("consensus process before miner failed", "err", err)
		return fmt.Errorf("consensus process before miner failed")
	}
	ctx.GetLog().Debug("consensus before miner succ", "truncateTarget", truncateTarget, "extData", string(extData))
	if truncateTarget != nil {
		// 裁剪掉账本目标区块，裁掉的交易判断冲突重新回放，裁剪完后继续出块操作
		if err := m.truncateForMiner(ctx, truncateTarget); err != nil {
			return err
		}
		// 重置高度
		height = m.ctx.Ledger.GetMeta().TrunkHeight + 1
		ctx.GetLog().Debug("truncateTarget result", "newHeight", height)
	}

	// 2.打包区块
	beginTime := time.Now()
	block, err := m.packBlock(ctx, height, now, extData)
	ctx.GetTimer().Mark("PackBlock")
	metrics.CallMethodHistogram.WithLabelValues("miner", "PackBlock").Observe(time.Since(beginTime).Seconds())
	if err != nil {
		ctx.GetLog().Warn("pack block error", "err", err)
		return err
	}
	ctx.GetLog().Debug("pack block succ", "height", height, "blockId", utils.F(block.GetBlockid()))

	// 3. 针对一些需要patch区块的共识
	origBlkId := block.Blockid
	blkAgent := state.NewBlockAgent(block)
	err = m.calculateBlock(blkAgent)
	ctx.GetTimer().Mark("CalculateBlock")
	if err == errCalculateBlockInterrupt {
		return nil
	}
	if err != nil {
		ctx.GetLog().Warn("consensus calculate block failed", "err", err,
			"blockId", utils.F(block.Blockid))
		return fmt.Errorf("consensus calculate block failed")
	}
	ctx.GetLog().Trace("start confirm block for miner", "originalBlockId", utils.F(origBlkId),
		"newBlockId", utils.F(block.Blockid))

	// 4.账本&状态机&共识确认新区块
	err = m.confirmBlockForMiner(ctx, block)
	if err != nil {
		ctx.GetLog().Warn("confirm block for miner failed", "err", err,
			"blockId", utils.F(block.GetBlockid()))
		return err
	}

	// 5.可插拔共识，根据区块高度确认是否需要切换升级共识实例
	err = m.ctx.Consensus.SwitchConsensus(block.Height)
	if err != nil {
		ctx.GetLog().Warn("SwitchConsensus failed", "bcname", m.ctx.BCName,
			"err", err, "blockId", utils.F(block.GetBlockid()))
		// todo 这里暂时不返回错误
	}

	ctx.GetLog().Info("finish new block generation", "blockId", utils.F(block.GetBlockid()),
		"height", height, "txCount", block.TxCount, "size", proto.Size(block), "costs", ctx.GetTimer().Print())
	return nil
}

// 裁剪掉账本最新的区块
func (m *Miner) truncateForMiner(ctx xctx.XContext, target []byte) error {
	_, err := m.ctx.Ledger.QueryBlockHeader(target)
	if err != nil {
		ctx.GetLog().Warn("truncate failed because query target error", "err", err)
		return err
	}

	// 状态机回滚到目标状态
	err = m.ctx.State.Walk(target, false)
	if err != nil {
		ctx.GetLog().Warn("truncate failed because state walk error", "ledgerTipId", utils.F(m.ctx.Ledger.GetMeta().TipBlockid),
			"walkTargetBlockId", utils.F(target))
		return err
	}

	// 账本裁剪到这个区块
	err = m.ctx.Ledger.Truncate(target)
	if err != nil {
		ctx.GetLog().Warn("truncate failed because ledger truncate error", "err", err)
		return err
	}

	return nil
}

func (m *Miner) packBlock(ctx xctx.XContext, height int64,
	now time.Time, consData []byte) (*lpb.InternalBlock, error) {
	// 区块大小限制
	sizeLimit, err := m.ctx.State.MaxTxSizePerBlock()
	if err != nil {
		return nil, err
	}
	ctx.GetLog().Debug("pack block get max size succ", "sizeLimit", sizeLimit)

	// 1.生成timer交易
	autoTx, err := m.getTimerTx(height)
	if err != nil {
		return nil, err
	}
	if len(autoTx.TxOutputsExt) > 0 {
		sizeLimit -= proto.Size(autoTx)
	}

	ctx.GetLog().Debug("pack block get timer tx succ", "auto tx", autoTx)

	// 2.选择本次要打包的tx
	generalTxList, err := m.getUnconfirmedTx(sizeLimit)
	if err != nil {
		return nil, err
	}
	ctx.GetLog().Debug("pack block get general tx succ", "txCount", len(generalTxList))

	// 3.获取矿工奖励交易
	awardTx, err := m.getAwardTx(height)
	if err != nil {
		return nil, err
	}
	ctx.GetLog().Debug("pack block get award tx succ", "txid", utils.F(awardTx.GetTxid()))

	txList := make([]*lpb.Transaction, 0, len(generalTxList)+1+1)
	// 先coinbase tx
	txList = append(txList, awardTx)
	// 再auto tx
	if len(autoTx.TxOutputsExt) > 0 {
		txList = append(txList, autoTx)
	}
	// 最后普通tx
	if len(generalTxList) > 0 {
		txList = append(txList, generalTxList...)
	}

	// 4.打包区块
	consInfo, err := m.convertConsData(consData)
	if err != nil {
		ctx.GetLog().Warn("convert consensus data failed", "err", err, "consData", string(consData))
		return nil, fmt.Errorf("convert consensus data failed")
	}
	block, err := m.ctx.Ledger.FormatMinerBlock(txList, []byte(m.ctx.Address.Address),
		m.ctx.Address.PrivateKey, now.UnixNano(), consInfo.CurTerm, consInfo.CurBlockNum,
		m.ctx.State.GetLatestBlockid(), consInfo.TargetBits, m.ctx.State.GetTotal(),
		consInfo.Justify, nil, height)
	if err != nil {
		ctx.GetLog().Warn("format block error", "err", err)
		return nil, err
	}

	return block, nil
}

func (m *Miner) convertConsData(data []byte) (*state.ConsensusStorage, error) {
	var consInfo state.ConsensusStorage
	if len(data) < 1 {
		return &consInfo, nil
	}

	err := json.Unmarshal(data, &consInfo)
	if err != nil {
		return nil, err
	}

	return &consInfo, nil
}

func (m *Miner) getTimerTx(height int64) (*lpb.Transaction, error) {
	autoTx, err := m.ctx.State.GetTimerTx(height)
	if err != nil {
		m.log.Error("Get timer tx error", "error", err)
		return nil, common.ErrGenerateTimerTxFailed
	}

	return autoTx, nil
}

func (m *Miner) getUnconfirmedTx(sizeLimit int) ([]*lpb.Transaction, error) {
	return m.ctx.State.GetUnconfirmedTx(false, sizeLimit)
}

func (m *Miner) getAwardTx(height int64) (*lpb.Transaction, error) {
	amount := m.ctx.Ledger.GenesisBlock.CalcAward(height)
	if amount.Cmp(big.NewInt(0)) < 0 {
		return nil, errors.New("amount in transaction can not be negative number")
	}

	awardTx, err := tx.GenerateAwardTx(m.ctx.Address.Address, amount.String(), []byte("award"))
	if err != nil {
		return nil, err
	}

	return awardTx, nil
}

// pow类共识的CompleteMaster结果并不能反映当前的矿工身份，每个节点都是潜在的矿工，
// 因此需要在calculateBlock这个阻塞点上进行同步区块的处理
func (m *Miner) calculateBlock(block *state.BlockAgent) error {
	ticker := time.NewTicker(tickOnCalcBlock)
	defer ticker.Stop()

	calcDone := make(chan error, 1)
	go func() {
		err := m.ctx.Consensus.CalculateBlock(block)
		calcDone <- err
	}()

	for !m.IsExit() {
		select {
		case err := <-calcDone:
			m.log.Info("calc block done", "error", err, "height", block.GetHeight(),
				"blockid", utils.F(block.GetBlockid()))
			return err
		case <-ticker.C:
			ctx := &xctx.BaseCtx{
				XLog:  m.log,
				Timer: timer.NewXTimer(),
			}
			err := m.syncWithNeighbors(ctx)
			if err != nil {
				m.log.Warn("syncBlockWithPeers error", "error", err)
			}
			if m.ctx.Ledger.GetMeta().TrunkHeight >= block.GetHeight() {
				// TODO: stop CalculateBlock
				m.log.Info("CalculateBlock interrupted", "trunk-height", m.ctx.Ledger.GetMeta().TrunkHeight,
					"block-height", block.GetHeight())
				return errCalculateBlockInterrupt
			}
		}
	}
	if m.IsExit() {
		return errors.New("miner already exit")
	}
	return nil
}

func (m *Miner) confirmBlockForMiner(ctx xctx.XContext, block *lpb.InternalBlock) error {
	tip := m.ctx.Ledger.GetMeta().TipBlockid
	if !bytes.Equal(block.PreHash, tip) {
		ctx.GetLog().Warn("confirmBlockForMiner error", "tip", utils.F(tip),
			"prehash", utils.F(block.PreHash))
		return errors.New("confirm block prehash mismatch")
	}

	// 账本确认区块
	confirmStatus := m.ctx.Ledger.ConfirmBlock(block, false)
	ctx.GetTimer().Mark("ConfirmBlock")
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
	err := m.ctx.State.PlayForMiner(block.Blockid)
	ctx.GetTimer().Mark("PlayForMiner")
	if err != nil {
		ctx.GetLog().Warn("state play error ", "error", err, "blockId", utils.F(block.Blockid))
	}

	// 共识确认区块
	blkAgent := state.NewBlockAgent(block)
	err = m.ctx.Consensus.ProcessConfirmBlock(blkAgent)
	ctx.GetTimer().Mark("ProcessConfirmBlock")
	if err != nil {
		ctx.GetLog().Warn("consensus confirm block error", "err", err,
			"blockId", utils.F(block.Blockid))
		return err
	}

	ctx.GetLog().Trace("confirm block for miner succ", "blockId", utils.F(block.Blockid))
	return nil
}
