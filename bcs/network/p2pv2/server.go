package p2pv2

import (
	"context"
	"errors"
	"fmt"
	ipfsaddr "github.com/ipfs/go-ipfs-addr"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	net "github.com/xuperchain/xupercore/kernel/network"
	"github.com/xuperchain/xupercore/kernel/network/config"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/kernel/network/pb"
	"github.com/xuperchain/xupercore/lib/logs"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ServerName          = "p2pv2"
	protocolID          = "/xuper/2.0.0" // protocol version
	persistAddrFileName = "address"
)

func init() {
	net.Register(ServerName, NewP2PServerV2)
}

var (
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
	ErrConnectCorePeers = errors.New("error to connect to all core peers")
	ErrInvalidParams    = errors.New("invalid params")

	ErrValidateConfig   = errors.New("config not valid")
	ErrCreateNode       = errors.New("create node error")
	ErrCreateHandlerMap = errors.New("create handlerMap error")
)

// P2PServerV2 is the node in the network
type P2PServerV2 struct {
	ctx    nctx.DomainCtx
	log    logs.Logger
	config *config.Config

	id         peer.ID
	host       host.Host
	kdht       *dht.IpfsDHT
	streamPool *StreamPool
	dispatcher p2p.Dispatcher

	cancel context.CancelFunc

	staticNodes map[string][]peer.ID
	// isStorePeers determine whether open isStorePeers
	isStorePeers bool
	p2pDataPath  string
}

var _ p2p.Server = &P2PServerV2{}

// NewP2PServerV2 create P2PServerV2 instance
func NewP2PServerV2() p2p.Server {
	return &P2PServerV2{}
}

// Init initialize p2p server using given config
func (p *P2PServerV2) Init(ctx nctx.DomainCtx) error {
	p.ctx = ctx
	p.log = ctx.GetLog()
	p.config = ctx.GetP2PConf()

	cfg := ctx.GetP2PConf()
	opts, err := genHostOption(cfg)
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

	p.log.Trace("Host", "address", p.GetMultiAddr(), "config", *cfg)

	p.isStorePeers = cfg.IsStorePeers
	p.p2pDataPath = cfg.P2PDataPath
	p.dispatcher = p2p.NewDispatcher()

	if p.kdht, err = dht.New(ctx, ho); err != nil {
		return ErrCreateKadDht
	}

	if p.streamPool, err = NewStreamPool(ctx, p); err != nil {
		return ErrCreateStreamPool
	}

	if !cfg.IsHidden {
		if err = p.kdht.Bootstrap(ctx); err != nil {
			return ErrCreateBootStrap
		}
	}

	var multiAddrs []string
	if p.isStorePeers {
		multiAddrs, err = p.getPeersFromDisk()
		if err != nil {
			p.log.Warn("getPeersFromDisk error", "err", err)
		}
	}
	if len(cfg.BootNodes) > 0 {
		multiAddrs = append(multiAddrs, cfg.BootNodes...)
	}
	for _, ps := range cfg.StaticNodes {
		multiAddrs = append(multiAddrs, ps...)
	}

	success := p.connectPeerByAddress(multiAddrs)
	if success == 0 && len(cfg.BootNodes) != 0 {
		return ErrConnectBootStrap
	}

	// setup static nodes
	setStaticNodes(ctx, p)

	// set broadcast peers limitation
	MaxBroadCastPeers = cfg.MaxBroadcastPeers

	return nil
}

func genHostOption(cfg *config.Config) ([]libp2p.Option, error) {
	muAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.Port))
	opts := []libp2p.Option{
		libp2p.ListenAddrs(muAddr),
		libp2p.EnableRelay(circuit.OptHop),
	}

	if cfg.IsIpv6 {
		muAddr, _ = multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%d", cfg.Port))
		opts = append(opts, libp2p.ListenAddrs(muAddr))
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
		opts = append(opts, libp2p.DefaultSecurity)
	}

	return opts, nil
}

func setStaticNodes(ctx nctx.DomainCtx, p *P2PServerV2) {
	cfg := ctx.GetP2PConf()
	staticNodes := map[string][]peer.ID{}
	for bcname, peers := range cfg.StaticNodes {
		peerIDs := make([]peer.ID, 0, len(peers))
		for _, peerAddr := range peers {
			id, err := p2p.GetIDFromAddr(peerAddr)
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

// Start start the node
func (p *P2PServerV2) Start() {
	p.log.Trace("StartP2PServer")
	p.host.SetStreamHandler(protocolID, p.streamHandler)

	ctx, cancel := context.WithCancel(p.ctx)
	p.cancel = cancel

	t := time.NewTicker(time.Second * 30)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				p.log.Trace("RoutingTable", "size", p.kdht.RoutingTable().Size())
				p.kdht.RoutingTable().Print()
				if p.isStorePeers {
					if err := p.persistPeersToDisk(); err != nil {
						p.log.Warn("persistPeersToDisk failed", "error", err)
					}
				}
			}
		}
	}()
}

func (p *P2PServerV2) streamHandler(netStream network.Stream) {
	if _, err := p.streamPool.NewStream(netStream); err != nil {
		p.log.Warn("new stream error")
	}
}

// Stop stop the node
func (p *P2PServerV2) Stop() {
	p.log.Info("StopP2PServer")
	p.kdht.Close()
	p.host.Close()
	p.cancel()
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

// GetNetURL return net url of the xuper node
// url = /ip4/127.0.0.1/tcp/<port>/p2p/<peer.Id>
func (p *P2PServerV2) GetMultiAddr() string {
	peerInfo := &peer.AddrInfo{
		ID:    p.host.ID(),
		Addrs: p.host.Addrs(),
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

	retry := 5
	success := 0
	for retry > 0 {
		for _, addrInfo := range addrInfos {
			if err := p.host.Connect(p.ctx, addrInfo); err != nil {
				p.log.Error("p2p: connection with peer node error", "error", err)
				continue
			}

			ok, err := p.kdht.RoutingTable().TryAddPeer(addrInfo.ID, true)
			if err != nil {
				p.log.Error("p2p: add peer to routing table error", "remotePeerID", addrInfo.ID, "error", err)
				continue
			}

			success++
			p.log.Info("p2p: connection established", "addrInfo", addrInfo, "RoutingTable", ok)
		}

		if success > 0 {
			break
		}

		retry--
		num := rand.Int63n(10)
		time.Sleep(time.Duration(num) * time.Second)
	}

	return success
}

// persistPeersToDisk persist peers connecting to each other to disk
func (p *P2PServerV2) persistPeersToDisk() error {
	if err := os.MkdirAll(p.p2pDataPath, 0777); err != nil {
		return err
	}

	multiAddrs := p.streamPool.limit.GetStreams()
	if len(multiAddrs) > 0 {
		data := strings.Join(multiAddrs, "\n")
		return ioutil.WriteFile(filepath.Join(p.p2pDataPath, persistAddrFileName), []byte(data), 0700)
	}

	return nil
}

// getPeersFromDisk get peers from disk
func (p *P2PServerV2) getPeersFromDisk() ([]string, error) {
	data, err := ioutil.ReadFile(filepath.Join(p.p2pDataPath, persistAddrFileName))
	if err != nil {
		return nil, err
	}

	multiAddrs := strings.Split(string(data), "\n")
	return multiAddrs, nil
}
