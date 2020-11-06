package p2pv2

import (
    "bufio"
    "context"
    "errors"
    ggio "github.com/gogo/protobuf/io"
    "github.com/libp2p/go-libp2p-core/network"
    "github.com/libp2p/go-libp2p-core/peer"
    ma "github.com/multiformats/go-multiaddr"
    "github.com/xuperchain/xupercore/kernel/network/config"
    nctx "github.com/xuperchain/xupercore/kernel/network/context"
    "github.com/xuperchain/xupercore/kernel/network/p2p"
    pb "github.com/xuperchain/xupercore/kernel/network/pb"
    "github.com/xuperchain/xupercore/lib/logs"
    "github.com/xuperchain/xupercore/lib/timer"
    "io"
    "sync"
    "time"
)

// define common errors
var (
    ErrNewStream        = errors.New("new stream error")
    ErrTimeout          = errors.New("request time out")
    ErrNullResult       = errors.New("request result is null")
    ErrStreamNotValid   = errors.New("stream not valid")
    ErrNoneMessageType  = errors.New("none message type")
)

// Stream is the IO wrapper for underly P2P connection
type Stream struct {
    ctx     nctx.DomainCtx
    config  *config.Config
    log     logs.Logger
    
    srv     *P2PServerV2
    stream  network.Stream
    streamMu *sync.Mutex
    id      peer.ID
    addr    ma.Multiaddr
    w       *bufio.Writer
    wc      ggio.WriteCloser
    rc      ggio.ReadCloser

    valid   bool
    
    grpcPort string
}

// NewStream create Stream instance
func NewStream(ctx nctx.DomainCtx, srv *P2PServerV2, netStream network.Stream) (*Stream, error) {
    w := bufio.NewWriter(netStream)
    wc := ggio.NewDelimitedWriter(w)
    maxMsgSize := int(ctx.GetP2PConf().MaxMessageSize) << 20
    stream := &Stream{
        ctx:    ctx,
        config: ctx.GetP2PConf(),
        log:    ctx.GetLog(),
        
        srv:    srv,
        stream: netStream,
        streamMu:new(sync.Mutex),
        id:     netStream.Conn().RemotePeer(),
        addr:   netStream.Conn().RemoteMultiaddr(),
        rc:     ggio.NewDelimitedReader(netStream, maxMsgSize),
        w:      w,
        wc:     wc,

        valid:  true,
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
    s.getRPCPort()
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
    s.srv.streamPool.DelStream(s)
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

    s.log.Trace("HandlerNewMessage", "log_id", msg.GetHeader().GetLogid(), "type", msg.GetHeader().GetType(), "from", msg.GetHeader().GetFrom())
    ctx, _ := nctx.CreateOperateCtx(s.ctx.GetLog(), timer.NewXTimer())
    if err := s.srv.dispatcher.Dispatch(ctx, msg, s); err != nil {
        s.log.Warn("Dispatcher", "log_id", msg.GetHeader().GetLogid(), "type", msg.GetHeader().GetType(), "error", err, "from", msg.GetHeader().GetFrom())
        return nil // not return err
    }

    return nil
}

// getRPCPort 刚建立链接的时候获取对方的GPRC端口
func (s *Stream) getRPCPort() {
    msg := p2p.NewMessage(pb.XuperMessage_GET_RPC_PORT, nil)
    ctx, _ := nctx.CreateOperateCtx(s.ctx.GetLog(), timer.NewXTimer())
    resp, err := s.SendMessageWithResponse(ctx, msg)
    if err != nil {
        s.log.Warn("getRPCPort error", "log_id", msg.GetHeader().GetLogid(), "err", err, "resp", resp)
        return
    }
    port := string(resp.GetData().GetMsgInfo())
    s.grpcPort = port
}

// SendMessage will send a message to a peer
func (s *Stream) SendMessage(ctx nctx.OperateCtx, msg *pb.XuperMessage) error {
    s.log.Trace("Stream SendMessage", "log_id", msg.GetHeader().GetLogid(), "msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "to", s.id.Pretty())
    err := s.Send(msg)
    if err != nil {
        s.Close()
        s.log.Error("Stream SendMessage error", "log_id", msg.GetHeader().GetLogid(),
            "msgType", msg.GetHeader().GetType(), "error", err)
        return err
    }

    return nil
}

// SendMessageWithResponse will send a message to a peer and wait for response
func (s *Stream) SendMessageWithResponse(ctx nctx.OperateCtx, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
    respType := p2p.GetRespMessageType(msg.GetHeader().GetType())
    if respType == pb.XuperMessage_MSG_TYPE_NONE {
        return nil, ErrNoneMessageType
    }

    observerCh := make(chan *pb.XuperMessage, 100)
    sub := p2p.NewSubscriber(s.ctx, respType, observerCh, p2p.WithFrom(s.id.Pretty()))
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
    s.log.Trace("Stream SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(), "msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "to", s.id.Pretty())
    if err := s.Send(msg); err != nil {
        s.Close()
        s.log.Warn("SendMessageWithResponse Send error", "log_id", msg.GetHeader().GetLogid(), "msgType", msg.GetHeader().GetType(), "err", err)
        return nil, err
    }

    // 等待返回
    select {
    case resp := <-respCh:
        ctx.GetLog().Trace("SendMessageWithResponse return", "log_id", resp.GetHeader().GetLogid())
        return resp, nil
    case err := <-errCh:
        return nil, err
    }
}

// waitResponse wait resp with timeout
func (s *Stream) waitResponse(ctx context.Context, msg *pb.XuperMessage, observerCh chan *pb.XuperMessage) (*pb.XuperMessage, error) {
    timeout := s.config.Timeout
    ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout) * time.Second)
    defer cancel()

    for {
        select {
        case <-ctx.Done():
            s.log.Warn("waitResponse ctx done", "log_id", msg.GetHeader().GetLogid(), "type", msg.GetHeader().GetType(), "pid", s.id.Pretty(), "error", ctx.Err())
            return nil, ctx.Err()
        case resp := <-observerCh:
            if p2p.VerifyMessageType(msg, resp, s.id.Pretty()) {
                s.log.Trace("waitResponse get resp done", "log_id", resp.GetHeader().GetLogid(), "type", resp.GetHeader().GetType(), "checksum", resp.GetHeader().GetDataCheckSum(), "resp.from", resp.GetHeader().GetFrom())
                return resp, nil
            }
            s.log.Trace("waitResponse get resp continue", "log_id", resp.GetHeader().GetLogid(), "type", resp.GetHeader().GetType(), "checksum", resp.GetHeader().GetDataCheckSum(), "resp.from", resp.GetHeader().GetFrom())
            continue
        }
    }
}