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
		netPB.XuperMessage_GET_RPC_PORT:             t.handleGetRPCPort,
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
			goto finish
		}
	}

finish:
	t.log.Trace("Wait for the processing message loop to end")
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
	txStatus := &pb.TxStatus{}
	if err := p2p.Unmarshal(request, txStatus); err != nil {
		t.log.Warn("handlePostTx Unmarshal request error", "logid", request.GetHeader().GetLogid(), "error", err)
		return
	}
}

func (t *NetEvent) handleSendBlock(request *netPB.XuperMessage) {

}

func (t *NetEvent) handleBatchPostTx(request *netPB.XuperMessage) {

}

func (t *NetEvent) handleNewBlockID(request *netPB.XuperMessage) {

}

func (t *NetEvent) handleGetBlock(ctx context.Context, request *netPB.XuperMessage) (*netPB.XuperMessage, error) {
	var input pb.BlockID
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

	chain := t.engine.Get(bcName)
	if chain == nil {
		t.log.Error("chain not exist", "logid", request.GetHeader().GetLogid(), "bcName", bcName)
		return response(def.ErrBlockChainNotExist)
	}

	return nil, nil
}

func (t *NetEvent) handleGetChainStatus(ctx context.Context, request *netPB.XuperMessage) (*netPB.XuperMessage, error) {
	return nil, nil
}

func (t *NetEvent) handleConfirmChainStatus(ctx context.Context, request *netPB.XuperMessage) (*netPB.XuperMessage, error) {
	return nil, nil
}

func (t *NetEvent) handleGetRPCPort(ctx context.Context, request *netPB.XuperMessage) (*netPB.XuperMessage, error) {
	return nil, nil
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
