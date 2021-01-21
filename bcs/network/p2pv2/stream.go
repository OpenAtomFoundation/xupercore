package p2pv2

import (
	"bufio"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/network/config"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/logs"
	pb "github.com/xuperchain/xupercore/protos"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// define common errors
var (
	ErrNewStream       = errors.New("new stream error")
	ErrStreamNotValid  = errors.New("stream not valid")
	ErrNoneMessageType = errors.New("none message type")
)

// Stream is the IO wrapper for underly P2P connection
type Stream struct {
	ctx    *nctx.NetCtx
	config *config.NetConf
	log    logs.Logger

	srv      *P2PServerV2
	stream   network.Stream
	streamMu *sync.Mutex
	id       peer.ID
	addr     ma.Multiaddr
	w        *bufio.Writer
	wc       ggio.WriteCloser
	rc       ggio.ReadCloser

	valid bool

	grpcPort string
}

// NewStream create Stream instance
func NewStream(ctx *nctx.NetCtx, srv *P2PServerV2, netStream network.Stream) (*Stream, error) {
	w := bufio.NewWriter(netStream)
	wc := ggio.NewDelimitedWriter(w)
	maxMsgSize := int(ctx.P2PConf.MaxMessageSize) << 20
	stream := &Stream{
		ctx:      ctx,
		config:   ctx.P2PConf,
		log:      ctx.GetLog(),
		srv:      srv,
		stream:   netStream,
		streamMu: new(sync.Mutex),
		id:       netStream.Conn().RemotePeer(),
		addr:     netStream.Conn().RemoteMultiaddr(),
		rc:       ggio.NewDelimitedReader(netStream, maxMsgSize),
		w:        w,
		wc:       wc,
		valid:    true,
	}
	stream.Start()
	return stream, nil
}

// PeerID get id
func (s *Stream) PeerID() peer.ID {
	return s.id
}

// MultiAddr get multi addr
func (s *Stream) MultiAddr() ma.Multiaddr {
	return s.addr
}

// Start used to start
func (s *Stream) Start() {
	go s.Recv()
}

// Close close the connected IO stream
func (s *Stream) Close() {
	s.reset()
}

func (s *Stream) reset() {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	s.resetLockFree()
}

func (s *Stream) resetLockFree() {
	if s.Valid() {
		if s.stream != nil {
			s.stream.Reset()
		}
		s.stream = nil
		s.valid = false
	}
}

func (s *Stream) Valid() bool {
	return s.valid
}

func (s *Stream) Send(msg *pb.XuperMessage) error {
	if !s.Valid() {
		return ErrStreamNotValid
	}
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	msg.Header.From = s.srv.PeerID()
	if err := s.wc.WriteMsg(msg); err != nil {
		s.resetLockFree()
		return err
	}
	return s.w.Flush()
}

// Recv loop to read data from stream
func (s *Stream) Recv() {
	for {
		msg := new(pb.XuperMessage)
		err := s.rc.ReadMsg(msg)
		switch err {
		case io.EOF:
			s.log.Trace("Stream Recv", "error", "io.EOF")
			s.reset()
			return
		case nil:
		default:
			s.log.Trace("Stream Recv error to reset", "error", err)
			s.reset()
			return
		}
		err = s.handlerNewMessage(msg)
		if err != nil {
			s.reset()
			return
		}
		msg = nil
	}
}

// handlerNewMessage handler new message from a peer
func (s *Stream) handlerNewMessage(msg *pb.XuperMessage) error {
	if s.srv.dispatcher == nil {
		s.log.Warn("Stream not ready, omit", "msg", msg)
		return nil
	}

	if err := s.srv.dispatcher.Dispatch(msg, s); err != nil {
		s.log.Warn("handle new message dispatch error", "log_id", msg.GetHeader().GetLogid(),
			"type", msg.GetHeader().GetType(), "from", msg.GetHeader().GetFrom(), "error", err)
		return nil // not return err
	}

	return nil
}

// SendMessage will send a message to a peer
func (s *Stream) SendMessage(ctx xctx.XContext, msg *pb.XuperMessage) error {
	err := s.Send(msg)
	ctx.GetTimer().Mark("write")
	if err != nil {
		s.Close()
		s.log.Error("Stream SendMessage error", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "error", err)
		return err
	}

	return nil
}

// SendMessageWithResponse will send a message to a peer and wait for response
func (s *Stream) SendMessageWithResponse(ctx xctx.XContext,
	msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	respType := p2p.GetRespMessageType(msg.GetHeader().GetType())
	if respType == pb.XuperMessage_MSG_TYPE_NONE {
		return nil, ErrNoneMessageType
	}

	observerCh := make(chan *pb.XuperMessage, 100)
	sub := p2p.NewSubscriber(s.ctx, respType, observerCh, p2p.WithFilterFrom(s.id.Pretty()))
	err := s.srv.dispatcher.Register(sub)
	if err != nil {
		s.log.Error("sendMessageWithResponse register error", "error", err)
		return nil, err
	}
	defer s.srv.dispatcher.UnRegister(sub)

	errCh := make(chan error, 1)
	respCh := make(chan *pb.XuperMessage, 1)
	go func() {
		resp, err := s.waitResponse(ctx, msg, observerCh)
		if resp != nil {
			respCh <- resp
		}
		if err != nil {
			errCh <- err
		}
	}()

	// 开始写消息
	err = s.Send(msg)
	ctx.GetTimer().Mark("write")
	if err != nil {
		s.Close()
		s.log.Warn("SendMessageWithResponse Send error", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "err", err)
		return nil, err
	}

	// 等待返回
	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}

// waitResponse wait resp with timeout
func (s *Stream) waitResponse(ctx xctx.XContext, msg *pb.XuperMessage,
	observerCh chan *pb.XuperMessage) (*pb.XuperMessage, error) {

	timeout := s.config.Timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			s.log.Warn("waitResponse ctx done", "log_id", msg.GetHeader().GetLogid(),
				"type", msg.GetHeader().GetType(), "pid", s.id.Pretty(), "error", timeoutCtx.Err())
			ctx.GetTimer().Mark("wait")
			return nil, timeoutCtx.Err()
		case resp := <-observerCh:
			if p2p.VerifyMessageType(msg, resp, s.id.Pretty()) {
				ctx.GetTimer().Mark("read")
				return resp, nil
			}

			s.log.Debug("waitResponse get resp continue", "log_id", resp.GetHeader().GetLogid(),
				"type", resp.GetHeader().GetType(), "checksum", resp.GetHeader().GetDataCheckSum(),
				"resp.from", resp.GetHeader().GetFrom())
			continue
		}
	}
}
