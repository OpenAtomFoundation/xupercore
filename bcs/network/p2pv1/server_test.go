package p2pv1

import (
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/kernel/network/pb"
	"github.com/xuperchain/xupercore/lib/utils"
	"path/filepath"
	"testing"
	"time"
)

var (
	nodePath = filepath.Join(utils.GetCurFileDir() + "/test")
)

type handler struct{}

func (h *handler) Handler(ctx nctx.OperateCtx, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	typ := p2p.GetRespMessageType(msg.Header.Type)
	resp := p2p.NewMessage(typ, msg, p2p.WithLogId(msg.Header.Logid))
	return resp, nil
}

func startNode1(t *testing.T) {
	node := NewP2PServerV1()
	ctx := nctx.MockDomainCtx(filepath.Join(nodePath, "/node1/conf/network.yaml"))
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
			t.Logf("recv msg: log_id=%v, msgType=%s", msg.GetHeader().GetLogid(), msg.GetHeader().GetType())
		}
	}(t)
}

func startNode2(t *testing.T) {
	node := NewP2PServerV1()
	ctx := nctx.MockDomainCtx(filepath.Join(nodePath, "/node2/conf/network.yaml"))
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
	}

	node.Start()
	if err := node.Register(p2p.NewSubscriber(ctx, pb.XuperMessage_GET_BLOCK, &handler{})); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}
}

func startNode3(t *testing.T) {
	node := NewP2PServerV1()
	ctx := nctx.MockDomainCtx(filepath.Join(nodePath, "/node3/conf/network.yaml"))
	ctx.SetMetricSwitch(true)
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
	}

	node.Start()
	msg := p2p.NewMessage(pb.XuperMessage_POSTTX, nil)
	if err := node.SendMessage(nctx.MockOperateCtx(), msg); err != nil {
		t.Errorf("sendMessage error: %v", err)
	}

	msg = p2p.NewMessage(pb.XuperMessage_GET_BLOCK, nil)
	if responses, err := node.SendMessageWithResponse(nctx.MockOperateCtx(), msg); err != nil {
		t.Errorf("sendMessage error: %v", err)
	} else {
		for i, resp := range responses {
			t.Logf("resp[%d]: log_id=%v", i, resp)
		}
	}
}

func TestP2PServerV1(t *testing.T) {
	startNode1(t)
	startNode2(t)
	time.Sleep(time.Second)
	startNode3(t)
}
