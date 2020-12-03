package p2pv2

import (
	"errors"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/xuperchain/xuperchain/core/common"

	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/lib/logs"
)

// define common errors
var (
	ErrStreamPoolFull = errors.New("stream pool is full")
)

// StreamPool manage all the stream
type StreamPool struct {
	ctx nctx.DomainCtx
	log logs.Logger

	srv            *P2PServerV2
	limit          *StreamLimit
	streams        *common.LRUCache // key: peer id, value: Stream
	maxStreamLimit int32
}

// NewStreamPool create StreamPool instance
func NewStreamPool(ctx nctx.DomainCtx, srv *P2PServerV2) (*StreamPool, error) {
	cfg := ctx.GetP2PConf()
	limit := &StreamLimit{}
	limit.Init(ctx)
	return &StreamPool{
		ctx: ctx,
		log: ctx.GetLog(),

		srv:            srv,
		limit:          limit,
		streams:        common.NewLRUCache(int(cfg.MaxStreamLimits)),
		maxStreamLimit: cfg.MaxStreamLimits,
	}, nil
}

// Get will probe and return a stream
func (sp *StreamPool) Get(peerId peer.ID) (*Stream, error) {
	if v, ok := sp.streams.Get(peerId.Pretty()); ok {
		if stream, ok := v.(*Stream); ok {
			if stream.Valid() {
				return stream, nil
			} else {
				sp.DelStream(stream)
				sp.log.Warn("stream not valid, create new stream", "peerId", peerId)
			}
		}
	}

	netStream, err := sp.srv.host.NewStream(sp.ctx, peerId, protocolID)
	if err != nil {
		sp.log.Warn("new net stream error", "peerId", peerId, "error", err)
		return nil, ErrNewStream
	}

	return sp.NewStream(netStream)
}

// Add used to add a new net stream into pool
func (sp *StreamPool) NewStream(netStream network.Stream) (*Stream, error) {
	stream, err := NewStream(sp.ctx, sp.srv, netStream)
	if err != nil {
		return nil, err
	}

	if err := sp.AddStream(stream); err != nil {
		stream.Close()
		sp.srv.kdht.RoutingTable().RemovePeer(stream.PeerID())
		sp.log.Warn("New stream is deleted", "error", err)
		return nil, ErrNewStream
	}

	return stream, nil
}

// AddStream used to add a new P2P stream into pool
func (sp *StreamPool) AddStream(stream *Stream) error {
	peerID := stream.PeerID()
	multiAddr := stream.MultiAddr()
	ok := sp.limit.AddStream(multiAddr.String(), peerID)
	if !ok || int32(sp.streams.Len()) > sp.maxStreamLimit {
		sp.log.Warn("add stream limit error", "peerID", peerID, "multiAddr", multiAddr, "error", "over limit")
		return ErrStreamPoolFull
	}

	if v, ok := sp.streams.Get(stream.id.Pretty()); ok {
		sp.log.Warn("replace stream", "peerID", peerID, "multiAddr", multiAddr)
		if s, ok := v.(*Stream); ok {
			sp.DelStream(s)
		}
	}

	sp.streams.Add(stream.id.Pretty(), stream)
	return nil
}

// DelStream delete a stream
func (sp *StreamPool) DelStream(stream *Stream) error {
	stream.Close()
	sp.streams.Del(stream.PeerID().Pretty())
	sp.limit.DelStream(stream.MultiAddr().String())
	return nil
}
