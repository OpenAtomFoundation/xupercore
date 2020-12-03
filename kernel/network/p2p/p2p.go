package p2p

import (
	"context"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	pb "github.com/xuperchain/xupercore/kernel/network/pb"
)

// P2P is the p2p server interface
type Server interface {
	Init(nctx.DomainCtx) error
	Start()
	Stop()

	NewSubscriber(pb.XuperMessage_MessageType, interface{}, ...SubscriberOption) Subscriber
	Register(Subscriber) error
	UnRegister(Subscriber) error

	SendMessage(context.Context, *pb.XuperMessage, ...OptionFunc) error
	SendMessageWithResponse(context.Context, *pb.XuperMessage, ...OptionFunc) ([]*pb.XuperMessage, error)

	Context() nctx.DomainCtx
	P2PState() *State
}

type State struct {
	PeerId     string
	PeerAddr   string
	RemotePeer map[string]string
}
