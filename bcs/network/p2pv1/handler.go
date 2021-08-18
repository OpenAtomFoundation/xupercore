package p2pv1

import (
	"time"

	"github.com/patrickmn/go-cache"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/protos"
)

const (
	MAX_TRIES      = 3
	SLEEP_INTERNAL = 3 * time.Second
)

func (p *P2PServerV1) GetPeerInfo(addresses []string) {
	peerInfo := p.PeerInfo()
	p.accounts.Set(peerInfo.GetAccount(), peerInfo.GetAddress(), 0)
	for _, addr := range addresses {
		go p.GetPeer(peerInfo, addr)
	}
}

func (p *P2PServerV1) GetPeer(peerInfo pb.PeerInfo, addr string) {
	f := func(peerInfo pb.PeerInfo, addr string) error {
		msg := p2p.NewMessage(pb.XuperMessage_GET_PEER_INFO, &peerInfo)
		response, err := p.SendMessageWithResponse(p.ctx, msg, p2p.WithAddresses([]string{addr}))
		if err != nil {
			p.log.Error("get peer error", "log_id", msg.GetHeader().GetLogid(), "error", err)
			return err
		}
		for _, msg := range response {
			var peer pb.PeerInfo
			err := p2p.Unmarshal(msg, &peer)
			if err != nil {
				p.log.Error("unmarshal NewNode response error", "log_id", msg.GetHeader().GetLogid(), "error", err)
				return err
			}
			// 更新路由表
			p.log.Trace("connect static node", "log_id", msg.GetHeader().GetLogid(), "peer", peer.Address, "next", addr)
			if peer.GetAddress() == peerInfo.Address && addr == peerInfo.Address {
				continue
			}
			p.pool.staticRouterInsert(peer.GetAddress(), addr)
			p.accounts.Set(peer.GetAccount(), peer.GetAddress(), 0)
			// 更新动态节点
			p.dynamicInsert(peer.Address)
		}
		return nil
	}
	try := 0
	for try <= MAX_TRIES {
		err := f(peerInfo, addr)
		if err != nil {
			try++
			time.Sleep(SLEEP_INTERNAL)
			continue
		}
		return
	}
}

func (p *P2PServerV1) registerConnectHandler() error {
	err := p.Register(p2p.NewSubscriber(p.ctx, pb.XuperMessage_GET_PEER_INFO, p2p.HandleFunc(p.handleGetPeerInfo)))
	if err != nil {
		p.log.Error("registerSubscribe error", "error", err)
		return err
	}

	return nil
}

func (p *P2PServerV1) handleGetPeerInfo(ctx xctx.XContext, request *pb.XuperMessage) (*pb.XuperMessage, error) {
	output := p.PeerInfo()
	opts := []p2p.MessageOption{
		p2p.WithBCName(request.GetHeader().GetBcname()),
		p2p.WithErrorType(pb.XuperMessage_SUCCESS),
		p2p.WithLogId(request.GetHeader().GetLogid()),
	}
	resp := p2p.NewMessage(pb.XuperMessage_GET_PEER_INFO_RES, &output, opts...)

	var peerInfo pb.PeerInfo
	err := p2p.Unmarshal(request, &peerInfo)
	if err != nil {
		p.log.Warn("unmarshal NewNode response error", "error", err)
		return resp, nil
	}

	p.dynamicInsert(peerInfo.Address)
	p.accounts.Set(peerInfo.GetAccount(), peerInfo.GetAddress(), cache.NoExpiration)
	// TODO: 后续其余节点变更时，需要再次更新本地路由表
	return resp, nil
}
