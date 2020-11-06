package p2p

import (
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

	SendMessage(nctx.OperateCtx, *pb.XuperMessage, ...OptionFunc) error
	SendMessageWithResponse(nctx.OperateCtx, *pb.XuperMessage, ...OptionFunc) ([]*pb.XuperMessage, error)

	//P2PState(nctx.OperateCtx) State
	GetMultiAddr() string
}

type State struct {
	PeerId string
	PeerAddr string
	RemotePeer map[string]string
}
