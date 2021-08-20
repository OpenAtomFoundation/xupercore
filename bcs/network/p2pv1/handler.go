package p2pv1

import (
	"errors"
	"sync"

	"github.com/patrickmn/go-cache"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/protos"
)

func (p *P2PServerV1) GetPeerInfo(addresses []string) ([]*pb.PeerInfo, error) {
	if len(addresses) == 0 {
		return nil, errors.New("neighbors empty")
	}
	peerInfo := p.PeerInfo()
	p.accounts.Set(peerInfo.GetAccount(), peerInfo.GetAddress(), 0)

	var remotePeers []*pb.PeerInfo
	var wg sync.WaitGroup
	var mutex sync.Mutex
	for _, addr := range addresses {
		wg.Add(1)
		go func(addr string, mu *sync.Mutex, peers []*pb.PeerInfo) {
			defer wg.Done()
			rps := p.GetPeer(peerInfo, addr)
			if rps == nil {
				return
			}
			mu.Lock()
			peers = append(peers, rps...)
			mu.Unlock()
		}(addr, &mutex, remotePeers)
	}
	wg.Wait()
	return remotePeers, nil
}

func (p *P2PServerV1) GetPeer(peerInfo pb.PeerInfo, addr string) []*pb.PeerInfo {
	var remotePeers []*pb.PeerInfo
	msg := p2p.NewMessage(pb.XuperMessage_GET_PEER_INFO, &peerInfo)
	response, err := p.SendMessageWithResponse(p.ctx, msg, p2p.WithAddresses([]string{addr}))
	if err != nil {
		p.log.Error("get peer error", "log_id", msg.GetHeader().GetLogid(), "error", err)
		return nil
	}
	for _, msg := range response {
		var peer pb.PeerInfo
		err := p2p.Unmarshal(msg, &peer)
		if err != nil {
			p.log.Warn("unmarshal NewNode response error", "log_id", msg.GetHeader().GetLogid(), "error", err)
			continue
		}
		peer.Address = addr
		p.accounts.Set(peer.GetAccount(), peer.GetAddress(), cache.NoExpiration)
		remotePeers = append(remotePeers, &peer)
	}
	return remotePeers
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

	if !p.pool.staticModeOn {
		uniq := make(map[string]struct{}, len(p.dynamicNodes))
		for _, address := range p.dynamicNodes {
			uniq[address] = struct{}{}
		}

		if _, ok := uniq[peerInfo.Address]; !ok {
			p.dynamicNodes = append(p.dynamicNodes, peerInfo.Address)
		}
		p.accounts.Set(peerInfo.GetAccount(), peerInfo.GetAddress(), cache.NoExpiration)
	}

	return resp, nil
}
