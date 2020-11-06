package xuperos

import (
	"fmt"

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
	engine   def.Engine
	msgChan  chan *netPB.XuperMessage
	exitChan chan bool
}

func NewNetEvent(engine def.Engine) (*NetEvent, error) {
	if engine == nil {
		return nil, fmt.Errorf("new net event failed because param error")
	}

	obj := &NetEvent{
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
	var SyncMsgHandle = map[netPB.XuperMessage_MessageType]net.XuperHandler{
		netPB.XuperMessage_GET_BLOCK:                t.handleGetBlock,
		netPB.XuperMessage_GET_BLOCKCHAINSTATUS:     t.handleGetChainStatus,
		netPB.XuperMessage_CONFIRM_BLOCKCHAINSTATUS: t.handleConfirmChainStatus,
		netPB.XuperMessage_GET_RPC_PORT:             t.handleGetRPCPort,
	}

	// 订阅异步处理消息
	for _, msgType := range AsyncMsgList {
		// 注册订阅

	}

	// 订阅同步处理消息
	for msgType, handle := range SyncMsgHandle {
		// 注册订阅
	}

	return fmt.Errorf("the interface is not implemented")
}

// 阻塞等待chan中消息，直到收到退出信号
func (t *NetEvent) procMsgLoop() {
	for {
		select {
		case msg := <-t.msgChan:
			go t.procAsyncMsg(msg)
		case <-t.exitChan:
			goto finish
		}
	}

finish:
	t.log.Trace("Wait for the processing message loop to end")
}

func (t *NetEvent) procAsyncMsg(msg *netPB.XuperMessage) {
	var AsyncMsgList = map[netPB.XuperMessage_MessageType]AsyncMsgHandle{
		netPB.XuperMessage_POSTTX:      t.handlePostTx,
		netPB.XuperMessage_SENDBLOCK:   t.handleSendBlock,
		netPB.XuperMessage_BATCHPOSTTX: t.handleBatchPostTx,
		netPB.XuperMessage_NEW_BLOCKID: t.handleNewBlockID,
	}

	// 解析消息

	// 验证消息

	// 处理任务
	if handle, ok := AsyncMsgList[msg.GetHeader().GetType()]; ok {
		handle(msg)
	}
}

func (t *NetEvent) handlePostTx(msg *netPB.XuperMessage) {

}

func (t *NetEvent) handleSendBlock(msg *netPB.XuperMessage) {

}

func (t *NetEvent) handleBatchPostTx(msg *netPB.XuperMessage) {

}

func (t *NetEvent) handleNewBlockID(msg *netPB.XuperMessage) {

}

func (t *NetEvent) handleGetBlock(ctx context.Context,
	msg *netPB.XuperMessage) (*netPB.XuperMessage, error) {

}

func (t *NetEvent) handleGetChainStatus(ctx context.Context,
	msg *netPB.XuperMessage) (*netPB.XuperMessage, error) {

}

func (t *NetEvent) handleConfirmChainStatus(ctx context.Context,
	msg *netPB.XuperMessage) (*netPB.XuperMessage, error) {

}

func (t *NetEvent) handleGetRPCPort(ctx context.Context,
	msg *netPB.XuperMessage) (*netPB.XuperMessage, error) {

}
