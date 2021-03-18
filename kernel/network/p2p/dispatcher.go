package p2p

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/lib/crypto/hash"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/lib/utils"
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
	Dispatch(*pb.XuperMessage, Stream) error
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

func (d *dispatcher) Dispatch(msg *pb.XuperMessage, stream Stream) error {
	if msg == nil || msg.GetHeader() == nil || msg.GetData() == nil {
		return ErrMessageEmpty
	}

	xlog, _ := logs.NewLogger(msg.Header.Logid, "p2p")
	ctx := &xctx.BaseCtx{XLog: xlog, Timer: timer.NewXTimer()}
	defer func() {
		ctx.GetLog().Info("Dispatch", "bc", msg.GetHeader().GetBcname(),
			"type", msg.GetHeader().GetType(), "from", msg.GetHeader().GetFrom(),
			"checksum", msg.GetHeader().GetDataCheckSum(), "timer", ctx.GetTimer().Print())
	}()

	if d.IsHandled(msg) {
		ctx.GetLog().SetInfoField("handled", true)
		// return ErrMessageHandled
		return nil
	}

	if stream == nil {
		return ErrStreamNil
	}

	if _, ok := d.mc[msg.GetHeader().GetType()]; !ok {
		return ErrNotRegister
	}

	d.mu.RLock()
	ctx.GetTimer().Mark("lock")
	if _, ok := d.mc[msg.GetHeader().GetType()]; !ok {
		d.mu.RUnlock()
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

			sub.HandleMessage(ctx, msg, stream)
			<-d.parallel
		}(sub)
	}
	d.mu.RUnlock()
	ctx.GetTimer().Mark("unlock")
	wg.Wait()

	ctx.GetTimer().Mark("dispatch")
	d.MaskHandled(msg)
	return nil
}

func MessageKey(msg *pb.XuperMessage) string {
	if msg == nil || msg.GetHeader() == nil {
		return ""
	}

	header := msg.GetHeader()
	buf := new(bytes.Buffer)
	buf.WriteString(header.GetType().String())
	buf.WriteString(header.GetBcname())
	buf.WriteString(header.GetFrom())
	buf.WriteString(header.GetLogid())
	buf.WriteString(fmt.Sprintf("%d", header.GetDataCheckSum()))
	return utils.F(hash.DoubleSha256(buf.Bytes()))
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
