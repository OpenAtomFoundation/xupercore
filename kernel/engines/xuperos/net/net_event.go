package xuperos

import (
	"bytes"
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xuperchain/core/global"
	"github.com/xuperchain/xuperchain/core/pb"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/logs"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	netPB "github.com/xuperchain/xupercore/kernel/network/pb"
)

const (
	// 默认消息队列buf大小
	DefMsgChanBufSize = 50000
)

// 异步消息处理handle类型
type AsyncMsgHandle func(*netPB.XuperMessage)

type NetEvent struct {
	log      logs.Logger
	engine   def.Engine
	msgChan  chan *netPB.XuperMessage
	exitChan chan bool
}

func NewNetEvent(engine def.Engine) (*NetEvent, error) {
	if engine == nil {
		return nil, fmt.Errorf("new net event failed because param error")
	}

	obj := &NetEvent{
		log:      engine.Context().XLog,
		engine:   engine,
		msgChan:  make(chan *netPB.XuperMessage, DefMsgChanBufSize),
		exitChan: make(chan bool, 1),
	}

	// 订阅监听事件
	err := obj.Subscriber()
	if err != nil {
		return nil, fmt.Errorf("new net event failed because register subscriber error.err:%v", err)
	}

	return obj, nil
}

// 阻塞
func (t *NetEvent) Start() {
	// 启动等待处理消息循环
	t.procMsgLoop()
}

func (t *NetEvent) Stop() {
	// 通知退出循环
	t.exitChan <- true
}

func (t *NetEvent) Subscriber() error {
	// 走异步处理的网络消息列表
	var AsyncMsgList = []netPB.XuperMessage_MessageType{
		netPB.XuperMessage_POSTTX,
		netPB.XuperMessage_SENDBLOCK,
		netPB.XuperMessage_BATCHPOSTTX,
		netPB.XuperMessage_NEW_BLOCKID,
	}

	// 走同步处理的网络消息句柄
	var SyncMsgHandle = map[netPB.XuperMessage_MessageType]p2p.HandleFunc{
		netPB.XuperMessage_GET_BLOCK:                t.handleGetBlock,
		netPB.XuperMessage_GET_BLOCKCHAINSTATUS:     t.handleGetChainStatus,
		netPB.XuperMessage_CONFIRM_BLOCKCHAINSTATUS: t.handleConfirmChainStatus,
		netPB.XuperMessage_GET_BLOCKIDS:             t.handleGetBlockIds,
		netPB.XuperMessage_GET_BLOCKS:               t.handleGetBlocks,
	}

	net := t.engine.Context().Net
	// 订阅异步处理消息
	for _, msgType := range AsyncMsgList {
		// 注册订阅
		if err := net.Register(p2p.NewSubscriber(net.Context(), msgType, t.msgChan)); err != nil {
			t.log.Error("register subscriber error", "type", msgType, "error", err)
		}
	}

	// 订阅同步处理消息
	for msgType, handle := range SyncMsgHandle {
		// 注册订阅
		// TODO: ctx
		if err := net.Register(p2p.NewSubscriber(net.Context(), msgType, handle)); err != nil {
			t.log.Error("register subscriber error", "type", msgType, "error", err)
		}
	}

	return fmt.Errorf("the interface is not implemented")
}

// 阻塞等待chan中消息，直到收到退出信号
func (t *NetEvent) procMsgLoop() {
	for {
		select {
		case request := <-t.msgChan:
			go t.procAsyncMsg(request)
		case <-t.exitChan:
			t.log.Trace("Wait for the processing message loop to end")
			return
		}
	}
}

func (t *NetEvent) procAsyncMsg(request *netPB.XuperMessage) {
	var AsyncMsgList = map[netPB.XuperMessage_MessageType]AsyncMsgHandle{
		netPB.XuperMessage_POSTTX:      t.handlePostTx,
		netPB.XuperMessage_SENDBLOCK:   t.handleSendBlock,
		netPB.XuperMessage_BATCHPOSTTX: t.handleBatchPostTx,
		netPB.XuperMessage_NEW_BLOCKID: t.handleNewBlockID,
	}

	// 处理任务
	if handle, ok := AsyncMsgList[request.GetHeader().GetType()]; ok {
		handle(request)
	} else {
		t.log.Warn("received unregister request", "logid", request.GetHeader().GetLogid(), "type", request.GetHeader().GetType())
		return
	}
}

func (t *NetEvent) handlePostTx(request *netPB.XuperMessage) {
	var txStatus pb.TxStatus
	if err := p2p.Unmarshal(request, &txStatus); err != nil {
		t.log.Warn("handlePostTx Unmarshal request error", "logid", request.GetHeader().GetLogid(), "error", err)
		return
	}

	if txStatus.Header == nil {
		txStatus.Header = global.GHeader()
	}

	out, _ := t.PostTx(&txStatus)
	if out != nil && out.Header.Error == pb.XChainErrorEnum_SUCCESS {
		ctx := t.engine.Context()
		go ctx.Net.SendMessage(context.Background(), request)
	}
}

func (t *NetEvent) PostTx(in *pb.TxStatus) (*pb.CommonReply, error) {
	out := &pb.CommonReply{Header: &pb.Header{Logid: in.Header.Logid}}
	if err := validatePostTx(in); err != nil {
		out.Header.Error = pb.XChainErrorEnum_VALIDATE_ERROR
		t.log.Trace("PostTx validate param errror", "logid", in.Header.Logid, "error", err.Error())
		return out, err
	}

	chain := t.engine.Get(in.GetBcname())
	if chain == nil {
		out.Header.Error = pb.XChainErrorEnum_BLOCKCHAIN_NOTEXIST
		t.log.Warn("block chain not exist", "logid", in.Header.Logid, "bcname", in.GetBcname())
		return out, def.ErrBlockChainNotExist
	}

	ctx := chain.Context()
	if len(in.Tx.TxInputs) == 0 && !ctx.Ledger.GetNoFee() {
		out.Header.Error = pb.XChainErrorEnum_NOT_ENOUGH_UTXO_ERROR // 拒绝
		t.log.Warn("PostTx TxInputs can not be null while need utxo", "logid", in.Header.Logid)
		return out, nil
	}

	if t.GetNodeMode() == def.NodeModeFastSync {
		out.Header.Error = pb.XChainErrorEnum_CONNECT_REFUSE // 拒绝
		t.log.Warn("PostTx NodeMode is FAST_SYNC, refused!")
		return out, nil
	}

	err := chain.ProcTx(in)
	out.Header.Error = def.ErrorEnum(err)
	return out, err
}

func (t *NetEvent) handleSendBlock(request *netPB.XuperMessage) {
	var block pb.Block
	if err := p2p.Unmarshal(request, &block); err != nil {
		t.log.Warn("handlePostTx Unmarshal request error", "logid", request.GetHeader().GetLogid(), "error", err)
		return
	}

	if block.Header == nil {
		block.Header = global.GHeader()
	}

	if err := t.SendBlock(&block); err != nil {
		t.log.Warn("proc block error", "logid", request.GetHeader().GetLogid(), "bcname", block.Bcname, "error", err)
		return
	}

	ctx := t.engine.Context()
	if ctx.EngCfg.BlockBroadcastMode == def.FullBroadCastMode {
		go ctx.Net.SendMessage(context.Background(), request)
	} else {
		blockID := &pb.Block{
			Bcname:  block.Bcname,
			Blockid: block.Blockid,
		}
		msg := p2p.NewMessage(netPB.XuperMessage_NEW_BLOCKID, blockID)
		go ctx.Net.SendMessage(context.Background(), msg)
	}
}

func (t *NetEvent) handleNewBlockID(request *netPB.XuperMessage) {
	var block pb.Block
	if err := p2p.Unmarshal(request, &block); err != nil {
		t.log.Warn("handleNewBlockID Unmarshal request error", "logid", request.GetHeader().GetLogid(), "error", err)
		return
	}

	in := &pb.BlockID{
		Header: &pb.Header{
			Logid: request.GetHeader().GetLogid(),
		},
		Bcname: request.GetHeader().GetBcname(),
		Blockid: block.Blockid,
		NeedContent: true,
	}
	out, err := t.GetBlock(context.Background(), in, p2p.WithPeerIDs([]string{request.GetHeader().GetFrom()}))
	if err != nil {
		t.log.Warn("GetBlock error", "logid", request.GetHeader().GetLogid(), "error", err, "blockId", block.Blockid)
		return
	}

	if out.Header == nil {
		out.Header = global.GHeader()
	}

	if err := t.SendBlock(out); err != nil {
		t.log.Warn("proc block error", "logid", request.GetHeader().GetLogid(), "bcname", block.Bcname, "error", err)
		return
	}

	ctx := t.engine.Context()
	go ctx.Net.SendMessage(context.Background(), request)
	return
}

func (t *NetEvent) SendBlock(in *pb.Block) error {
	if err := validateSendBlock(in); err != nil {
		t.log.Trace("SendBlock validate param errror", "logid", in.Header.Logid, "error", err)
		return err
	}

	chain := t.engine.Get(in.GetBcname())
	if chain == nil {
		t.log.Warn("block chain not exist", "logid", in.GetHeader().GetLogid(), "bcname", in.GetBcname())
		return def.ErrBlockChainNotExist
	}

	if err := chain.ProcBlock(in); err != nil {
		t.log.Warn("proc block error", "logid", in.GetHeader().GetLogid(), "bcname", in.GetBcname(), "error", err)
		return err
	}

	ledgerMeta := chain.Context().Ledger.GetMeta()
	stateMeta := chain.Context().State.GetMeta()
	t.log.Info("SendBlock", "logid", in.GetHeader().GetLogid(),
		"genesis", fmt.Sprintf("%x", ledgerMeta.RootBlockid),
		"last", fmt.Sprintf("%x", ledgerMeta.TipBlockid),
		"height", ledgerMeta.TrunkHeight,
		"utxo", global.F(stateMeta.GetLatestBlockid()))
	return nil
}

func (t *NetEvent) handleBatchPostTx(request *netPB.XuperMessage) {
	var txs pb.BatchTxs
	if err := p2p.Unmarshal(request, &txs); err != nil {
		t.log.Warn("handleBatchPostTx Unmarshal request error", "logid", request.GetHeader().GetLogid(), "error", err)
		return
	}

	broadcastTx := make([]*pb.TxStatus, 0, len(txs.Txs))
	for _, tx := range txs.Txs {
		out, err := t.PostTx(tx)
		if err != nil || out.Header.Error != pb.XChainErrorEnum_SUCCESS {
			t.log.Warn("post tx error", "logid", request.GetHeader().GetLogid(), "bcname", request.GetHeader().GetBcname(), "error", err)
			return
		}

		broadcastTx = append(broadcastTx, tx)
	}

	txs.Txs = broadcastTx
	msg := p2p.NewMessage(netPB.XuperMessage_BATCHPOSTTX, &txs)

	ctx := t.engine.Context()
	go ctx.Net.SendMessage(context.Background(), msg)
}

func (t *NetEvent) handleGetBlock(ctx context.Context, request *netPB.XuperMessage) (*netPB.XuperMessage, error) {
	var input pb.BlockID
	var output *pb.Block

	bcName := request.GetHeader().GetBcname()
	response := func(err error) (*netPB.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(def.ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(netPB.XuperMessage_GET_BLOCKIDS_RES, output, opts...)
		return resp, err
	}

	err := p2p.Unmarshal(request, &input)
	if err != nil {
		t.log.Error("unmarshal error", "logid", request.GetHeader().GetLogid(), "bcName", bcName, "error", err)
		return response(def.ErrMessageUnmarshal)
	}

	chain := t.engine.Get(bcName)
	if chain == nil {
		t.log.Error("chain not exist", "logid", request.GetHeader().GetLogid(), "bcName", bcName)
		return response(def.ErrBlockChainNotExist)
	}

	output = chain.Reader().QueryBlock(&input)
	if output.GetHeader().GetError() != pb.XChainErrorEnum_SUCCESS {
		t.log.Error("handleGetBlock GetBlock error", "logid", request.GetHeader().GetLogid(), "bcName", bcName, "error", output.Header.Error)
		return response(def.ErrGetBlockError)
	}

	return response(nil)
}

func (t *NetEvent) handleGetChainStatus(ctx context.Context, request *netPB.XuperMessage) (*netPB.XuperMessage, error) {
	var input pb.BCStatus
	var output *pb.BCStatus

	bcName := request.GetHeader().GetBcname()
	response := func(err error) (*netPB.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(def.ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(netPB.XuperMessage_GET_BLOCKCHAINSTATUS_RES, output, opts...)
		return resp, err
	}

	err := p2p.Unmarshal(request, &input)
	if err != nil {
		t.log.Error("unmarshal error", "logid", request.GetHeader().GetLogid(), "bcName", bcName, "error", err)
		return response(def.ErrMessageUnmarshal)
	}

	chain := t.engine.Get(bcName)
	if chain == nil {
		t.log.Error("chain not exist", "logid", request.GetHeader().GetLogid(), "bcName", bcName)
		return response(def.ErrBlockChainNotExist)
	}

	output = chain.Reader().GetBlockChainStatus(&input, pb.ViewOption_NONE)
	// no need to transfer branch id msg
	output.BranchBlockid = nil
	if output.GetHeader().GetError() != pb.XChainErrorEnum_SUCCESS {
		t.log.Error("handleGetChainStatus error", "logid", request.GetHeader().GetLogid(), "bcName", bcName, "error", output.Header.Error)
		return response(def.ErrGetBlockChainError)
	}

	return response(nil)
}

func (t *NetEvent) handleConfirmChainStatus(ctx context.Context, request *netPB.XuperMessage) (*netPB.XuperMessage, error) {
	var input pb.BCStatus
	var output *pb.BCTipStatus

	bcName := request.GetHeader().GetBcname()
	response := func(err error) (*netPB.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(def.ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(netPB.XuperMessage_CONFIRM_BLOCKCHAINSTATUS_RES, output, opts...)
		return resp, err
	}

	err := p2p.Unmarshal(request, &input)
	if err != nil {
		t.log.Error("unmarshal error", "logid", request.GetHeader().GetLogid(), "bcName", bcName, "error", err)
		return response(def.ErrMessageUnmarshal)
	}

	chain := t.engine.Get(bcName)
	if chain == nil {
		t.log.Error("chain not exist", "logid", request.GetHeader().GetLogid(), "bcName", bcName)
		return response(def.ErrBlockChainNotExist)
	}

	output = chain.Reader().GetTipStatus(&input)
	if output.GetHeader().GetError() != pb.XChainErrorEnum_SUCCESS {
		t.log.Error("handleConfirmChainStatus error", "logid", request.GetHeader().GetLogid(), "bcName", bcName, "error", output.Header.Error)
		return response(def.ErrGetBlockChainStatusError)
	}

	return response(nil)
}

func (t *NetEvent) handleGetBlockIds(ctx context.Context, request *netPB.XuperMessage) (*netPB.XuperMessage, error) {
	var input pb.GetBlockIdsRequest
	var output pb.GetBlockIdsResponse

	bcName := request.GetHeader().GetBcname()
	response := func(err error) (*netPB.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(def.ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(netPB.XuperMessage_GET_BLOCKIDS_RES, &output, opts...)
		return resp, err
	}

	err := p2p.Unmarshal(request, &input)
	if err != nil {
		t.log.Error("unmarshal error", "logid", request.GetHeader().GetLogid(), "bcName", bcName, "error", err)
		return response(def.ErrMessageUnmarshal)
	}

	if input.GetCount() <= 0 {
		t.log.Error("handleGetBlockIds: headersCount invalid", "logid", request.GetHeader().GetLogid(), "headersCount", input.GetCount())
		return response(def.ErrMessageParam)
	}

	chain := t.engine.Get(bcName)
	if chain == nil {
		t.log.Error("chain not exist", "logid", request.GetHeader().GetLogid(), "bcName", bcName)
		return response(def.ErrBlockChainNotExist)
	}

	chainCtx := chain.Context()

	tipBlockId := chainCtx.State.GetLatestBlockId()
	output.TipBlockId = tipBlockId

	block, err := chainCtx.Ledger.QueryBlock(input.GetBlockId())
	if err != nil {
		t.log.Warn("handleGetBlockIds: not found blockId", "Logid", request.GetHeader().GetLogid(), "error", err, "headerBlockId", global.F(input.GetBlockId()))
		return response(nil)
	}

	if !block.GetInTrunk() {
		t.log.Warn("handleGetBlockIds: block not in trunk", "Logid", request.GetHeader().GetLogid(), "headerBlock", global.F(input.GetBlockId()))
		return response(nil)
	}

	blockIds := make([][]byte, 0, input.GetCount())
	for !bytes.Equal(tipBlockId, block.GetBlockid()) {
		nextBlock, err := chainCtx.Ledger.QueryBlockHeader(block.GetNextHash())
		if err != nil {
			t.log.Warn("handleGetBlockIds: QueryBlock error", "error", err, "blockId", global.F(block.GetNextHash()))
			break
		}

		blockIds = append(blockIds, nextBlock.GetBlockid())
		block = nextBlock
	}

	output.BlockIds = blockIds
	t.log.Debug("handleGetBlockIds", "logid", request.GetHeader().GetLogid(), "tipBlockId", tipBlockId, "blockIds", len(blockIds))
	return response(nil)
}

func (t *NetEvent) handleGetBlocks(ctx context.Context, request *netPB.XuperMessage) (*netPB.XuperMessage, error) {
	var input pb.GetBlocksRequest
	var output pb.GetBlocksResponse
	bcName := request.GetHeader().GetBcname()

	response := func(err error) (*netPB.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(def.ErrorType(err)),
			p2p.WithLogId(request.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(netPB.XuperMessage_GET_BLOCKS_RES, &output, opts...)
		return resp, err
	}

	err := p2p.Unmarshal(request, &input)
	if err != nil {
		t.log.Error("unmarshal error", "logid", request.GetHeader().GetLogid(), "bcName", bcName, "error", err)
		return response(def.ErrMessageUnmarshal)
	}

	chain := t.engine.Get(bcName)
	if chain == nil {
		t.log.Error("chain not exist", "logid", request.GetHeader().GetLogid(), "bcName", bcName)
		return response(def.ErrBlockChainNotExist)
	}

	chainCtx := chain.Context()

	blockSizeQuota := int(chainCtx.Ledger.GetMaxBlockSize())
	blocks := make([]*pb.InternalBlock, 0, len(input.GetBlockIds()))
	for _, blockId := range input.GetBlockIds() {
		block, err := chainCtx.Ledger.QueryBlock(blockId)
		if err != nil {
			t.log.Warn("handleGetBlockIds: QueryBlock error", "logid", request.GetHeader().GetLogid(), "blockId", global.F(blockId), "error", err)
			continue
		}

		blockSizeQuota -= proto.Size(block)
		if blockSizeQuota > 0 {
			break
		}

		blocks = append(blocks, block)
	}

	output.BlocksInfo = blocks
	return response(nil)
}
