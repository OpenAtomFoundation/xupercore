package p2pv2

import (
	"fmt"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/mock"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/protos"
	"testing"
)

func Handler(ctx xctx.XContext, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	typ := p2p.GetRespMessageType(msg.Header.Type)
	resp := p2p.NewMessage(typ, msg, p2p.WithLogId(msg.Header.Logid))
	return resp, nil
}

func HandlerError(ctx xctx.XContext, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	return nil, fmt.Errorf("handler error")
}

func startNode1(t *testing.T) {
	ecfg, err := mock.NewEnvConfForTest("p2pv2/node1/conf/env.yaml")
	if err != nil {
		t.Errorf("env conf error: %v", err)
		return
	}

	fmt.Printf("root=%s, net=%s", ecfg.RootPath, ecfg.NetConf)

	ctx, err := nctx.NewNetCtx(ecfg)
	if err != nil {
		t.Errorf("net ctx error: %v", err)
		return
	}

	node := NewP2PServerV2()
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
		return
	}

	node.Start()
	ch := make(chan *pb.XuperMessage, 1024)
	if err := node.Register(p2p.NewSubscriber(ctx, pb.XuperMessage_POSTTX, ch)); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}

	if err := node.Register(p2p.NewSubscriber(ctx, pb.XuperMessage_GET_BLOCK, p2p.HandleFunc(Handler))); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}

	go func(t *testing.T) {
		select {
		case msg := <-ch:
			t.Logf("recv msg: log_id=%v, msgType=%s\n", msg.GetHeader().GetLogid(), msg.GetHeader().GetType())
		}
	}(t)
}

func startNode2(t *testing.T) {
	ecfg, _ := mock.NewEnvConfForTest("p2pv2/node2/conf/env.yaml")
	ctx, _ := nctx.NewNetCtx(ecfg)

	node := NewP2PServerV2()
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
		return
	}

	node.Start()
	if err := node.Register(p2p.NewSubscriber(ctx, pb.XuperMessage_GET_RPC_PORT, p2p.HandleFunc(Handler))); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}

	if err := node.Register(p2p.NewSubscriber(ctx, pb.XuperMessage_GET_BLOCK, p2p.HandleFunc(HandlerError))); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}
}

func startNode3(t *testing.T) {
	ecfg, _ := mock.NewEnvConfForTest("p2pv2/node3/conf/env.yaml")
	ctx, _ := nctx.NewNetCtx(ecfg)

	node := NewP2PServerV2()
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
		return
	}

	node.Start()
	msg := p2p.NewMessage(pb.XuperMessage_POSTTX, nil)
	if err := node.SendMessage(ctx, msg); err != nil {
		t.Errorf("sendMessage error: %v", err)
	}

	msg = p2p.NewMessage(pb.XuperMessage_GET_BLOCK, nil)
	if responses, err := node.SendMessageWithResponse(ctx, msg); err != nil {
		t.Errorf("sendMessage error: %v", err)
	} else {
		for i, resp := range responses {
			t.Logf("resp[%d]: log_id=%v", i, resp)
		}
	}
}

func TestP2PServerV2(t *testing.T) {
	mock.InitLogForTest()

	go startNode1(t)
	startNode2(t)
	startNode3(t)
}
