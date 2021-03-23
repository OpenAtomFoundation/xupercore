package p2pv2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xaddress"
	knet "github.com/xuperchain/xupercore/kernel/network"
	"github.com/xuperchain/xupercore/kernel/network/config"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/logs"
	pb "github.com/xuperchain/xupercore/protos"

	ipfsaddr "github.com/ipfs/go-ipfs-addr"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"
	secio "github.com/libp2p/go-libp2p-secio"
	"github.com/multiformats/go-multiaddr"
	"github.com/patrickmn/go-cache"
)

const (
	ServerName = "p2pv2"

	namespace = "xuper"
	retry     = 10
)

func init() {
	knet.Register(ServerName, NewP2PServerV2)
}

var (
	// protocol prefix
	prefix = fmt.Sprintf("/%s", namespace)

	// protocol version
	protocolID = fmt.Sprintf("%s/2.0.0", prefix)

	// MaxBroadCastPeers define the maximum number of common peers to broadcast messages
	MaxBroadCastPeers = 20
)

// define errors
var (
	ErrGenerateOpts     = errors.New("generate host opts error")
	ErrCreateHost       = errors.New("create host error")
	ErrCreateKadDht     = errors.New("create kad dht error")
	ErrCreateStreamPool = errors.New("create stream pool error")
	ErrCreateBootStrap  = errors.New("create bootstrap error pool error")
	ErrConnectBootStrap = errors.New("error to connect to all bootstrap")
	ErrLoadAccount      = errors.New("load account error")
	ErrStoreAccount     = errors.New("dht store account error")
	ErrConnect          = errors.New("connect all boot and static peer error")
)

// P2PServerV2 is the node in the network
type P2PServerV2 struct {
	ctx    *nctx.NetCtx
	log    logs.Logger
	config *config.NetConf

	id         peer.ID
	host       host.Host
	kdht       *dht.IpfsDHT
	streamPool *StreamPool
	dispatcher p2p.Dispatcher

	cancel context.CancelFunc

	staticNodes map[string][]peer.ID

	// local host account
	account string
	// accounts store remote peer account: key:account => v:peer.ID
	// accounts as cache, store in dht
	accounts *cache.Cache
}

var _ p2p.Server = &P2PServerV2{}

// NewP2PServerV2 create P2PServerV2 instance
func NewP2PServerV2() p2p.Server {
	return &P2PServerV2{}
}

// Init initialize p2p server using given config
func (p *P2PServerV2) Init(ctx *nctx.NetCtx) error {
	p.ctx = ctx
	p.log = ctx.GetLog()
	p.config = ctx.P2PConf

	// host
	cfg := ctx.P2PConf
	opts, err := genHostOption(ctx)
	if err != nil {
		p.log.Error("genHostOption error", "error", err)
		return ErrGenerateOpts
	}

	ho, err := libp2p.New(ctx, opts...)
	if err != nil {
		p.log.Error("Create p2p host error", "error", err)
		return ErrCreateHost
	}

	p.id = ho.ID()
	p.host = ho
	p.log.Trace("Host", "address", p.getMultiAddr(p.host.ID(), p.host.Addrs()), "config", *cfg)

	// dht
	dhtOpts := []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.RoutingTableRefreshPeriod(10 * time.Second),
		dht.ProtocolPrefix(protocol.ID(prefix)),
		dht.NamespacedValidator(namespace, &record.NamespacedValidator{
			namespace: blankValidator{},
		}),
	}
	if p.kdht, err = dht.New(ctx, ho, dhtOpts...); err != nil {
		return ErrCreateKadDht
	}

	if !cfg.IsHidden {
		if err = p.kdht.Bootstrap(ctx); err != nil {
			return ErrCreateBootStrap
		}
	}

	keyPath := ctx.EnvCfg.GenDataAbsPath(ctx.EnvCfg.KeyDir)
	p.account, err = xaddress.LoadAddress(keyPath)
	if err != nil {
		return ErrLoadAccount
	}

	p.accounts = cache.New(cache.NoExpiration, cache.NoExpiration)

	// dispatcher
	p.dispatcher = p2p.NewDispatcher(ctx)

	p.streamPool, err = NewStreamPool(ctx, p)
	if err != nil {
		return ErrCreateStreamPool
	}

	// set static nodes
	setStaticNodes(ctx, p)

	// set broadcast peers limitation
	MaxBroadCastPeers = cfg.MaxBroadcastPeers

	if err := p.connect(); err != nil {
		p.log.Error("connect all boot and static peer error")
		return ErrConnect
	}

	return nil
}

func genHostOption(ctx *nctx.NetCtx) ([]libp2p.Option, error) {
	cfg := ctx.P2PConf
	muAddr, err := multiaddr.NewMultiaddr(cfg.Address)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(muAddr),
		libp2p.EnableRelay(circuit.OptHop),
	}

	if cfg.IsNat {
		opts = append(opts, libp2p.NATPortMap())
	}

	if cfg.IsTls {
		priv, err := p2p.GetPemKeyPairFromPath(cfg.KeyPath)
		if err != nil {
			return nil, err
		}
		opts = append(opts, libp2p.Identity(priv))
		opts = append(opts, libp2p.Security(ID, NewTLS(cfg.KeyPath, cfg.ServiceName)))
	} else {
		priv, err := p2p.GetKeyPairFromPath(cfg.KeyPath)
		if err != nil {
			return nil, err
		}
		opts = append(opts, libp2p.Identity(priv))
		opts = append(opts, libp2p.Security(secio.ID, secio.New))
	}

	return opts, nil
}

func setStaticNodes(ctx *nctx.NetCtx, p *P2PServerV2) {
	cfg := ctx.P2PConf
	staticNodes := map[string][]peer.ID{}
	for bcname, peers := range cfg.StaticNodes {
		peerIDs := make([]peer.ID, 0, len(peers))
		for _, peerAddr := range peers {
			id, err := p2p.GetPeerIDByAddress(peerAddr)
			if err != nil {
				p.log.Warn("static node addr error", "peerAddr", peerAddr)
				continue
			}
			peerIDs = append(peerIDs, id)
		}
		staticNodes[bcname] = peerIDs
	}
	p.staticNodes = staticNodes
}

func (p *P2PServerV2) setKdhtValue() {
	// store: account => address
	account := GenAccountKey(p.account)
	address := p.getMultiAddr(p.host.ID(), p.host.Addrs())
	err := p.kdht.PutValue(context.Background(), account, []byte(address))
	if err != nil {
		p.log.Error("dht put account=>address value error", "error", err)
	}

	// store: peer.ID => account
	id := GenPeerIDKey(p.id)
	err = p.kdht.PutValue(context.Background(), id, []byte(p.account))
	if err != nil {
		p.log.Error("dht put id=>account value error", "error", err)
	}
}

// Start start the node
func (p *P2PServerV2) Start() {
	p.log.Trace("StartP2PServer", "address", p.host.Addrs())
	p.host.SetStreamHandler(protocol.ID(protocolID), p.streamHandler)

	p.setKdhtValue()

	ctx, cancel := context.WithCancel(p.ctx)
	p.cancel = cancel

	t := time.NewTicker(time.Second * 180)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				p.log.Trace("RoutingTable", "id", p.host.ID(), "size", p.kdht.RoutingTable().Size())
				// p.kdht.RoutingTable().Print()
			}
		}
	}()
}

func (p *P2PServerV2) connect() error {
	var multiAddrs []string
	if len(p.config.BootNodes) > 0 {
		multiAddrs = append(multiAddrs, p.config.BootNodes...)
	}
	for _, ps := range p.config.StaticNodes {
		multiAddrs = append(multiAddrs, ps...)
	}
	success := p.connectPeerByAddress(multiAddrs)
	if success == 0 && len(p.config.BootNodes) != 0 {
		return ErrConnectBootStrap
	}

	return nil
}

func (p *P2PServerV2) streamHandler(netStream network.Stream) {
	if _, err := p.streamPool.NewStream(p.ctx, netStream); err != nil {
		p.log.Warn("new stream error")
	}
}

// Stop stop the node
func (p *P2PServerV2) Stop() {
	p.log.Info("StopP2PServer")
	p.kdht.Close()
	p.host.Close()
	if p.cancel != nil {
		p.cancel()
	}
}

// PeerID return the peer ID
func (p *P2PServerV2) PeerID() string {
	return p.id.Pretty()
}

func (p *P2PServerV2) NewSubscriber(typ pb.XuperMessage_MessageType, v interface{}, opts ...p2p.SubscriberOption) p2p.Subscriber {
	return p2p.NewSubscriber(p.ctx, typ, v, opts...)
}

// Register register message subscriber to handle messages
func (p *P2PServerV2) Register(sub p2p.Subscriber) error {
	return p.dispatcher.Register(sub)
}

// UnRegister remove message subscriber
func (p *P2PServerV2) UnRegister(sub p2p.Subscriber) error {
	return p.dispatcher.UnRegister(sub)
}

func (p *P2PServerV2) Context() *nctx.NetCtx {
	return p.ctx
}

func (p *P2PServerV2) PeerInfo() pb.PeerInfo {
	peerInfo := pb.PeerInfo{
		Id:      p.host.ID().Pretty(),
		Address: p.getMultiAddr(p.host.ID(), p.host.Addrs()),
		Account: p.account,
	}

	peerStore := p.host.Peerstore()
	for _, peerID := range p.kdht.RoutingTable().ListPeers() {
		key := GenPeerIDKey(peerID)
		account, err := p.kdht.GetValue(context.Background(), key)
		if err != nil {
			p.log.Warn("get account error", "peerID", peerID)
		}

		addrInfo := peerStore.PeerInfo(peerID)
		remotePeerInfo := &pb.PeerInfo{
			Id:      peerID.String(),
			Address: p.getMultiAddr(addrInfo.ID, addrInfo.Addrs),
			Account: string(account),
		}
		peerInfo.Peer = append(peerInfo.Peer, remotePeerInfo)
	}

	return peerInfo
}

func (p *P2PServerV2) getMultiAddr(peerID peer.ID, addrs []multiaddr.Multiaddr) string {
	peerInfo := &peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}

	multiAddrs, err := peer.AddrInfoToP2pAddrs(peerInfo)
	if err != nil {
		p.log.Warn("gen multi addr error", "peerID", p.host.ID(), "addr", p.host.Addrs())
	}

	if len(multiAddrs) >= 1 {
		return multiAddrs[0].String()
	}

	return ""
}

// ConnectPeerByAddress provide connection support using peer address(netURL)
func (p *P2PServerV2) connectPeerByAddress(addresses []string) int {
	return p.connectPeer(p.getAddrInfos(addresses))
}

func (p *P2PServerV2) getAddrInfos(addresses []string) []peer.AddrInfo {
	addrInfos := make([]peer.AddrInfo, 0, len(addresses))
	for _, addr := range addresses {
		peerAddr, err := ipfsaddr.ParseString(addr)
		if err != nil {
			p.log.Error("p2p: parse peer address error", "peerAddr", peerAddr, "error", err)
			continue
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(peerAddr.Multiaddr())
		if err != nil {
			p.log.Error("p2p: get peer node info error", "peerAddr", peerAddr, "error", err)
			continue
		}

		addrInfos = append(addrInfos, *addrInfo)
	}

	return addrInfos
}

// connectPeer connect to given peers, return the connected number of peers
// only retry if all connection failed
func (p *P2PServerV2) connectPeer(addrInfos []peer.AddrInfo) int {
	if len(addrInfos) <= 0 {
		return 0
	}

	retry := retry
	success := 0
	for retry > 0 {
		for _, addrInfo := range addrInfos {
			if err := p.host.Connect(p.ctx, addrInfo); err != nil {
				p.log.Error("p2p: connection with peer node error", "error", err)
				continue
			}

			success++
			p.log.Info("p2p: connection established", "addrInfo", addrInfo)
		}

		if success > 0 {
			break
		}

		retry--
		time.Sleep(3 * time.Second)
	}

	return success
}
