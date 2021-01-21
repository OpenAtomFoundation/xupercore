package p2pv1

import (
	"errors"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/protos"
)

var (
	ErrEmptyPeer  = errors.New("empty peer")
	ErrNoResponse = errors.New("no response")
)

// SendMessage send message to peers using given filter strategy
func (p *P2PServerV1) SendMessage(ctx xctx.XContext, msg *pb.XuperMessage, optFunc ...p2p.OptionFunc) error {
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
	peerIDs, err := filter.Filter()
	if err != nil {
		p.log.Warn("p2p: filter error", "log_id", msg.GetHeader().GetLogid())
		return errors.New("p2p SendMessage: filter returned error data")
	}

	if len(peerIDs) <= 0 {
		p.log.Warn("SendMessageWithResponse peerID empty", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return ErrEmptyPeer
	}

	p.log.Trace("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
		"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "peerID", peerIDs)
	return p.sendMessage(ctx, msg, peerIDs)
}

func (p *P2PServerV1) sendMessage(ctx xctx.XContext, msg *pb.XuperMessage, peerIDs []string) error {
	wg := sync.WaitGroup{}
	for _, peerID := range peerIDs {
		conn, err := p.pool.Get(peerID)
		if err != nil {
			p.log.Warn("p2p: get conn error",
				"log_id", msg.GetHeader().GetLogid(), "peerID", peerID, "error", err)
			continue
		}

		wg.Add(1)
		go func(conn *Conn) {
			defer wg.Done()
			err = conn.SendMessage(ctx, msg)
			if err != nil {
				p.log.Warn("p2p: SendMessage error",
					"log_id", msg.GetHeader().GetLogid(), "peerID", conn.id, "error", err)
			}
		}(conn)
	}
	wg.Wait()

	return nil
}

// SendMessageWithResponse send message to peers using given filter strategy, expect response from peers
// 客户端再使用该方法请求带返回的消息时，最好带上log_id, 否则会导致收消息时收到不匹配的消息而影响后续的处理
func (p *P2PServerV1) SendMessageWithResponse(ctx xctx.XContext, msg *pb.XuperMessage, optFunc ...p2p.OptionFunc) ([]*pb.XuperMessage, error) {
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
	peerIDs, err := filter.Filter()
	if err != nil {
		p.log.Warn("p2p: filter error", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return nil, errors.New("p2p: SendMessageWithRes: filter returned error data")
	}

	if len(peerIDs) <= 0 {
		p.log.Warn("SendMessageWithResponse peerID empty", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return nil, ErrEmptyPeer
	}

	p.log.Trace("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
		"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "peerID", peerIDs)
	return p.sendMessageWithResponse(ctx, msg, peerIDs, opt.Percent)
}

func (p *P2PServerV1) sendMessageWithResponse(ctx xctx.XContext, msg *pb.XuperMessage, peerIDs []string, percent float32) ([]*pb.XuperMessage, error) {
	wg := sync.WaitGroup{}
	respCh := make(chan *pb.XuperMessage, len(peerIDs))
	for _, peerID := range peerIDs {
		conn, err := p.pool.Get(peerID)
		if err != nil {
			p.log.Warn("p2p: get conn error", "log_id", msg.GetHeader().GetLogid(),
				"peerID", peerID, "error", err)
			continue
		}

		wg.Add(1)
		go func(conn *Conn) {
			defer wg.Done()

			resp, err := conn.SendMessageWithResponse(ctx, msg)
			if err != nil {
				return
			}
			respCh <- resp
		}(conn)
	}
	wg.Wait()

	if len(respCh) <= 0 {
		p.log.Warn("p2p: no response", "log_id", msg.GetHeader().GetLogid())
		return nil, ErrNoResponse
	}

	i := 0
	length := len(respCh)
	threshold := int(float32(len(peerIDs)) * percent)
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

	return response, nil
}

func (p *P2PServerV1) getFilter(msg *pb.XuperMessage, opt *p2p.Option) PeerFilter {
	if len(opt.Filters) <= 0 && len(opt.Addresses) <= 0 &&
		len(opt.PeerIDs) <= 0 && len(opt.Accounts) <= 0 {
		opt.Filters = []p2p.FilterStrategy{p2p.DefaultStrategy}
	}

	bcname := msg.GetHeader().GetBcname()
	filters := opt.Filters
	peerFilters := make([]PeerFilter, 0)
	for _, f := range filters {
		var filter PeerFilter
		switch f {
		default:
			filter = &StaticNodeStrategy{broadcast: p.config.IsBroadCast, srv: p, bcname: bcname}
		}
		peerFilters = append(peerFilters, filter)
	}

	peerIDs := make([]string, 0)
	if len(opt.Addresses) > 0 {
		peerIDs = append(peerIDs, opt.Addresses...)
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

	return NewMultiStrategy(peerFilters, peerIDs)
}

func (p *P2PServerV1) GetPeerIdByAccount(account string) (string, error) {
	if value, ok := p.accounts.Get(account); ok {
		return value.(string), nil
	}

	return "", ErrAccountNotExist
}
