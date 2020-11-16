package network

import (
	"fmt"
	"sort"
	"sync"

	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/kernel/network/pb"
)

// 创建P2PServer实例方法
type NewP2PServFunc func() p2p.Server

var (
	servMu   sync.RWMutex
	services = make(map[string]NewP2PServFunc)
)

// Register makes a driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,it panics.
func Register(name string, f NewP2PServFunc) {
	servMu.Lock()
	defer servMu.Unlock()

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
	servMu.RLock()
	defer servMu.RUnlock()
	list := make([]string, 0, len(services))
	for name := range services {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func createP2PServ(name string) p2p.Server {
	servMu.RLock()
	defer servMu.RUnlock()

	if f, ok := services[name]; ok {
		return f()
	}

	return nil
}

// network对外提供的接口
type Network interface {
	SendMessage(nctx.OperateCtx, *pb.XuperMessage, ...p2p.OptionFunc) error
	SendMessageWithResponse(nctx.OperateCtx, *pb.XuperMessage, ...p2p.OptionFunc) ([]*pb.XuperMessage, error)

	//NewSubscriber(context.Context, pb.XuperMessage_MessageType, ...SubscriberOption) Subscriber
	Register(p2p.Subscriber) error
	UnRegister(p2p.Subscriber) error

	P2PState(nctx.OperateCtx) p2p.State
}

// 如果有领域内公共逻辑，可以在这层扩展，对上层暴露高级接口
// 暂时没有特殊的逻辑，先简单透传，预留方便后续扩展
type NetworkImpl struct {
	ctx     nctx.DomainCtx
	p2pServ p2p.Server
}

func CreateNetwork(ctx nctx.DomainCtx, servName string) (Network, error) {
	// check param
	if ctx == nil || !ctx.IsValid() {
		return nil, fmt.Errorf("new network failed because context set error")
	}

	// get p2p service
	p2pServ := createP2PServ(servName)
	if p2pServ == nil {
		return nil, fmt.Errorf("new network failed because service not exist.name:%s", servName)
	}
	// init p2p server
	err := p2pServ.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("new network failed because init p2p service error.err:%v", err)
	}
	// start p2p server
	p2pServ.Start()

	return &NetworkImpl{ctx, p2pServ}, nil
}

func (t *NetworkImpl) SendMessage(ctx nctx.OperateCtx, msg *pb.XuperMessage, opts ...p2p.OptionFunc) error {
	if !t.isInit() || ctx == nil || msg == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.SendMessage(ctx, msg, opts...)
}

func (t *NetworkImpl) SendMessageWithResponse(ctx nctx.OperateCtx, msg *pb.XuperMessage, opts ...p2p.OptionFunc) ([]*pb.XuperMessage, error) {
	if !t.isInit() || ctx == nil || msg == nil {
		return nil, fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.SendMessageWithResponse(ctx, msg, opts...)
}

func (t *NetworkImpl) Register(sub p2p.Subscriber) error {
	if !t.isInit() || sub == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.Register(sub)
}

func (t *NetworkImpl) UnRegister(sub p2p.Subscriber) error {
	if !t.isInit() || sub == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.UnRegister(sub)
}

func (t *NetworkImpl) P2PState(nctx.OperateCtx) p2p.State {
	return p2p.State{}
}

func (t *NetworkImpl) isInit() bool {
	if t.ctx == nil || t.p2pServ == nil {
		return false
	}

	return true
}
