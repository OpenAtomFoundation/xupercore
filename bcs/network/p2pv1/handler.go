package p2pv1

import (
	"github.com/patrickmn/go-cache"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/protos"
)

func (p *P2PServerV1) GetPeerInfo(addresses []string) ([]*pb.PeerInfo, error) {
	peerInfo := p.PeerInfo()
	msg := p2p.NewMessage(pb.XuperMessage_GET_PEER_INFO, &peerInfo)
	response, err := p.SendMessageWithResponse(p.ctx, msg, p2p.WithAddresses(addresses))
	if err != nil {
		p.log.Error("get peer error", "log_id", msg.GetHeader().GetLogid(), "error", err)
		return nil, err
	}

	remotePeers := make([]*pb.PeerInfo, 0, len(response))
	for _, msg := range response {
		var peerInfo pb.PeerInfo
		err := p2p.Unmarshal(msg, &peerInfo)
		if err != nil {
			p.log.Warn("unmarshal NewNode response error", "log_id", msg.GetHeader().GetLogid(), "error", err)
			continue
		}

		remotePeers = append(remotePeers, &peerInfo)
	}

	return remotePeers, nil
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

	// 动态节点
	uniq := make(map[string]struct{}, len(p.dynamicNodes))
	for _, address := range p.dynamicNodes {
		uniq[address] = struct{}{}
	}

	if _, ok := uniq[peerInfo.Address]; !ok {
		p.dynamicNodes = append(p.dynamicNodes, peerInfo.Address)
	}
	p.accounts.Set(peerInfo.GetAccount(), peerInfo.GetAddress(), cache.NoExpiration)

	return resp, nil
}
