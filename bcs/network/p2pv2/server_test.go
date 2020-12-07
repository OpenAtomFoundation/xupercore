package p2pv2

import (
	"context"
	"testing"

	"github.com/xuperchain/xupercore/kernel/mock"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/protos"
)

type handler struct{}

func (h *handler) Handler(ctx context.Context, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	typ := p2p.GetRespMessageType(msg.Header.Type)
	resp := p2p.NewMessage(typ, msg, p2p.WithLogId(msg.Header.Logid))
	return resp, nil
}

func startNode1(t *testing.T) {
	ecfg, _ := mock.NewEnvConfForTest()
	ecfg.NetConf = "p2pv2/node1.yaml"
	ctx, _ := nctx.NewNetCtx(ecfg)
	ctx.P2PConf.KeyPath = "p2pv2/node1/data/netkeys"
	ctx.P2PConf.P2PDataPath = "p2pv2/node1/data/p2p"

	node := NewP2PServerV2()
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
	}

	node.Start()
	ch := make(chan *pb.XuperMessage, 1024)
	if err := node.Register(p2p.NewSubscriber(ctx, pb.XuperMessage_POSTTX, ch)); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}

	if err := node.Register(p2p.NewSubscriber(ctx, pb.XuperMessage_GET_BLOCK, &handler{})); err != nil {
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
	mock.InitLogForTest()
	ecfg, _ := mock.NewEnvConfForTest()
	ecfg.NetConf = "p2pv2/node2.yaml"
	ctx, _ := nctx.NewNetCtx(ecfg)
	ctx.P2PConf.KeyPath = "p2pv2/node2/data/netkeys"
	ctx.P2PConf.P2PDataPath = "p2pv2/node2/data/p2p"

	node := NewP2PServerV2()
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
	}

	node.Start()
	if err := node.Register(p2p.NewSubscriber(ctx, pb.XuperMessage_GET_RPC_PORT, &handler{})); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}

	if err := node.Register(p2p.NewSubscriber(ctx, pb.XuperMessage_GET_BLOCK, &handler{})); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}
}

func startNode3(t *testing.T) {
	mock.InitLogForTest()
	ecfg, _ := mock.NewEnvConfForTest()
	ecfg.NetConf = "p2pv2/node3.yaml"
	ctx, _ := nctx.NewNetCtx(ecfg)
	ctx.P2PConf.KeyPath = "p2pv2/node3/data/netkeys"
	ctx.P2PConf.P2PDataPath = "p2pv2/node3/data/p2p"

	node := NewP2PServerV2()
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
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
