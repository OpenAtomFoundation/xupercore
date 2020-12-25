package p2p

import (
	"context"

	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	pb "github.com/xuperchain/xupercore/protos"
)

// P2P is the p2p server interface
type Server interface {
	Init(*nctx.NetCtx) error
	Start()
	Stop()

	NewSubscriber(pb.XuperMessage_MessageType, interface{}, ...SubscriberOption) Subscriber
	Register(Subscriber) error
	UnRegister(Subscriber) error

	SendMessage(context.Context, *pb.XuperMessage, ...OptionFunc) error
	SendMessageWithResponse(context.Context, *pb.XuperMessage, ...OptionFunc) ([]*pb.XuperMessage, error)

	Context() *nctx.NetCtx

	PeerInfo() pb.PeerInfo
}

type State struct {
	Local  pb.PeerInfo
	Remote []pb.PeerInfo
}
