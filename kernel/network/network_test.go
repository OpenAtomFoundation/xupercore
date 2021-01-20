package network

import (
	"fmt"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"testing"

	"github.com/xuperchain/xupercore/kernel/mock"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/protos"
)

type MockP2PServ struct {
	ctx *nctx.NetCtx
}

func NewMockP2PServ() p2p.Server {
	return &MockP2PServ{}
}

func (t *MockP2PServ) Init(ctx *nctx.NetCtx) error {
	t.ctx = ctx
	return nil
}

func (t *MockP2PServ) Start() {
}

func (t *MockP2PServ) Stop() {
}

func (t *MockP2PServ) NewSubscriber(pb.XuperMessage_MessageType,
	interface{}, ...p2p.SubscriberOption) p2p.Subscriber {

	return nil
}

func (t *MockP2PServ) Register(p2p.Subscriber) error {
	return fmt.Errorf("mock interface")
}

func (t *MockP2PServ) UnRegister(p2p.Subscriber) error {
	return fmt.Errorf("mock interface")
}

func (t *MockP2PServ) SendMessage(xctx.XContext, *pb.XuperMessage, ...p2p.OptionFunc) error {
	return fmt.Errorf("mock interface")
}

func (t *MockP2PServ) SendMessageWithResponse(xctx.XContext,
	*pb.XuperMessage, ...p2p.OptionFunc) ([]*pb.XuperMessage, error) {

	return nil, fmt.Errorf("mock interface")
}

func (t *MockP2PServ) Context() *nctx.NetCtx {
	return t.ctx
}

func (t *MockP2PServ) PeerInfo() pb.PeerInfo {
	return pb.PeerInfo{}
}

func TestNewNetwork(t *testing.T) {
	mock.InitLogForTest()

	Register("p2pv2", NewMockP2PServ)

	ecfg, _ := mock.NewEnvConfForTest()
	netCtx, _ := nctx.NewNetCtx(ecfg)

	n, err := NewNetwork(netCtx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(n)
}
