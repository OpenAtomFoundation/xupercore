package p2pv1

import (
    "context"
    "errors"
    prom "github.com/prometheus/client_golang/prometheus"
    "github.com/xuperchain/xupercore/kernel/network/config"
    nctx "github.com/xuperchain/xupercore/kernel/network/context"
    "github.com/xuperchain/xupercore/kernel/network/p2p"
    pb "github.com/xuperchain/xupercore/kernel/network/pb"
    "github.com/xuperchain/xupercore/lib/logs"
    "github.com/xuperchain/xupercore/lib/timer"
    "google.golang.org/grpc"
    "google.golang.org/grpc/peer"
    "google.golang.org/grpc/reflection"
    "net"
    "strconv"
    "strings"
    "time"
)

var _ p2p.Server = &P2PServerV1{}

// P2PServerV1
type P2PServerV1 struct {
    ctx             nctx.DomainCtx
    log             logs.Logger
    config          *config.Config

    id              string
    pool            *ConnPool
    dispatcher      p2p.Dispatcher

    bootNodes       []string
    staticNodes     map[string][]string
    dynamicNodes    []string
}

// NewP2PServerV1 create P2PServerV1 instance
func NewP2PServerV1() *P2PServerV1 {
    return &P2PServerV1{}
}

// Init initialize p2p server using given config
func (p *P2PServerV1) Init(ctx nctx.DomainCtx) error {
    pool, err := NewConnPool(ctx)
    if err != nil {
        p.log.Error("Init P2PServerV1 NewConnPool error", "error", err)
        return err
    }

    p.ctx = ctx
    p.log = ctx.GetLog()
    p.config = ctx.GetP2PConf()
    p.pool = pool
    p.dispatcher = p2p.NewDispatcher()

    p.connectBootNodes()
    p.connectStaticNodes()
    p.connectDynamicNodes()
    go p.Start()
    return nil
}

func (p *P2PServerV1) Start() {
    p.log.Info("StartP2PServer", "port", p.config.Port)
    p.serve()
}

func (p *P2PServerV1) Stop() {
    p.log.Info("StopP2PServer", "port", p.config.Port)
}

// serve
func (p *P2PServerV1) serve() {
    options := append([]grpc.ServerOption{},
        grpc.MaxRecvMsgSize(int(p.config.MaxMessageSize)<<20),
        grpc.MaxSendMsgSize(int(p.config.MaxMessageSize)<<20),
    )

    if p.config.IsTls {
        creds, err := p2p.NewTLS(p.config.KeyPath, p.config.ServiceName)
        if err != nil {
            panic(err)
        }
        options = append(options, grpc.Creds(creds))
    }

    l, err := net.Listen("tcp", ":"+strconv.Itoa((int)(p.config.Port)))
    if err != nil {
        panic(err)
    }

    server := grpc.NewServer(options...)
    pb.RegisterP2PServiceServer(server, p)
    reflection.Register(server)
    
    if err := server.Serve(l); err != nil {
        panic(err)
    }
}

// SendP2PMessage implement the SendP2PMessageServer
func (p *P2PServerV1) SendP2PMessage(stream pb.P2PService_SendP2PMessageServer) error {
    stream.Context()
    msg, err := stream.Recv()
    if err != nil {
        p.log.Warn("SendP2PMessage Recv msg error", "error", err)
        return err
    }

    if p.ctx.GetMetricSwitch() {
        tm := time.Now()
        defer func() {
            labels := prom.Labels{
                "bcname": msg.GetHeader().GetBcname(),
                "type":   msg.GetHeader().GetType().String(),
                "method": "SendP2PMessage",
            }

            p2p.Metrics.QPS.With(labels).Inc()
            p2p.Metrics.Cost.With(labels).Add(float64(time.Since(tm).Microseconds()))
            // p2p.Metrics.Packet.With(labels).Add(float64(proto.Size(msg)))
        }()
    }

    p.log.Trace("SendP2PMessage", "log_id", msg.GetHeader().GetLogid(), "type", msg.GetHeader().GetType())
    if !strings.Contains(msg.Header.From, ":") {
        ip, _ := getRemoteIP(stream.Context())
        msg.Header.From = ip + ":" + msg.Header.From
    }

    ctx, _ := nctx.CreateOperateCtx(p.ctx.GetLog(), timer.NewXTimer())
    if err = p.dispatcher.Dispatch(ctx, msg, stream); err != nil {
        p.log.Warn("dispatch error", "log_id", msg.GetHeader().GetLogid(), "type", msg.GetHeader().GetType(), "error", err)
        return err
    }
    return nil
}

func (p *P2PServerV1) NewSubscriber(typ pb.XuperMessage_MessageType, v interface{}, opts ...p2p.SubscriberOption) p2p.Subscriber {
    return p2p.NewSubscriber(p.ctx, typ, v, opts...)
}

func (p *P2PServerV1) Register(sub p2p.Subscriber) error {
    return p.dispatcher.Register(sub)
}

func (p *P2PServerV1) UnRegister(sub p2p.Subscriber) error {
    return p.dispatcher.UnRegister(sub)
}

func (p *P2PServerV1) GetMultiAddr() string {
    return ""
}

// connectBootNodes connect to boot node
func (p *P2PServerV1) connectBootNodes() error {
    p.bootNodes = p.config.BootNodes
    p.connectPeerByAddr(p.bootNodes)

    msg := p2p.NewMessage(pb.XuperMessage_NEW_NODE, nil)
    msg.Header.From = strconv.Itoa(int(p.config.Port))
    opts := []p2p.OptionFunc{
        p2p.WithAddresses(p.bootNodes),
    }

    ctx, _ := nctx.CreateOperateCtx(p.ctx.GetLog(), timer.NewXTimer())
    go p.SendMessage(ctx, msg, opts...)
    return nil
}

// connectPeersByAddr establish contact with given nodes
func (p *P2PServerV1) connectPeerByAddr(addresses []string) {
    for _, addr := range addresses {
        _, err := p.pool.Get(addr)
        if err != nil {
            p.log.Error("connectPeersByAddr error", "addr", addr, "error", err)
        }
    }
}

func (p *P2PServerV1) connectStaticNodes() error {
    p.staticNodes = p.config.StaticNodes
    if len(p.staticNodes) <= 0 {
        return nil
    }

    peerIDs := make([]string, 0, 128)
    uniqueAddr := map[string]bool{}
    for _, addresses := range p.staticNodes {
        for _, addr := range addresses {
            if _, ok := uniqueAddr[addr]; ok {
                continue
            }

            _, err := p.pool.Get(addr)
            if err != nil {
                p.log.Warn("p2p connect to peer failed", "addr", addr, "error", err)
                continue
            }

            uniqueAddr[addr] = true
            peerIDs = append(peerIDs, addr)
        }
    }

    // "xuper" blockchain is super set of all blockchains
    if len(p.staticNodes[p2p.BlockChain]) < len(peerIDs) {
        p.staticNodes[p2p.BlockChain] = peerIDs
    }

    return nil
}

func (p *P2PServerV1) connectDynamicNodes() error {
    dynamicNodeChan := make(chan *pb.XuperMessage, 1024)
    err := p.Register(p2p.NewSubscriber(p.ctx, pb.XuperMessage_NEW_NODE, dynamicNodeChan))
    if err != nil {
        p.log.Error("registerSubscribe error", "error", err)
        return err
    }

    go func() {
        for {
            select {
            case msg := <-dynamicNodeChan:
                p.log.Trace("HandleNewNode", "log_id", msg.GetHeader().GetLogid(), "msgType", msg.GetHeader().GetType())
                go p.handleNewNode(msg)
            }
        }
    }()

    return nil
}

func (p *P2PServerV1) handleNewNode(msg *pb.XuperMessage) {
    if msg.GetHeader().GetFrom() == "" {
        return
    }

    for _, peerID := range p.dynamicNodes {
        if peerID == msg.GetHeader().GetFrom() {
            p.log.Warn("P2PServerV1 handleReceivedMsg this dynamicNodes have been added, omit")
            return
        }
    }

    p.dynamicNodes = append(p.dynamicNodes, msg.GetHeader().GetFrom())
}

func getRemoteIP(ctx context.Context) (string, error) {
    var pr, ok = peer.FromContext(ctx)
    if ok && pr.Addr != net.Addr(nil) {
        return strings.Split(pr.Addr.String(), ":")[0], nil
    }

    return "", errors.New("get node addr error")
}
