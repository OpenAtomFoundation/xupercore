package p2p

import (
	"context"
	"errors"
	"reflect"
	"time"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
	pb "github.com/xuperchain/xupercore/protos"

	"github.com/golang/protobuf/proto"
	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	ErrHandlerError    = errors.New("handler error")
	ErrResponseNil     = errors.New("handler response is nil")
	ErrStreamSendError = errors.New("send response error")
	ErrChannelBlock    = errors.New("channel block")
)

// Subscriber is the interface for p2p message subscriber
type Subscriber interface {
	GetMessageType() pb.XuperMessage_MessageType
	Match(*pb.XuperMessage) bool
	HandleMessage(xctx.XContext, *pb.XuperMessage, Stream) error
}

// Stream send p2p response message
type Stream interface {
	Send(*pb.XuperMessage) error
}

type HandleFunc func(xctx.XContext, *pb.XuperMessage) (*pb.XuperMessage, error)

type SubscriberOption func(*subscriber)

func WithFilterFrom(from string) SubscriberOption {
	return func(s *subscriber) {
		s.from = from
	}
}

func WithFilterBCName(bcName string) SubscriberOption {
	return func(s *subscriber) {
		s.bcName = bcName
	}
}

func NewSubscriber(ctx *nctx.NetCtx, typ pb.XuperMessage_MessageType,
	v interface{}, opts ...SubscriberOption) Subscriber {

	s := &subscriber{
		ctx: ctx,
		log: ctx.XLog,
		typ: typ,
	}

	switch obj := v.(type) {
	case HandleFunc:
		s.handler = obj
	case chan *pb.XuperMessage:
		s.channel = obj
	default:
		ctx.GetLog().Error("not handler or channel", "msgType", typ, "obj", reflect.TypeOf(obj))
		return nil
	}

	if s.handler == nil && s.channel == nil {
		ctx.GetLog().Error("need handler or channel", "msgType", typ)
		return nil
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type subscriber struct {
	ctx *nctx.NetCtx
	log logs.Logger

	typ pb.XuperMessage_MessageType // 订阅消息类型

	// filter
	bcName string // 接收指定链的消息
	from   string // 接收指定节点的消息

	channel chan *pb.XuperMessage
	handler HandleFunc
}

var _ Subscriber = &subscriber{}

func (s *subscriber) GetMessageType() pb.XuperMessage_MessageType {
	return s.typ
}

func (s *subscriber) Match(msg *pb.XuperMessage) bool {
	if s.from != "" && s.from != msg.GetHeader().GetFrom() {
		s.log.Debug("subscriber: subscriber from not match", "log_id", msg.GetHeader().GetLogid(),
			"from", s.from, "req.from", msg.GetHeader().GetFrom(), "type", msg.GetHeader().GetType())
		return false
	}

	if s.bcName != "" && s.bcName != msg.GetHeader().GetBcname() {
		s.log.Debug("subscriber: subscriber bcName not match", "log_id", msg.GetHeader().GetLogid(),
			"bc", s.bcName, "req.from", msg.GetHeader().GetBcname(), "type", msg.GetHeader().GetType())
		return false
	}

	return true
}

func (s *subscriber) HandleMessage(ctx xctx.XContext, msg *pb.XuperMessage, stream Stream) error {
	ctx = &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
	defer func() {
		ctx.GetLog().Debug("HandleMessage", "bc", msg.GetHeader().GetBcname(),
			"type", msg.GetHeader().GetType(), "from", msg.GetHeader().GetFrom(), "timer", ctx.GetTimer().Print())
	}()

	if s.handler != nil {
		resp, err := s.handler(ctx, msg)
		ctx.GetTimer().Mark("handle")
		if err != nil {
			ctx.GetLog().Error("subscriber: call user handler error", "err", err)
		}

		if resp == nil || resp.Header == nil {
			ctx.GetLog().Error("subscriber: handler response is nil")

			opts := []MessageOption{
				WithBCName(msg.Header.Bcname),
				WithErrorType(pb.XuperMessage_UNKNOW_ERROR),
				WithLogId(msg.Header.Logid),
			}
			resp = NewMessage(GetRespMessageType(msg.Header.Type), nil, opts...)
		}

		resp.Header.Logid = msg.Header.Logid
		err = stream.Send(resp)
		ctx.GetTimer().Mark("send")
		if err != nil {
			ctx.GetLog().Error("subscriber: send response error", "err", err)
			return ErrStreamSendError
		}

		if s.ctx.EnvCfg.MetricSwitch {
			labels := prom.Labels{
				"bcname": resp.GetHeader().GetBcname(),
				"type":   resp.GetHeader().GetType().String(),
				"method": "HandleMessage",
			}
			Metrics.Packet.With(labels).Add(float64(proto.Size(resp)))
		}
	}

	if s.channel != nil {
		timeout, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		select {
		case <-timeout.Done():
			ctx.GetLog().Error("subscriber: discard message because channel block", "err", timeout.Err())
			return ErrChannelBlock
		case s.channel <- msg:
			ctx.GetTimer().Mark("channel")
		default:
		}
	}

	return nil
}
