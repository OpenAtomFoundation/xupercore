package p2pv2

import (
	"errors"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/lib/cache"
	"github.com/xuperchain/xupercore/lib/logs"
)

// define common errors
var (
	ErrStreamPoolFull = errors.New("stream pool is full")
)

// StreamPool manage all the stream
type StreamPool struct {
	ctx            *nctx.NetCtx
	log            logs.Logger
	srv            *P2PServerV2
	limit          *StreamLimit
	mutex          sync.Mutex
	streams        *cache.LRUCache // key: peer id, value: Stream
	maxStreamLimit int32
}

// NewStreamPool create StreamPool instance
func NewStreamPool(ctx *nctx.NetCtx, srv *P2PServerV2) (*StreamPool, error) {
	cfg := ctx.P2PConf
	limit := &StreamLimit{}
	limit.Init(ctx)
	return &StreamPool{
		ctx: ctx,
		log: ctx.GetLog(),

		srv:            srv,
		limit:          limit,
		mutex:          sync.Mutex{},
		streams:        cache.NewLRUCache(int(cfg.MaxStreamLimits)),
		maxStreamLimit: cfg.MaxStreamLimits,
	}, nil
}

// Get will probe and return a stream
func (sp *StreamPool) Get(ctx xctx.XContext, peerId peer.ID) (*Stream, error) {
	if v, ok := sp.streams.Get(peerId.Pretty()); ok {
		if stream, ok := v.(*Stream); ok {
			if stream.Valid() {
				return stream, nil
			} else {
				sp.DelStream(stream)
				ctx.GetLog().Warn("stream not valid, create new stream", "peerId", peerId)
			}
		}
	}

	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	if v, ok := sp.streams.Get(peerId.Pretty()); ok {
		if stream, ok := v.(*Stream); ok {
			if stream.Valid() {
				return stream, nil
			} else {
				sp.DelStream(stream)
				ctx.GetLog().Warn("stream not valid, create new stream", "peerId", peerId)
			}
		}
	}

	netStream, err := sp.srv.host.NewStream(sp.ctx, peerId, protocol.ID(protocolID))
	if err != nil {
		ctx.GetLog().Warn("new net stream error", "peerId", peerId, "error", err)
		return nil, ErrNewStream
	}

	return sp.NewStream(ctx, netStream)
}

// Add used to add a new net stream into pool
func (sp *StreamPool) NewStream(ctx xctx.XContext, netStream network.Stream) (*Stream, error) {
	stream, err := NewStream(sp.ctx, sp.srv, netStream)
	if err != nil {
		return nil, err
	}

	if err := sp.AddStream(ctx, stream); err != nil {
		stream.Close()
		sp.srv.kdht.RoutingTable().RemovePeer(stream.PeerID())
		ctx.GetLog().Warn("New stream is deleted", "error", err)
		return nil, ErrNewStream
	}

	return stream, nil
}

// AddStream used to add a new P2P stream into pool
func (sp *StreamPool) AddStream(ctx xctx.XContext, stream *Stream) error {
	peerID := stream.PeerID()
	multiAddr := stream.MultiAddr()
	ok := sp.limit.AddStream(multiAddr.String(), peerID)
	if !ok || int32(sp.streams.Len()) > sp.maxStreamLimit {
		ctx.GetLog().Warn("add stream limit error", "peerID", peerID, "multiAddr", multiAddr, "error", "over limit")
		return ErrStreamPoolFull
	}

	if v, ok := sp.streams.Get(stream.id.Pretty()); ok {
		ctx.GetLog().Warn("replace stream", "peerID", peerID, "multiAddr", multiAddr)
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
