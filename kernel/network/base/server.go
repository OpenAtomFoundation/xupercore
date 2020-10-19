package base

import (
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	xuperp2p "github.com/xuperchain/xupercore/kernel/network/pb"
)

// P2PServer is the p2p server interface of Xuper
type P2PServer interface {
	// Initialize the p2p server with given config
	Init(octx *nctx.ObjCtx) error
	Start()
	Stop()

	SendMessage(fctx *nctx.FuncCtx, msg *xuperp2p.XuperMessage, opts ...MessageOption) error
	SendMessageWithResponse(fctx *nctx.FuncCtx, msg *xuperp2p.XuperMessage,
		opts ...MessageOption) ([]*xuperp2p.XuperMessage, error)

	GetNetURL() string
	// 查询所连接节点的信息
	GetPeerUrls() []string
	GetPeerIDAndUrls() map[string]string
	// SetCorePeers set core peers' info to P2P server
	SetCorePeers(cp *CorePeersInfo) error
	// SetXchainAddr Set xchain address from xchaincore
	SetXchainAddr(bcname string, info *XchainAddrInfo)

	// NewSubscriber create a subscriber instance
	NewSubscriber(msgChan chan *xuperp2p.XuperMessage, msgType xuperp2p.XuperMessage_MessageType,
		handle XuperHandler, msgFrom string, octx *nctx.ObjCtx) (Subscriber, error)
	// 注册订阅者，支持多个用户订阅同一类消息
	Register(sub Subscriber) (Subscriber, error)
	// 注销订阅者，需要根据当时注册时返回的Subscriber实例删除
	UnRegister(sub Subscriber) error
}
