package network

import (
	"fmt"
	"sort"
	"sync"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	nbase "github.com/xuperchain/xupercore/kernel/network/base"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
)

// 创建P2PServer实例方法
type NewP2PServFunc func() Network

var (
	servsMu  sync.RWMutex
	services = make(map[string]NewP2PServFunc)
)

// Register makes a driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,it panics.
func Register(name string, f NewP2PServFunc) {
	servsMu.Lock()
	defer servsMu.Unlock()

	if f == nil {
		panic("network: Register new func is nil")
	}
	if _, dup := services[name]; dup {
		panic("network: Register called twice for func " + name)
	}
	services[name] = f
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	servsMu.RLock()
	defer servsMu.RUnlock()
	list := make([]string, 0, len(services))
	for name := range services {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func createP2PServ(name string) nbase.P2PServer {
	servsMu.RLock()
	defer servsMu.RUnlock()

	if f, ok := services[name]; ok {
		return f()
	}

	return nil
}

// network对外提供的接口
type Network interface {
	// send msg
	SendMsg(ctx *xctx.BaseCtx, msg *xuperp2p.XuperMessage, opts ...MessageOption) error
	SendMsgWithResp(ctx *xctx.BaseCtx, msg *xuperp2p.XuperMessage,
		opts ...MessageOption) ([]*xuperp2p.XuperMessage, error)
	// query info
	GetNetURL() string
	GetPeerUrls() []string
	GetPeerIDAndUrls() map[string]string
	SetCorePeers(cp *CorePeersInfo) error
	SetXchainAddr(bcname string, info *XchainAddrInfo)
	// subscriber msg
	NewSubscriber(msgChan chan *xuperp2p.XuperMessage, msgType xuperp2p.XuperMessage_MessageType,
		handle XuperHandler, msgFrom string, octx nctx.ObjCtx) (Subscriber, error)
	Register(sub Subscriber) (Subscriber, error)
	UnRegister(sub Subscriber) error
}

// 如果有领域内公共逻辑，可以在这层扩展，对上层暴露高级接口
// 暂时没有特殊的逻辑，先简单透传，预留方便后续扩展
type NetworkImpl struct {
	netCtx  *nctx.NetCtx
	p2pServ nbase.P2PServer
}

func CreateNetwork(servName string, netCtx *nctx.NetCtx) (Network, error) {
	// check param
	if octx == nil || !octx.IsVaild() {
		return nil, fmt.Errorf("new network failed because context set error")
	}

	// get p2p service
	p2pServ := createP2PServ(servName)
	if p2pServ == nil {
		return nil, fmt.Errorf("new network failed because service not exist.name:%s", servName)
	}
	// init p2p server
	err := p2pServ.Init(netCtx)
	if err != nil {
		return nil, fmt.Errorf("new network failed because init p2p service error.err:%v", err)
	}
	// start p2p server
	p2pServ.Start()

	return &NetworkImpl{netCtx, p2pServ}, nil
}

func (t *NetworkImpl) SendMsg(ctx *xctx.BaseCtx, msg *xuperp2p.XuperMessage,
	opts ...MessageOption) error {

	if !t.isInit() || ctx == nil || msg == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.SendMessage(ctx, msg, opts)
}

func (t *NetworkImpl) SendMsgWithResp(ctx *xctx.BaseCtx, msg *xuperp2p.XuperMessage,
	opts ...MessageOption) ([]*xuperp2p.XuperMessage, error) {

	if !t.isInit() || ctx == nil || msg == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.SendMessageWithResponse(ctx, msg, opts)
}

func (t *NetworkImpl) GetNetURL() string {
	if !t.isInit() {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.GetNetURL()
}

func (t *NetworkImpl) GetPeerUrls() []string {
	if !t.isInit() {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.GetPeerUrls()
}

func (t *NetworkImpl) GetPeerIDAndUrls() map[string]string {
	if !t.isInit() {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.GetPeerIDAndUrls()
}

func (t *NetworkImpl) SetCorePeers(cp *CorePeersInfo) error {
	if !t.isInit() || cp == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.SetCorePeers(cp)
}

func (t *NetworkImpl) SetXchainAddr(bcname string, info *XchainAddrInfo) {
	if !t.isInit() || bcname == "" || info == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.SetXchainAddr(bcname, info)
}

func (t *NetworkImpl) NewSubscriber(msgChan chan *xuperp2p.XuperMessage,
	msgType xuperp2p.XuperMessage_MessageType, handle XuperHandler, msgFrom string,
	netCtx *nctx.NetCtx) (Subscriber, error) {

	if !t.isInit() || msgChan == nil || netCtx == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.NewSubscriber(msgChan, msgType, handle, msgFrom, netCtx)
}

func (t *NetworkImpl) Register(sub Subscriber) (Subscriber, error) {
	if !t.isInit() || sub == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.Register(sub)
}

func (t *NetworkImpl) UnRegister(sub Subscriber) error {
	if !t.isInit() || sub == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.UnRegister(sub)
}

func (t *NetworkImpl) isInit() bool {
	if t.netCtx == nil || t.p2pServ == nil {
		return false
	}

	return true
}
