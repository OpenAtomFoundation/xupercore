package miner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xupercore/bcs/consensus/tdpos"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/xpb"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/metrics"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

const (
	batchBlockNumber    = 2
	maxBatchBlockNumber = 10

	peersKey = "peers"
)

var (
	ErrHashMissMatch = errors.New("hash miss match")
	ErrNoNewBlock    = errors.New("no new block found")
)

func traceSync() func(string) {
	last := time.Now()
	return func(action string) {
		metrics.CallMethodHistogram.WithLabelValues("sync", action).Observe(time.Since(last).Seconds())
		last = time.Now()
	}
}

// 获取验证人（潜在的矿工节点）列表，除了自己
func (t *Miner) getValidators(excludeAddr string) ([]string, error) {
	status, err := t.ctx.Consensus.GetConsensusStatus()
	if err != nil {
		return nil, err
	}
	buf := status.GetCurrentValidatorsInfo()
	var info tdpos.ValidatorsInfo
	err = json.Unmarshal(buf, &info)
	if err != nil {
		return nil, err
	}
	if len(info.Validators) == 0 {
		return nil, errors.New("empty miners")
	}
	validators := info.Validators
	if excludeAddr != "" {
		for i, addr := range validators {
			if addr == excludeAddr {
				validators = append(validators[:i], validators[i+1:]...)
				break
			}
		}
	}
	return validators, nil
}

// getMaxBlockHeight 从验证人列表里面获取当前最大的区块高度以及地址
func (t *Miner) getMaxBlockHeight(ctx xctx.XContext) (string, int64, []byte, error) {
	validators, err := t.getValidators(t.ctx.Address.Address)
	if err != nil {
		return "", 0, nil, err
	}

	if len(validators) == 0 {
		return "", 0, nil, nil
	}
	opt := []p2p.MessageOption{
		p2p.WithBCName(t.ctx.BCName),
		// p2p.WithLogId(ctx.GetLog().GetLogId()),
	}
	msg := p2p.NewMessage(protos.XuperMessage_GET_BLOCKCHAINSTATUS, nil, opt...)
	ctx.GetLog().Debug("getMaxBlockHeight", "validators", validators)
	responses, err := t.ctx.EngCtx.Net.SendMessageWithResponse(t.ctx, msg, p2p.WithAccounts(validators))
	if err != nil {
		ctx.GetLog().Warn("get block chain status error", "err", err)
		return "", 0, nil, err
	}

	maxHeight := int64(0)
	peer := ""
	blockId := []byte("")
	for _, response := range responses {
		var status xpb.ChainStatus
		err = p2p.Unmarshal(response, &status)
		if err != nil {
			ctx.GetLog().Warn("unmarshal block chain status error", "err", err)
			continue
		}
		if status.LedgerMeta.TrunkHeight > maxHeight {
			// 判断该TipBlockid是否曾经验证出错过
			if curPeerId, has := t.faultBlockIdCache.Get(string(status.LedgerMeta.TipBlockid)); has {
				ctx.GetLog().Debug("faultBlockIdCache blockId hit", "TipBlockid", status.LedgerMeta.TipBlockid, "curPeerId", curPeerId)
				curPeerIdStr, okConvert := curPeerId.(string)
				if !okConvert {
					ctx.GetLog().Warn("faultBlockIdCache convert peerId failed", "TipBlockid", status.LedgerMeta.TipBlockid, "curPeerId", curPeerId)
					continue
				}
				// peerId记录不一致则更新peerid信息
				if curPeerIdStr != response.Header.From {
					t.faultBlockIdCache.Set(string(status.LedgerMeta.TipBlockid), response.Header.From, faultBlockIdCacheExpired)
				}
				// 增加peerId记录错误次数
				count, errInc := t.faultPeerIdCache.IncrementInt32(response.Header.From, int32(1))
				if errInc != nil {
					count = 1
					t.faultPeerIdCache.Set(response.Header.From, count, faultPeerIdCacheExpired)
				}
				ctx.GetLog().Debug("faultPeerIdCache Increment count", "curPeerIdStr", curPeerIdStr, "count", count, "errInc", errInc)
				continue
			} else {
				// 检查peerId是否超过故障次数
				countInterface, hasPeer := t.faultPeerIdCache.Get(response.Header.From)
				if hasPeer {
					count, okConvert := countInterface.(int32)
					if !okConvert {
						ctx.GetLog().Warn("faultPeerIdCache convert countInterface failed", "TipBlockid", status.LedgerMeta.TipBlockid,
							"from", response.Header.From, "countInterface", countInterface)
					}

					ctx.GetLog().Debug("faultPeerIdCache peerId hit and count >= 2", "count", count)
					// 出错达到标准阈值，不采纳该节点的信息
					if count >= faultPeerIdCacheCount {
						ctx.GetLog().Info("faultPeerIdCache peerId hit and count >= 2", "TipBlockid", status.LedgerMeta.TipBlockid,
							"TrunkHeight", status.LedgerMeta.TrunkHeight, "from", response.Header.From, "count", count)
						continue
					}
				}
			}
			maxHeight = status.LedgerMeta.TrunkHeight
			peer = response.Header.From
			blockId = status.LedgerMeta.TipBlockid
		}
	}
	return peer, maxHeight, blockId, nil
}

// syncWithValidators 向拥有最长链的验证人节点进行区块同步，直到区块高度完全一致，timeout用于设置同步超时时间，超时之后无论是否同步完毕都停止。
func (t *Miner) syncWithValidators(ctx xctx.XContext, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		size, err := t.syncWithLongestChain(ctx)
		if err != nil {
			ctx.GetLog().Warn("syncWithLongestChain error", "error", err)
			continue
		}
		if size == 0 {
			return nil
		}
	}
	return errors.New("syncUpValidators timeout")
}

// syncWithLongestChain 向验证人结合进行一次区块同步，返回同步的区块个数
func (t *Miner) syncWithLongestChain(ctx xctx.XContext) (int, error) {
	currentHeight := t.ctx.Ledger.GetMeta().TrunkHeight
	peer, maxHeight, blockId, err := t.getMaxBlockHeight(ctx)
	if err != nil {
		ctx.GetLog().Error("getMaxBlockHeight error", "error", err)
		return 0, err
	}
	if maxHeight == 0 {
		return 0, nil
	}
	if maxHeight <= currentHeight {
		return 0, nil
	}
	ctx = xctx.WithNewContext(ctx, context.WithValue(ctx, peersKey, []string{peer}))
	height := currentHeight + 1
	size := maxHeight - currentHeight
	ctx.GetLog().Info("syncWithLongestChain", "peer", peer, "beginHeight", height, "size", size)
	realSize, err := t.syncBlockWithHeight(ctx, height, int(size))
	if err != nil {
		// 同步出错，记录blockId
		t.faultBlockIdCache.Set(string(blockId), peer, faultBlockIdCacheExpired)
		// 同步出错，记录对应的peerId，增加错误计数
		count, errInc := t.faultPeerIdCache.IncrementInt32(peer, int32(1))
		if errInc != nil {
			count = 1
			t.faultPeerIdCache.Set(peer, count, faultPeerIdCacheExpired)
		}
		ctx.GetLog().Warn("syncWithLongestChain syncBlockWithHeight failed", "peer", peer,
			"beginHeight", height, "size", size, "blockId", blockId, "count", count,
			"maxHeight", maxHeight, "currentHeight", currentHeight, "errInc", errInc, "err", err)

		return 0, err
	}
	return realSize, nil
}

// syncWithNeighbors 向p2p邻居节点进行区块同步
func (t *Miner) syncWithNeighbors(ctx xctx.XContext) error {
	for {
		currentHeight := t.ctx.Ledger.GetMeta().TrunkHeight
		height := currentHeight + 1
		size, err := t.syncBlockWithHeight(ctx, height, batchBlockNumber)
		if err != nil {
			return err
		}
		if size <= 0 {
			break
		}
	}
	return nil
}

func (t *Miner) syncBlockWithHeight(ctx xctx.XContext, height int64, size int) (int, error) {
	ctx.GetLog().Debug("getBlocksByHeight", "height", height, "size", size)
	trace := traceSync()
	blocks, err := t.getBlocksByHeight(ctx, height, size)
	if err == ErrNoNewBlock {
		return 0, nil
	}

	if err != nil {
		ctx.GetLog().Warn("getBlocksByHeight error", "height", height, "error", err)
		return 0, err
	}
	trace("getBlockByHeight")
	ctx.GetLog().Info("getBlocksByHeight return blocks", "height", height, "size", size, "realSize", len(blocks))
	err = t.batchConfirmBlocks(ctx, blocks)
	if err == ErrHashMissMatch {
		// 发生了分叉，处理分叉
		ctx.GetLog().Error("sync peers with fork")
		err = t.handleFork(ctx)
		if err != nil {
			ctx.GetLog().Error("handle fork error", "error", err)
			return 0, err
		}
	}
	if err != nil {
		ctx.GetLog().Warn("batchConfirmBlocks error", "error", err)
		return 0, err
	}
	trace("batchConfirmBlock")
	return len(blocks), nil
}

// getBlocksByHeight 获取指定的区块高度(height)，个数为size的区块头，如果ctx里面有peersKey，则向指定的peer列表发送消息
func (t *Miner) getBlocksByHeight(ctx xctx.XContext, height int64, size int) ([]*lpb.InternalBlock, error) {
	if size > maxBatchBlockNumber {
		size = maxBatchBlockNumber
	}

	input := &xpb.GetBlockHeaderRequest{
		Bcname: t.ctx.BCName,
		Height: height,
		Size:   int64(size),
	}

	trace := traceSync()
	opts := []p2p.OptionFunc{
		// p2p.WithPercent(0.1),
	}
	if ctx.Value(peersKey) != nil {
		ctx.GetLog().Debug("sync with peer address", "address", ctx.Value(peersKey))
		opts = append(opts, p2p.WithPeerIDs(ctx.Value(peersKey).([]string)))
	} else {
		switch t.ctx.EngCtx.EngCfg.SyncBlockFilterMode {
		case common.SyncWithNearestBucket:
			opts = append(opts, p2p.WithFilter([]p2p.FilterStrategy{p2p.NearestBucketStrategy}))
		case common.SyncWithFactorBucket:
			opts = append(opts, p2p.WithFilter([]p2p.FilterStrategy{p2p.BucketsWithFactorStrategy}),
				p2p.WithFactor(t.ctx.EngCtx.EngCfg.SyncFactorForFactorBucketMode))
		default:
			opts = append(opts, p2p.WithFilter([]p2p.FilterStrategy{p2p.NearestBucketStrategy}))
		}
	}
	msg := p2p.NewMessage(protos.XuperMessage_GET_BLOCK_HEADERS, input, p2p.WithBCName(t.ctx.BCName))
	responses, err := t.ctx.EngCtx.Net.SendMessageWithResponse(ctx, msg, opts...)
	if err != nil {
		ctx.GetLog().Warn("p2p get block header error", "err", err)
		return nil, err
	}
	trace("getBlockHeader")
	blocks := quorumBlocks(responses, size)
	for _, blk := range blocks {
		blkid, _ := ledger.MakeBlockID(blk)
		if !bytes.Equal(blkid, blk.GetBlockid()) {
			ctx.GetLog().Warn("download bad block id", "height", blk.GetHeight(),
				"got", utils.F(blk.GetBlockid()), "expect", utils.F(blkid))
			return nil, errors.New("bad block id")
		}
	}

	if len(blocks) == 0 {
		return nil, ErrNoNewBlock
	}
	for _, block := range blocks {
		err = t.fillBlockTxs(ctx, block)
		if err != nil {
			return nil, err
		}
	}
	trace("fillBlockTxs")
	return blocks, nil
}

func (t *Miner) fillBlockTxs(ctx xctx.XContext, block *lpb.InternalBlock) error {
	trace := traceSync()
	txids := block.GetMerkleTree()[:block.GetTxCount()]

	blockTxs := make([]*lpb.Transaction, len(txids))
	if len(block.Transactions) > 0 && block.Transactions[0] != nil {
		// 取coinbase交易
		blockTxs[0] = block.Transactions[0]
	}

	var missingTxIdx []int32
	for idx, txid := range txids {
		if blockTxs[idx] != nil {
			continue
		}
		tx, ok := t.ctx.State.GetUnconfirmedTxFromId(txid)
		if !ok {
			missingTxIdx = append(missingTxIdx, int32(idx))
			continue
		}
		blockTxs[idx] = tx
	}
	trace("fillUnconfirmed")
	ctx.GetLog().Info("fillBlockTxs", "total", int(block.GetTxCount()), "missing", len(missingTxIdx))
	missingTxs, err := t.downloadMissingTxs(ctx, block.Blockid, missingTxIdx)
	if err != nil {
		return err
	}
	for i, idx := range missingTxIdx {
		if !bytes.Equal(txids[idx], missingTxs[i].Txid) {
			return fmt.Errorf("download tx for %x error, got:%x", txids[idx], missingTxs[i].Txid)
		}
		blockTxs[idx] = missingTxs[i]
	}
	block.Transactions = blockTxs
	trace("fillMissing")
	return nil
}

func (t *Miner) downloadMissingTxs(ctx xctx.XContext, blockid []byte, txidx []int32) ([]*lpb.Transaction, error) {
	if len(txidx) == 0 {
		return nil, nil
	}
	input := &xpb.GetBlockTxsRequest{
		Bcname:  t.ctx.BCName,
		Blockid: blockid,
		Txs:     txidx,
	}

	opts := []p2p.OptionFunc{
		// p2p.WithPercent(0.1),
	}
	if ctx.Value(peersKey) != nil {
		ctx.GetLog().Info("sync with peer address", "address", ctx.Value(peersKey))
		opts = append(opts, p2p.WithPeerIDs(ctx.Value(peersKey).([]string)))
	} else {
		opts = append(opts, p2p.WithFilter([]p2p.FilterStrategy{p2p.NearestBucketStrategy}))
	}

	msg := p2p.NewMessage(protos.XuperMessage_GET_BLOCK_TXS, input, p2p.WithBCName(t.ctx.BCName))
	responses, err := t.ctx.EngCtx.Net.SendMessageWithResponse(ctx, msg, opts...)
	if err != nil {
		ctx.GetLog().Warn("confirm block chain status error", "err", err)
		return nil, err
	}

	var txs []*lpb.Transaction
	for _, response := range responses {
		if response.GetHeader().GetErrorType() != protos.XuperMessage_SUCCESS {
			continue
		}
		var block xpb.GetBlockTxsResponse
		err = p2p.Unmarshal(response, &block)
		if err != nil {
			ctx.GetLog().Warn("unmarshal block error", "err", err)
			continue
		}

		if block.Txs == nil {
			continue
		}
		txs = block.Txs
		break
	}
	if len(txs) == 0 {
		return nil, errors.New("get block txs no response")
	}
	for _, tx := range txs {
		txid, _ := txhash.MakeTransactionID(tx)
		if !bytes.Equal(txid, tx.GetTxid()) {
			ctx.GetLog().Warn("download bad tx id", "expect", utils.F(txid), "got", tx.GetTxid())
			return nil, errors.New("bad tx id")
		}
	}
	return txs, nil
}

// 追加区块到账本中
func (t *Miner) batchConfirmBlocks(ctx xctx.XContext, blocks []*lpb.InternalBlock) error {
	if len(blocks) < 1 {
		return nil
	}

	for _, block := range blocks {
		trace := traceSync()
		timer := timer.NewXTimer()
		valid, err := t.ctx.Ledger.VerifyBlock(block, ctx.GetLog().GetLogId())
		if !valid {
			ctx.GetLog().Warn("the verification of block failed.",
				"blockId", utils.F(block.Blockid))
			return fmt.Errorf("the verification of block failed from ledger")
		}
		timer.Mark("VerifyBlock")
		trace("VerifyBlock")

		if !bytes.Equal(t.ctx.Ledger.GetMeta().TipBlockid, block.PreHash) {
			ctx.GetLog().Error("block.prehash != chunkBlockId",
				"height", block.Height,
				"chunk", utils.F(t.ctx.Ledger.GetMeta().TipBlockid),
				"block", utils.F(block.Blockid),
				"block.prehash", utils.F(block.PreHash),
			)
			return ErrHashMissMatch
		}

		blockAgent := state.NewBlockAgent(block)
		isMatch, err := t.ctx.Consensus.CheckMinerMatch(ctx, blockAgent)
		if !isMatch {
			ctx.GetLog().Warn("consensus check miner match failed",
				"blockId", utils.F(block.Blockid), "err", err)
			return errors.New("consensus check miner match failed")
		}
		timer.Mark("CheckMinerMatch")
		trace("CheckMinerMatch")

		status := t.ctx.Ledger.ConfirmBlock(block, false)
		if !status.Succ {
			ctx.GetLog().Warn("ledger confirm block failed",
				"blockId", utils.F(block.Blockid), "err", status.Error)
			return errors.New("ledger confirm block failed")
		}
		timer.Mark("ConfirmBlock")
		trace("ConfirmBlock")

		// 状态机确认区块
		err = t.ctx.State.PlayAndRepost(block.Blockid, false, false)
		if err != nil {
			ctx.GetLog().Warn("state play error", "error", err, "height", block.Height, "blockId", utils.F(block.Blockid))
		}
		trace("PlayAndRepost")
		timer.Mark("PlayAndRepost")

		err = t.ctx.Consensus.ProcessConfirmBlock(blockAgent)
		if err != nil {
			ctx.GetLog().Warn("consensus process confirm block failed",
				"blockId", utils.F(block.Blockid), "err", err)
			return errors.New("consensus process confirm block failed")
		}
		trace("ConProcessConfirmBlock")
		err = t.ctx.Consensus.SwitchConsensus(block.Height)
		if err != nil {
			ctx.GetLog().Warn("SwitchConsensus failed", "bcname", t.ctx.BCName,
				"err", err, "blockId", utils.F(block.GetBlockid()))
			// todo 这里暂时不返回错误
		}
		ctx.GetLog().Info("confirm block finish", "blockId", utils.F(block.Blockid), "height", block.Height, "txCount", block.TxCount, "size", proto.Size(block), "costs", timer.Print())
	}

	ctx.GetLog().Trace("batch confirm block to ledger succ", "blockCount", len(blocks))
	return nil
}

type blockCount struct {
	Block *lpb.InternalBlock
	Count int
}

type quorumBlockCounter struct {
	count []blockCount
}

func (q *quorumBlockCounter) updateBlock(blk *lpb.InternalBlock) {
	for i := range q.count {
		if bytes.Equal(q.count[i].Block.Blockid, blk.Blockid) {
			q.count[i].Count++
			return
		}
	}
	q.count = append(q.count, blockCount{
		Block: blk,
		Count: 1,
	})
}

func (q *quorumBlockCounter) quorumBlock() *lpb.InternalBlock {
	if len(q.count) == 0 {
		return nil
	}
	sort.Slice(q.count, func(i, j int) bool {
		return q.count[i].Count > q.count[j].Count
	})
	return q.count[0].Block
}

// quorumBlocks 根据节点们返回的p2p区块头消息列表算出大多数都认可的区块头列表，如果没有区块合适的区块信息，则返回nil
func quorumBlocks(responses []*protos.XuperMessage, blockAmount int) []*lpb.InternalBlock {
	var peerBlocks [][]*lpb.InternalBlock
	for _, response := range responses {
		if response.GetHeader().GetErrorType() != protos.XuperMessage_SUCCESS {
			continue
		}
		var block xpb.GetBlockHeaderResponse
		err := p2p.Unmarshal(response, &block)
		if err != nil {
			continue
		}
		if len(block.Blocks) == 0 {
			continue
		}
		peerBlocks = append(peerBlocks, block.Blocks)
	}

	if len(peerBlocks) == 0 {
		return nil
	}

	var retBlocks []*lpb.InternalBlock
	for i := 0; i < blockAmount; i++ {
		var counter quorumBlockCounter
		for _, blocks := range peerBlocks {
			if blocks == nil {
				continue
			}
			if i >= len(blocks) {
				continue
			}
			blk := blocks[i]
			counter.updateBlock(blk)
		}

		qblk := counter.quorumBlock()
		// 没有更多的block
		if qblk == nil {
			break
		}
		retBlocks = append(retBlocks, qblk)
	}
	return retBlocks
}

func (t *Miner) findForkPoint(ctx xctx.XContext) (*lpb.InternalBlock, error) {
	currentHeight := t.ctx.Ledger.GetMeta().TrunkHeight
	ledger := t.ctx.Ledger

	opts := []p2p.OptionFunc{
		// p2p.WithPercent(0.1),
	}
	if ctx.Value(peersKey) != nil {
		ctx.GetLog().Info("sync with peer address", "address", ctx.Value(peersKey))
		opts = append(opts, p2p.WithPeerIDs(ctx.Value(peersKey).([]string)))
	} else {
		opts = append(opts, p2p.WithFilter([]p2p.FilterStrategy{p2p.NearestBucketStrategy}))
	}

	height := currentHeight
	for {
		if height == 0 {
			ctx.GetLog().Error("the genesis block is different",
				"genesisBlockId", utils.F(ledger.GetMeta().RootBlockid))
			return nil, errors.New("block diff at genesis block")
		}
		height -= 1

		currentBlk, err := ledger.QueryBlockHeaderByHeight(height)
		if err != nil {
			return nil, err
		}
		input := &xpb.GetBlockHeaderRequest{
			Bcname: t.ctx.BCName,
			Height: height,
			Size:   1,
		}

		msg := p2p.NewMessage(protos.XuperMessage_GET_BLOCK_HEADERS, input, p2p.WithBCName(t.ctx.BCName))
		responses, err := t.ctx.EngCtx.Net.SendMessageWithResponse(ctx, msg, opts...)
		if err != nil {
			ctx.GetLog().Warn("query block header error", "err", err)
			return nil, err
		}
		blks := quorumBlocks(responses, 1)
		if len(blks) == 0 {
			ctx.GetLog().Warn("query block header with no response")
			return nil, errors.New("query block header with no response")
		}
		if bytes.Equal(currentBlk.Blockid, blks[0].Blockid) {
			return currentBlk, nil
		}
		ctx.GetLog().Info("find fork point not equal", "height", height,
			"our", utils.F(currentBlk.Blockid), "theirs", utils.F(blks[0].Blockid))
	}
}

func (m *Miner) handleFork(ctx xctx.XContext) error {
	forkPoint, err := m.findForkPoint(ctx)
	if err != nil {
		ctx.GetLog().Error("findForkPoint error", err)
		return err
	}
	ctx.GetLog().Info("findForkPoint", "blockid", utils.F(forkPoint.GetBlockid()),
		"height", forkPoint.GetHeight())
	err = m.truncateForMiner(ctx, forkPoint.GetBlockid())
	return err
}
