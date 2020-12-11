package p2p

import (
	"errors"
	"fmt"
	"sync"
	"time"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/lib/logs"
	pb "github.com/xuperchain/xupercore/protos"

	"github.com/patrickmn/go-cache"
)

var (
	ErrSubscriber     = errors.New("subscribe error")
	ErrRegistered     = errors.New("subscriber already registered")
	ErrMessageEmpty   = errors.New("message empty")
	ErrMessageHandled = errors.New("message handled")
	ErrStreamNil      = errors.New("stream is nil")
	ErrNotRegister    = errors.New("message not register")
)

// Dispatcher
type Dispatcher interface {
	Register(sub Subscriber) error
	UnRegister(sub Subscriber) error

	// Dispatch dispatch message to registered subscriber
	Dispatch(xctx.XContext, *pb.XuperMessage, Stream) error
}

// dispatcher implement interface Dispatcher
type dispatcher struct {
	ctx *nctx.NetCtx
	log logs.Logger

	mu      sync.RWMutex
	mc      map[pb.XuperMessage_MessageType]map[Subscriber]struct{}
	handled *cache.Cache

	// control goroutinue number
	parallel chan struct{}
}

var _ Dispatcher = &dispatcher{}

func NewDispatcher(ctx *nctx.NetCtx) Dispatcher {
	d := &dispatcher{
		ctx:     ctx,
		log:     ctx.XLog,
		mc:      make(map[pb.XuperMessage_MessageType]map[Subscriber]struct{}),
		handled: cache.New(time.Duration(3)*time.Second, 1*time.Second),
		// TODO: 根据压测数据调整并发度，修改为配置
		parallel: make(chan struct{}, 1024),
	}

	return d
}

func (d *dispatcher) Register(sub Subscriber) error {
	if sub == nil || sub.GetMessageType() == pb.XuperMessage_MSG_TYPE_NONE {
		return ErrSubscriber
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.mc[sub.GetMessageType()]; !ok {
		d.mc[sub.GetMessageType()] = make(map[Subscriber]struct{}, 1)
	}

	if _, ok := d.mc[sub.GetMessageType()][sub]; ok {
		return ErrRegistered
	}

	d.mc[sub.GetMessageType()][sub] = struct{}{}
	return nil
}

func (d *dispatcher) UnRegister(sub Subscriber) error {
	if sub == nil || sub.GetMessageType() == pb.XuperMessage_MSG_TYPE_NONE {
		return ErrSubscriber
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.mc[sub.GetMessageType()]; !ok {
		return ErrNotRegister
	}

	if _, ok := d.mc[sub.GetMessageType()][sub]; !ok {
		return ErrNotRegister
	}

	delete(d.mc[sub.GetMessageType()], sub)
	return nil
}

func (d *dispatcher) Dispatch(ctx xctx.XContext, msg *pb.XuperMessage, stream Stream) error {
	if msg == nil || msg.GetHeader() == nil || msg.GetData() == nil {
		return ErrMessageEmpty
	}

	if d.IsHandled(msg) {
		return ErrMessageHandled
	}

	if stream == nil {
		return ErrStreamNil
	}

	if _, ok := d.mc[msg.GetHeader().GetType()]; !ok {
		return ErrNotRegister
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	if _, ok := d.mc[msg.GetHeader().GetType()]; !ok {
		return ErrNotRegister
	}

	var wg sync.WaitGroup
	for sub, _ := range d.mc[msg.GetHeader().GetType()] {
		if !sub.Match(msg) {
			continue
		}

		d.parallel <- struct{}{}
		wg.Add(1)
		go func(sub Subscriber) {
			defer wg.Done()

			err := sub.HandleMessage(ctx, msg, stream)
			if err != nil {
				d.log.Trace("dispatch handle message error",
					"log_id", msg.GetHeader().GetLogid(), "type", msg.GetHeader().GetType(),
					"resp.from", msg.GetHeader().GetFrom(), "error", err)
			}

			<-d.parallel
		}(sub)
	}
	wg.Wait()

	d.MaskHandled(msg)
	return nil
}

func MessageKey(msg *pb.XuperMessage) string {
	if msg == nil || msg.GetHeader() == nil {
		return ""
	}

	return fmt.Sprintf("%s_%d", msg.GetHeader().GetLogid(), msg.GetHeader().GetDataCheckSum())
}

// filter handled message
func (d *dispatcher) MaskHandled(msg *pb.XuperMessage) {
	key := MessageKey(msg)
	d.handled.Set(key, true, time.Duration(3)*time.Second)
}

func (d *dispatcher) IsHandled(msg *pb.XuperMessage) bool {
	key := MessageKey(msg)
	_, ok := d.handled.Get(key)
	return ok
}
