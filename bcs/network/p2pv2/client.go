package p2pv2

import (
	"context"
	"errors"
	"fmt"
	"github.com/patrickmn/go-cache"
	"sync"
	"time"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/timer"
	pb "github.com/xuperchain/xupercore/protos"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	ErrEmptyPeer  = errors.New("empty peer")
	ErrNoResponse = errors.New("no response")
)

// SendMessage send message to peers using given filter strategy
func (p *P2PServerV2) SendMessage(ctx xctx.XContext, msg *pb.XuperMessage,
	optFunc ...p2p.OptionFunc) error {
	ctx = &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
	defer func() {
		ctx.GetLog().Info("SendMessage", "log_id", msg.GetHeader().GetLogid(),
			"bcname", msg.GetHeader().GetBcname(), "msgType", msg.GetHeader().GetType(),
			"checksum", msg.GetHeader().GetDataCheckSum(), "timer", ctx.GetTimer().Print())
	}()

	if p.ctx.EnvCfg.MetricSwitch {
		tm := time.Now()
		defer func() {
			labels := prom.Labels{
				"bcname": msg.GetHeader().GetBcname(),
				"type":   msg.GetHeader().GetType().String(),
				"method": "SendMessage",
			}

			p2p.Metrics.QPS.With(labels).Inc()
			p2p.Metrics.Cost.With(labels).Add(float64(time.Since(tm).Microseconds()))
			p2p.Metrics.Packet.With(labels).Add(float64(proto.Size(msg)))
		}()
	}

	opt := p2p.Apply(optFunc)
	filter := p.getFilter(msg, opt)
	peers, _ := filter.Filter()
	ctx.GetTimer().Mark("filter")

	var peerIDs []peer.ID
	whiteList := opt.WhiteList
	if len(whiteList) > 0 {
		for _, v := range peers {
			if _, exist := whiteList[v.Pretty()]; exist {
				peerIDs = append(peerIDs, v)
			}
		}
	} else {
		peerIDs = peers
	}

	if len(peerIDs) <= 0 {
		p.log.Warn("SendMessageWithResponse peerID empty", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return ErrEmptyPeer
	}

	ctx.GetLog().SetInfoField("peerCount", len(peerIDs))
	return p.sendMessage(ctx, msg, peerIDs)
}

func (p *P2PServerV2) sendMessage(ctx xctx.XContext, msg *pb.XuperMessage, peerIDs []peer.ID) error {
	var wg sync.WaitGroup
	for _, peerID := range peerIDs {
		wg.Add(1)

		go func(peerID peer.ID) {
			streamCtx := &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
			defer func() {
				wg.Done()
				streamCtx.GetLog().Debug("SendMessage", "log_id", msg.GetHeader().GetLogid(),
					"bcname", msg.GetHeader().GetBcname(), "msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(),
					"peer", peerID, "timer", streamCtx.GetTimer().Print())
			}()

			stream, err := p.streamPool.Get(ctx, peerID)
			streamCtx.GetTimer().Mark("connect")
			if err != nil {
				p.log.Warn("p2p: get stream error", "log_id", msg.GetHeader().GetLogid(),
					"msgType", msg.GetHeader().GetType(), "error", err)
				return
			}

			if err := stream.SendMessage(streamCtx, msg); err != nil {
				p.log.Error("SendMessage error", "log_id", msg.GetHeader().GetLogid(),
					"msgType", msg.GetHeader().GetType(), "error", err)
				return
			}
		}(peerID)
	}
	wg.Wait()
	ctx.GetTimer().Mark("send")
	return nil
}

// SendMessageWithResponse send message to peers using given filter strategy, expect response from peers
// 客户端再使用该方法请求带返回的消息时，最好带上log_id, 否则会导致收消息时收到不匹配的消息而影响后续的处理
func (p *P2PServerV2) SendMessageWithResponse(ctx xctx.XContext, msg *pb.XuperMessage,
	optFunc ...p2p.OptionFunc) ([]*pb.XuperMessage, error) {
	ctx = &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
	defer func() {
		ctx.GetLog().Info("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
			"bcname", msg.GetHeader().GetBcname(), "msgType", msg.GetHeader().GetType(),
			"checksum", msg.GetHeader().GetDataCheckSum(), "timer", ctx.GetTimer().Print())
	}()

	if p.ctx.EnvCfg.MetricSwitch {
		tm := time.Now()
		defer func() {
			labels := prom.Labels{
				"bcname": msg.GetHeader().GetBcname(),
				"type":   msg.GetHeader().GetType().String(),
				"method": "SendMessageWithResponse",
			}

			p2p.Metrics.QPS.With(labels).Inc()
			p2p.Metrics.Cost.With(labels).Add(float64(time.Since(tm).Microseconds()))
			p2p.Metrics.Packet.With(labels).Add(float64(proto.Size(msg)))
		}()
	}

	opt := p2p.Apply(optFunc)
	filter := p.getFilter(msg, opt)
	peers, _ := filter.Filter()

	var peerIDs []peer.ID
	// 做一层过滤(基于白名单过滤)
	whiteList := opt.WhiteList
	if len(whiteList) > 0 {
		for _, v := range peers {
			if _, exist := whiteList[v.Pretty()]; exist {
				peerIDs = append(peerIDs, v)
			}
		}
	} else {
		peerIDs = peers
	}
	ctx.GetTimer().Mark("filter")

	if len(peerIDs) <= 0 {
		p.log.Warn("SendMessageWithResponse peerID empty", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return nil, ErrEmptyPeer
	}

	ctx.GetLog().SetInfoField("peerCount", len(peerIDs))
	return p.sendMessageWithResponse(ctx, msg, peerIDs, opt)
}

func (p *P2PServerV2) sendMessageWithResponse(ctx xctx.XContext, msg *pb.XuperMessage,
	peerIDs []peer.ID, opt *p2p.Option) ([]*pb.XuperMessage, error) {

	respCh := make(chan *pb.XuperMessage, len(peerIDs))
	var wg sync.WaitGroup
	for _, peerID := range peerIDs {
		wg.Add(1)
		go func(peerID peer.ID) {
			streamCtx := &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
			defer func() {
				wg.Done()
				streamCtx.GetLog().Debug("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
					"bcname", msg.GetHeader().GetBcname(), "msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(),
					"peer", peerID, "timer", streamCtx.GetTimer().Print())
			}()

			stream, err := p.streamPool.Get(ctx, peerID)
			streamCtx.GetTimer().Mark("connect")
			if err != nil {
				p.log.Warn("p2p: get stream error", "log_id", msg.GetHeader().GetLogid(),
					"msgType", msg.GetHeader().GetType(), "error", err)
				return
			}

			resp, err := stream.SendMessageWithResponse(streamCtx, msg)
			if err != nil {
				p.log.Warn("p2p: SendMessageWithResponse error", "log_id", msg.GetHeader().GetLogid(),
					"msgType", msg.GetHeader().GetType(), "error", err)
				return
			}

			respCh <- resp
		}(peerID)
	}
	wg.Wait()
	ctx.GetTimer().Mark("send")

	if len(respCh) <= 0 {
		p.log.Warn("p2p: no response", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType())
		return nil, ErrNoResponse
	}

	i := 0
	length := len(respCh)
	threshold := int(float32(len(peerIDs)) * opt.Percent)
	response := make([]*pb.XuperMessage, 0, len(peerIDs))
	for resp := range respCh {
		if p2p.VerifyChecksum(resp) {
			response = append(response, resp)
		}

		i++
		if i >= length || len(response) >= threshold {
			break
		}
	}

	ctx.GetTimer().Mark("recv")
	return response, nil
}

func (p *P2PServerV2) getFilter(msg *pb.XuperMessage, opt *p2p.Option) PeerFilter {
	if len(opt.Filters) <= 0 && len(opt.Addresses) <= 0 &&
		len(opt.PeerIDs) <= 0 && len(opt.Accounts) <= 0 {
		opt.Filters = []p2p.FilterStrategy{p2p.DefaultStrategy}
	}

	bcname := msg.GetHeader().GetBcname()
	if len(p.getStaticNodes(bcname)) != 0 {
		return &StaticNodeStrategy{srv: p, bcname: bcname}
	}

	peerFilters := make([]PeerFilter, 0)
	for _, strategy := range opt.Filters {
		var filter PeerFilter
		switch strategy {
		case p2p.NearestBucketStrategy:
			filter = &NearestBucketFilter{srv: p}
		case p2p.BucketsWithFactorStrategy:
			filter = &BucketsFilterWithFactor{srv: p}
		default:
			filter = &BucketsFilter{srv: p}
		}
		peerFilters = append(peerFilters, filter)
	}

	peerIDs := make([]peer.ID, 0)
	if len(opt.Addresses) > 0 {
		go p.connectPeerByAddress(opt.Addresses)
		for _, addr := range opt.Addresses {
			peerID, err := p2p.GetPeerIDByAddress(addr)
			if err != nil {
				p.log.Warn("p2p: getFilter parse peer address failed", "paddr", addr, "error", err)
				continue
			}
			peerIDs = append(peerIDs, peerID)
		}
	}

	if len(opt.Accounts) > 0 {
		for _, account := range opt.Accounts {
			peerID, err := p.GetPeerIdByAccount(account)
			if err != nil {
				p.log.Warn("p2p: getFilter get peer id by account failed", "account", account, "error", err)
				continue
			}
			peerIDs = append(peerIDs, peerID)
		}
	}

	if len(opt.PeerIDs) > 0 {
		for _, encodedPeerID := range opt.PeerIDs {
			peerID, err := peer.Decode(encodedPeerID)
			if err != nil {
				p.log.Warn("p2p: getFilter parse peer ID failed", "pid", peerID, "error", err)
				continue
			}
			peerIDs = append(peerIDs, peerID)
		}
	}
	return NewMultiStrategy(peerFilters, peerIDs)
}

// GetStaticNodes get StaticNode a chain
func (p *P2PServerV2) getStaticNodes(bcname string) []peer.ID {
	return p.staticNodes[bcname]
}

func (p *P2PServerV2) GetPeerIdByAccount(account string) (peer.ID, error) {
	if value, ok := p.accounts.Get(account); ok {
		return value.(peer.ID), nil
	}

	key := Key(account)
	value, err := p.kdht.GetValue(context.Background(), key)
	if err != nil {
		return "", fmt.Errorf("dht get peer id error: %s", err)
	}

	peerID, err := p2p.GetPeerIDByAddress(string(value))
	if err != nil {
		return "", fmt.Errorf("address error: %s, address=%s", err, value)
	}

	p.accounts.Set(key, peerID, cache.NoExpiration)
	return peerID, nil
}
