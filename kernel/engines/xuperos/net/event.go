package net

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"golang.org/x/sync/errgroup"

	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/reader"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/xpb"
	"github.com/xuperchain/xupercore/kernel/network"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/metrics"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

const (
	// DefMsgChanBufSize 默认消息队列buf大小
	DefMsgChanBufSize = 50000
)

// AsyncMsgHandle 异步消息处理handle类型
type AsyncMsgHandle func(xctx.XContext, *protos.XuperMessage)

type Event struct {
	log      logs.Logger
	engine   common.Engine
	msgChan  chan *protos.XuperMessage
	exitChan chan bool
}

func NewEvent(engine common.Engine) (*Event, error) {
	if engine == nil {
		return nil, fmt.Errorf("new net event failed because param error")
	}

	obj := &Event{
		log:      engine.Context().XLog,
		engine:   engine,
		msgChan:  make(chan *protos.XuperMessage, DefMsgChanBufSize),
		exitChan: make(chan bool, 1),
	}

	// 订阅监听事件
	err := obj.Subscriber()
	if err != nil {
		return nil, fmt.Errorf("new net event failed because register subscriber error.err:%v", err)
	}

	return obj, nil
}

// Start 阻塞
func (e *Event) Start() {
	// 启动等待处理消息循环
	e.procMsgLoop()
}

func (e *Event) Stop() {
	// 通知退出循环
	e.exitChan <- true
}

func (e *Event) Subscriber() error {
	// 走异步处理的网络消息列表
	var AsyncMsgList = []protos.XuperMessage_MessageType{
		protos.XuperMessage_POSTTX,
		protos.XuperMessage_SENDBLOCK,
		protos.XuperMessage_BATCHPOSTTX,
		protos.XuperMessage_NEW_BLOCKID,
	}

	// 走同步处理的网络消息句柄
	var SyncMsgHandle = map[protos.XuperMessage_MessageType]p2p.HandleFunc{
		protos.XuperMessage_GET_BLOCK:                e.handleGetBlock,
		protos.XuperMessage_GET_BLOCKCHAINSTATUS:     e.handleGetChainStatus,
		protos.XuperMessage_CONFIRM_BLOCKCHAINSTATUS: e.handleConfirmChainStatus,
		protos.XuperMessage_GET_BLOCK_HEADERS:        e.handleGetBlockHeaders,
		protos.XuperMessage_GET_BLOCK_TXS:            e.handleGetBlockTxs,
	}

	net := e.net()
	// 订阅异步处理消息
	for _, msgType := range AsyncMsgList {
		// 注册订阅
		if err := net.Register(p2p.NewSubscriber(net.Context(), msgType, e.msgChan)); err != nil {
			e.log.Error("register subscriber error", "type", msgType, "error", err)
			return fmt.Errorf("register subscriber failed")
		}
	}

	// 订阅同步处理消息
	for msgType, handle := range SyncMsgHandle {
		// 注册订阅
		if err := net.Register(p2p.NewSubscriber(net.Context(), msgType, handle)); err != nil {
			e.log.Error("register subscriber error", "type", msgType, "error", err)
			return fmt.Errorf("register subscriber failed")
		}
	}

	e.log.Trace("register subscriber succ")
	return nil
}

// 阻塞等待chan中消息，直到收到退出信号
func (e *Event) procMsgLoop() {
	for {
		select {
		case request := <-e.msgChan:
			go e.procAsyncMsg(request)
		case <-e.exitChan:
			e.log.Trace("wait for the processing message loop to end")
			return
		}
	}
}

func (e *Event) procAsyncMsg(request *protos.XuperMessage) {
	var AsyncMsgList = map[protos.XuperMessage_MessageType]AsyncMsgHandle{
		protos.XuperMessage_POSTTX:      e.handlePostTx,
		protos.XuperMessage_SENDBLOCK:   e.handleSendBlock,
		protos.XuperMessage_BATCHPOSTTX: e.handleBatchPostTx,
		protos.XuperMessage_NEW_BLOCKID: e.handleNewBlockID,
	}

	// 处理任务
	log, _ := logs.NewLogger(request.Header.Logid, common.BCEngineName)
	ctx := &xctx.BaseCtx{
		XLog:  log,
		Timer: timer.NewXTimer(),
	}
	if handle, ok := AsyncMsgList[request.GetHeader().GetType()]; ok {
		beginTime := time.Now()
		handle(ctx, request)
		metrics.CallMethodHistogram.
			WithLabelValues(request.Header.Bcname, request.Header.Type.String()).
			Observe(time.Since(beginTime).Seconds())
	} else {
		log.Warn("received unregister request", "type", request.GetHeader().GetType())
		return
	}
}

func (e *Event) handlePostTx(ctx xctx.XContext, request *protos.XuperMessage) {
	var tx lpb.Transaction
	if err := p2p.Unmarshal(request, &tx); err != nil {
		ctx.GetLog().Warn("handlePostTx Unmarshal request error", "error", err)
		return
	}

	chain, err := e.engine.Get(request.Header.Bcname)
	if err != nil {
		ctx.GetLog().Warn("chain not exist", "error", err, "bcName", request.Header.Bcname)
		return
	}

	err = e.PostTx(ctx, chain, &tx)
	if err == nil {
		go e.sendMessage(ctx, request)
	}
}

func (e *Event) handleBatchPostTx(ctx xctx.XContext, request *protos.XuperMessage) {
	var input xpb.Transactions
	if err := p2p.Unmarshal(request, &input); err != nil {
		ctx.GetLog().Warn("handleBatchPostTx Unmarshal request error", "error", err)
		return
	}

	chain, err := e.engine.Get(request.Header.Bcname)
	if err != nil {
		ctx.GetLog().Warn("chain not exist", "error", err, "bcName", request.Header.Bcname)
		return
	}

	broadcastTx := make([]*lpb.Transaction, 0, len(input.Txs))
	for _, tx := range input.Txs {
		err := e.PostTx(ctx, chain, tx)
		if err != nil {
			ctx.GetLog().Warn("post tx error", "bcName", request.GetHeader().GetBcname(), "error", err)
			return
		}

		broadcastTx = append(broadcastTx, tx)
	}

	input.Txs = broadcastTx
	msg := p2p.NewMessage(protos.XuperMessage_BATCHPOSTTX, &input)

	go e.sendMessage(ctx, msg)
}

func (e *Event) PostTx(ctx xctx.XContext, chain common.Chain, tx *lpb.Transaction) error {
	if err := validatePostTx(tx); err != nil {
		ctx.GetLog().Trace("PostTx validate param error", "error", err)
		return common.CastError(err)
	}

	// chain已经Stop
	if chain.Context() == nil {
		return nil
	}

	if len(tx.TxInputs) == 0 && !chain.Context().Ledger.GetNoFee() {
		ctx.GetLog().Warn("TxInputs can not be null while need utxo")
		return common.ErrTxNotEnough
	}

	return chain.SubmitTx(ctx, tx)
}

func (e *Event) handleSendBlock(ctx xctx.XContext, request *protos.XuperMessage) {
	var block lpb.InternalBlock
	if err := p2p.Unmarshal(request, &block); err != nil {
		ctx.GetLog().Warn("handleSendBlock Unmarshal request error", "error", err)
		return
	}

	chain, err := e.engine.Get(request.Header.Bcname)
	if chain == nil {
		ctx.GetLog().Warn("chain not exist", "error", err, "bcName", request.Header.Bcname)
		return
	}

	if err := e.SendBlock(ctx, chain, &block); err != nil {
		return
	}

	var msg *protos.XuperMessage
	if e.engine.Context().EngCfg.BlockBroadcastMode == common.FullBroadCastMode {
		msg = request
	} else {
		blockID := &lpb.InternalBlock{
			Blockid: block.Blockid,
		}
		msg = p2p.NewMessage(protos.XuperMessage_NEW_BLOCKID, blockID, p2p.WithBCName(request.Header.Bcname))
	}
	go e.sendMessage(ctx, msg)
}

func (e *Event) handleNewBlockID(ctx xctx.XContext, request *protos.XuperMessage) {
	chain, err := e.engine.Get(request.Header.Bcname)
	if err != nil {
		ctx.GetLog().Warn("chain not exist", "error", err, "bcName", request.Header.Bcname)
		return
	}

	block, err := e.GetBlock(ctx, request)
	if err != nil {
		ctx.GetLog().Warn("GetBlock error", "error", err, "blockId", block.Blockid)
		return
	}

	if err := e.SendBlock(ctx, chain, block); err != nil {
		return
	}

	go e.sendMessage(ctx, request)
}

// sendMessage wrapper function which ignore error
func (e *Event) sendMessage(ctx xctx.XContext, msg *protos.XuperMessage, of ...p2p.OptionFunc) {
	_ = e.net().SendMessage(ctx, msg, of...)
}

func (e *Event) SendBlock(ctx xctx.XContext, chain common.Chain, in *lpb.InternalBlock) error {
	if err := validateSendBlock(in); err != nil {
		ctx.GetLog().Trace("SendBlock validate param error", "error", err)
		return err
	}

	if err := chain.ProcBlock(ctx, in); err != nil {
		if common.CastError(err).Equal(common.ErrForbidden) {
			ctx.GetLog().Trace("forbidden process block", "err", err)
			return err
		}

		if common.CastError(err).Equal(common.ErrParameter) {
			ctx.GetLog().Trace("process block param error", "err", err)
			return err
		}

		ctx.GetLog().Warn("process block error", "error", err)
		return err
	}

	// chain已经Stop
	if chain.Context() == nil {
		return nil
	}

	ledgerMeta := chain.Context().Ledger.GetMeta()
	stateMeta := chain.Context().State.GetMeta()
	ctx.GetLog().Info("SendBlock",
		"height", ledgerMeta.TrunkHeight,
		"last", utils.F(ledgerMeta.TipBlockid),
		"utxo", utils.F(stateMeta.GetLatestBlockid()),
		"genesis", utils.F(ledgerMeta.RootBlockid))
	return nil
}

func (e *Event) handleGetBlock(ctx xctx.XContext,
	request *protos.XuperMessage) (*protos.XuperMessage, error) {

	var input xpb.BlockID
	var output = new(xpb.BlockInfo)
	defer func(begin time.Time) {
		metrics.CallMethodHistogram.WithLabelValues("sync", "p2pGetBlock").Observe(time.Since(begin).Seconds())
	}(time.Now())

	bcName := request.Header.Bcname
	response := func(err error) (*protos.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(p2p.GetRespMessageType(request.GetHeader().GetType()), output, opts...)
		return resp, nil
	}

	err := p2p.Unmarshal(request, &input)
	if err != nil {
		ctx.GetLog().Error("unmarshal error", "bcName", bcName, "error", err)
		return response(common.ErrParameter)
	}

	chain, err := e.engine.Get(bcName)
	if err != nil {
		ctx.GetLog().Warn("chain not exist", "error", err, "bcName", bcName)
		return response(common.ErrChainNotExist)
	}

	ledgerReader := reader.NewLedgerReader(chain.Context(), ctx)
	if input.Blockid != nil {
		output, err = ledgerReader.QueryBlock(input.Blockid, input.NeedContent)
		if err != nil {
			ctx.GetLog().Error("ledger reader query block error", "error", err)
			return response(err)
		}
		ctx.GetLog().SetInfoField("height", output.Block.Height)
		ctx.GetLog().SetInfoField("blockId", utils.F(output.Block.Blockid))
		ctx.GetLog().SetInfoField("status", output.Status)
	}

	return response(nil)
}

func (e *Event) handleGetBlockHeaders(ctx xctx.XContext,
	request *protos.XuperMessage) (*protos.XuperMessage, error) {

	output := new(xpb.GetBlockHeaderResponse)
	defer func(begin time.Time) {
		metrics.CallMethodHistogram.WithLabelValues("sync", "p2pGetBlockHeaders").Observe(time.Since(begin).Seconds())
	}(time.Now())

	bcName := request.Header.Bcname
	response := func(err error) (*protos.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(p2p.GetRespMessageType(request.GetHeader().GetType()), output, opts...)
		return resp, nil
	}

	var input xpb.GetBlockHeaderRequest
	err := p2p.Unmarshal(request, &input)
	if err != nil {
		ctx.GetLog().Error("unmarshal error", "bcName", bcName, "error", err)
		return response(common.ErrParameter)
	}

	chain, err := e.engine.Get(bcName)
	if err != nil {
		ctx.GetLog().Warn("chain not exist", "error", err, "bcName", bcName)
		return response(common.ErrChainNotExist)
	}

	ledgerReader := reader.NewLedgerReader(chain.Context(), ctx)

	if input.Height < 0 {
		return response(errors.New("bad input height"))
	}

	// TODO: max input.Size
	blocks := make([]*lpb.InternalBlock, input.Size)
	group := errgroup.Group{}
	mutex := sync.Mutex{}
	maxIdx := -1
	for i := int64(0); i < input.GetSize(); i++ {
		i := i
		height := input.Height + i
		group.Go(func() error {
			blkInfo, err := ledgerReader.QueryBlockHeaderByHeight(height)
			if err != nil {
				ctx.GetLog().Debug("query block header error", "error", err, "height", height)
				return err
			}
			if blkInfo.Status == lpb.BlockStatus_BLOCK_NOEXIST {
				ctx.GetLog().Debug("query block header error", "error", "not exist", "height", height)
				return nil
			}
			// 拷贝区块头，避免修改原缓存
			block := *blkInfo.Block
			// 取coinbase交易
			if block.TxCount > 0 {
				txID := block.MerkleTree[0]
				coinbaseTx, err := ledgerReader.QueryTx(txID)
				if err == nil {
					// 避免修改Transactions结构
					block.Transactions = []*lpb.Transaction{coinbaseTx.GetTx()}
				}
			}
			ctx.GetLog().Debug("query block header", "height", height, "size", proto.Size(&block))
			mutex.Lock()
			blocks[i] = &block
			if int(i) > maxIdx {
				maxIdx = int(i)
			}
			mutex.Unlock()
			return nil
		})
	}
	err = group.Wait()
	if err != nil {
		return response(err)
	}
	output.Blocks = blocks[:maxIdx+1]

	return response(nil)
}

func (e *Event) handleGetBlockTxs(ctx xctx.XContext,
	request *protos.XuperMessage) (*protos.XuperMessage, error) {

	output := new(xpb.GetBlockTxsResponse)
	defer func(begin time.Time) {
		metrics.CallMethodHistogram.WithLabelValues("sync", "p2pGetBlockTxs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	bcName := request.Header.Bcname
	response := func(err error) (*protos.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(p2p.GetRespMessageType(request.GetHeader().GetType()), output, opts...)
		return resp, nil
	}

	var input xpb.GetBlockTxsRequest
	err := p2p.Unmarshal(request, &input)
	if err != nil {
		ctx.GetLog().Error("unmarshal error", "bcName", bcName, "error", err)
		return response(common.ErrParameter)
	}

	chain, err := e.engine.Get(bcName)
	if err != nil {
		ctx.GetLog().Warn("chain not exist", "error", err, "bcName", bcName)
		return response(common.ErrChainNotExist)
	}

	ledger := chain.Context().Ledger

	if input.Blockid != nil && len(input.Txs) > 0 {
		header, err := ledger.QueryBlockHeader(input.Blockid)
		if err != nil {
			return response(err)
		}
		blockTxIDs := header.GetMerkleTree()[:header.GetTxCount()]
		for _, idx := range input.Txs {
			if int(idx) >= len(blockTxIDs) {
				return response(fmt.Errorf("bad tx index, got:%d, max:%d, count:%d", idx, len(blockTxIDs)-1, header.TxCount))
			}
			txID := blockTxIDs[idx]
			tx, err := ledger.QueryTransaction(txID)
			if err != nil {
				return response(err)
			}
			output.Txs = append(output.Txs, tx)
		}
	}

	return response(nil)
}

func (e *Event) handleGetChainStatus(ctx xctx.XContext, request *protos.XuperMessage) (*protos.XuperMessage, error) {
	var output *xpb.ChainStatus

	bcName := request.GetHeader().GetBcname()
	response := func(err error) (*protos.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(p2p.GetRespMessageType(request.GetHeader().GetType()), output, opts...)
		return resp, nil
	}

	chain, err := e.engine.Get(bcName)
	if err != nil {
		ctx.GetLog().Warn("chain not exist", "error", err, "bcName", request.Header.Bcname)
		return response(common.ErrChainNotExist)
	}

	chainReader := reader.NewChainReader(chain.Context(), ctx)
	output, err = chainReader.GetChainStatus()
	if err != nil {
		ctx.GetLog().Error("handleGetChainStatus error", "error", err)
		return response(err)
	}

	return response(nil)
}

func (e *Event) handleConfirmChainStatus(ctx xctx.XContext,
	request *protos.XuperMessage) (*protos.XuperMessage, error) {

	var input lpb.InternalBlock
	var output *xpb.TipStatus

	bcName := request.GetHeader().GetBcname()
	response := func(err error) (*protos.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(p2p.GetRespMessageType(request.GetHeader().GetType()), output, opts...)
		return resp, nil
	}

	err := p2p.Unmarshal(request, &input)
	if err != nil {
		ctx.GetLog().Error("unmarshal error", "bcName", bcName, "error", err)
		return response(common.ErrParameter)
	}

	chain, err := e.engine.Get(bcName)
	if err != nil {
		ctx.GetLog().Warn("chain not exist", "error", err, "bcName", request.Header.Bcname)
		return response(common.ErrChainNotExist)
	}

	chainReader := reader.NewChainReader(chain.Context(), ctx)
	chainStatus, err := chainReader.GetChainStatus()
	if err != nil {
		ctx.GetLog().Error("handleConfirmChainStatus error", "bcName", bcName, "error", err)
		return response(err)
	}

	output = &xpb.TipStatus{
		IsTrunkTip: false,
	}
	if bytes.Equal(input.Blockid, chainStatus.LedgerMeta.TipBlockid) {
		output.IsTrunkTip = true
	}

	return response(nil)
}

// net gets Net object in engine context
func (e *Event) net() network.Network {
	return e.engine.Context().Net
}
