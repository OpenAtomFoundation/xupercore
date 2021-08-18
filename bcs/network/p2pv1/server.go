package p2pv1

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/patrickmn/go-cache"
	prom "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/xuperchain/xupercore/kernel/common/xaddress"
	"github.com/xuperchain/xupercore/kernel/network"
	"github.com/xuperchain/xupercore/kernel/network/config"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/def"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/metrics"
	pb "github.com/xuperchain/xupercore/protos"
)

const (
	ServerName = "p2pv1"
)

var (
	ErrAddressIllegal  = errors.New("address illegal")
	ErrLoadAccount     = errors.New("load account error")
	ErrAccountNotExist = errors.New("account not exist")
)

func init() {
	network.Register(ServerName, NewP2PServerV1)
}

// P2PServerV1
type P2PServerV1 struct {
	ctx    *nctx.NetCtx
	log    logs.Logger
	config *config.NetConf

	address    multiaddr.Multiaddr
	pool       *ConnPool
	dispatcher p2p.Dispatcher

	bootNodes   []string
	staticNodes map[string][]string
	dynamicSet  map[string]struct{}
	mutex       sync.RWMutex

	// local host account
	account string
	// accounts store remote peer account: key:account => v:peer.ID
	accounts *cache.Cache
}

var _ p2p.Server = &P2PServerV1{}

// NewP2PServerV1 create P2PServerV1 instance
func NewP2PServerV1() p2p.Server {
	return &P2PServerV1{}
}

func (p *P2PServerV1) dynamicInsert(addr string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if _, ok := p.dynamicSet[addr]; ok {
		return
	}
	p.dynamicSet[addr] = struct{}{}
}

func (p *P2PServerV1) getDynamicNodes() []string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	var nodes []string
	for k, _ := range p.dynamicSet {
		nodes = append(nodes, k)
	}
	return nodes
}

// Init initialize p2p server using given config
func (p *P2PServerV1) Init(ctx *nctx.NetCtx) error {
	pool, err := NewConnPool(ctx)
	if err != nil {
		p.log.Error("Init P2PServerV1 NewConnPool error", "error", err)
		return err
	}

	p.ctx = ctx
	p.log = ctx.GetLog()
	p.config = ctx.P2PConf
	p.pool = pool
	p.dispatcher = p2p.NewDispatcher(ctx)

	// address
	p.address, err = multiaddr.NewMultiaddr(ctx.P2PConf.Address)
	if err != nil {
		log.Printf("network address error: %v", err)
		return ErrAddressIllegal
	}
	// 更新localhost
	_, ip, err := manet.DialArgs(p.address)
	if err != nil {
		log.Printf("network address error: %v", err)
		return ErrAddressIllegal
	}
	p.pool.staticRouterInsert("localhost", ip)
	// account
	keyPath := ctx.EnvCfg.GenDataAbsPath(ctx.EnvCfg.KeyDir)
	p.account, err = xaddress.LoadAddress(keyPath)
	if err != nil {
		p.log.Error("load account error", "path", keyPath)
		return ErrLoadAccount
	}
	p.accounts = cache.New(cache.NoExpiration, cache.NoExpiration)
	p.bootNodes = make([]string, 0)
	p.dynamicSet = make(map[string]struct{})
	return nil
}

func (p *P2PServerV1) Start() {
	p.log.Info("StartP2PServer", "address", p.config.Address)
	p.registerConnectHandler()
	p.connectBootNodes()
	p.connectStaticNodes()
	go p.serve()
}

func (p *P2PServerV1) Stop() {
	p.log.Info("StopP2PServer", "address", p.config.Address)
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

	network, ip, err := manet.DialArgs(p.address)
	if err != nil {
		panic(fmt.Sprintf("address error: address=%s", err))
	}

	l, err := net.Listen(network, ip)
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
	msg, err := stream.Recv()
	if err != nil {
		p.log.Warn("SendP2PMessage Recv msg error", "error", err)
		return err
	}

	if p.ctx.EnvCfg.MetricSwitch {
		tm := time.Now()
		defer func() {
			labels := prom.Labels{
				metrics.LabelBCName:      msg.GetHeader().GetBcname(),
				metrics.LabelMessageType: msg.GetHeader().GetType().String(),
			}
			metrics.NetworkMsgReceivedCounter.With(labels).Inc()
			metrics.NetworkMsgReceivedBytesCounter.With(labels).Add(float64(proto.Size(msg)))
			metrics.NetworkServerHandlingHistogram.With(labels).Observe(time.Since(tm).Seconds())
		}()
	}

	if err = p.dispatcher.Dispatch(msg, stream); err != nil {
		p.log.Warn("handle new message dispatch error", "log_id", msg.GetHeader().GetLogid(),
			"type", msg.GetHeader().GetType(), "from", msg.GetHeader().GetFrom(), "error", err)
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

func (p *P2PServerV1) Context() *nctx.NetCtx {
	return p.ctx
}

func (p *P2PServerV1) PeerInfo() pb.PeerInfo {
	_, ip, err := manet.DialArgs(p.address)
	if err != nil {
		p.log.Warn("address illegal", "error", err)
	}

	peerInfo := pb.PeerInfo{
		Id:      ip,
		Address: ip,
		Account: p.account,
	}

	accounts := p.accounts.Items()
	peerID2Accounts := make(map[string]string, len(accounts))
	for account, item := range accounts {
		if id, ok := item.Object.(string); ok {
			peerID2Accounts[id] = account
		}
	}

	peers := p.pool.GetAll()
	for id, address := range peers {
		if address == ip {
			continue
		}

		remotePeerInfo := &pb.PeerInfo{
			Id:      id,
			Address: address,
			Account: peerID2Accounts[id],
		}
		peerInfo.Peer = append(peerInfo.Peer, remotePeerInfo)
	}

	return peerInfo
}

// connectBootNodes connect to boot node
func (p *P2PServerV1) connectBootNodes() {
	p.bootNodes = p.config.BootNodes
	if len(p.bootNodes) <= 0 {
		p.log.Warn("connectBootNodes error: boot node empty")
		return
	}

	addresses := make([]string, 0, len(p.bootNodes))
	for _, address := range p.bootNodes {
		_, err := p.pool.Get(address)
		if err != nil {
			p.log.Error("connectPeersByAddr error", "address", address, "error", err)
			continue
		}
		addresses = append(addresses, address)
	}
	p.GetPeerInfo(addresses)
}

func (p *P2PServerV1) connectStaticNodes() {
	p.staticNodes = p.config.StaticNodes
	if len(p.pool.staticNodeSet) <= 0 {
		p.log.Warn("connectStaticNodes error: static node empty")
		return
	}

	allAddresses := make([]string, 0, 128)
	for address, _ := range p.pool.staticNodeSet {
		_, err := p.pool.Get(address)
		if err != nil {
			p.log.Warn("p2p connect to peer failed", "address", address, "error", err)
			continue
		}
		allAddresses = append(allAddresses, address)
	}
	// "xuper" blockchain is super set of all blockchains
	if len(p.staticNodes[def.BlockChain]) < len(allAddresses) {
		p.staticNodes[def.BlockChain] = allAddresses
	}
	p.GetPeerInfo(allAddresses)
}
