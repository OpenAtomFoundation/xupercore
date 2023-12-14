package bls

import (
	"fmt"
	"sync"
	"time"

	xctx "github.com/OpenAtomFoundation/xupercore/global/kernel/common/xcontext"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/net"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/network"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/network/p2p"
	"github.com/OpenAtomFoundation/xupercore/global/lib/logs"
	"github.com/OpenAtomFoundation/xupercore/global/protos"
	"github.com/OpenAtomFoundation/xupercore/global/service/pb"
)

// Event is struct to handle BLS related p2p event
type Event struct {
	log      logs.Logger
	engine   *common.EngineCtx
	exitChan chan bool
}

var eventOnce sync.Once

func RegisterEvent(engine *common.EngineCtx) error {
	if engine == nil {
		return fmt.Errorf("new bls event failed because param error")
	}

	event := &Event{
		log:      engine.XLog,
		engine:   engine,
		exitChan: make(chan bool, 1),
	}

	// 订阅监听事件
	var err error
	eventOnce.Do(func() {
		err = event.subscriber()
		if err != nil {
			err = fmt.Errorf("new bls event failed because register subscriber error.err:%v", err)
		}
	})
	return err
}

func (e *Event) subscriber() error {
	var SyncMsgHandle = map[protos.XuperMessage_MessageType]p2p.HandleFunc{
		protos.XuperMessage_BLS_GET_PUBLIC_KEY:        e.handleGetPublicKey,
		protos.XuperMessage_BLS_GET_MEMBER_SIGN_PART:  e.handleGetMemberSignPart,
		protos.XuperMessage_BLS_GET_MESSAGE_SIGN_PART: e.handleGetMessageSignPart,
	}

	// subscribe events
	net := e.net()
	for msgType, handle := range SyncMsgHandle {
		if err := net.Register(p2p.NewSubscriber(net.Context(), msgType, handle)); err != nil {
			e.log.Error("register subscriber error", "type", msgType, "error", err)
			return fmt.Errorf("register subscriber failed")
		}
	}

	e.log.Trace("register bls subscriber succ")
	return nil
}

// net gets Net object in engine context
func (e *Event) net() network.Network {
	return e.engine.Net
}

func (e *Event) handleGetPublicKey(ctx xctx.XContext, message *protos.XuperMessage) (*protos.XuperMessage, error) {
	ctx.GetLog().Debug("handleGetPublicKey invoked",
		"message", message,
		"time", time.Now().Nanosecond())

	bcName := message.Header.Bcname
	var output = new(pb.BroadcastPublicKeyResponse)
	response := func(err error) (*protos.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(net.ErrorType(err)),
			p2p.WithLogId(message.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(p2p.GetRespMessageType(message.GetHeader().GetType()), output, opts...)

		ctx.GetLog().Debug("handleGetPublicKey response invoked",
			"output", output,
			"error", err,
			"response", resp)
		return resp, nil
	}

	var req pb.BroadcastPublicKeyRequest
	err := p2p.Unmarshal(message, &req)
	if err != nil {
		ctx.GetLog().Error("unmarshal error", "bcName", bcName, "error", err)
		return response(common.ErrParameter)
	}

	ctx.GetLog().Debug("updateGroupMember()",
		"peerIndex", req.Index,
		"peerId", message.Header.From)
	output, err = thresholdSign.updateGroupMember(ctx, req, message.Header.From)
	if err != nil {
		ctx.GetLog().Error("updateGroupMember error", "bcName", bcName, "error", err)
		return response(err)
	}
	return response(nil)
}

func (e *Event) handleGetMemberSignPart(ctx xctx.XContext, message *protos.XuperMessage) (*protos.XuperMessage, error) {
	ctx.GetLog().Debug("handleGetMemberSignPart invoked", "message", message)

	bcName := message.Header.Bcname
	var output = new(pb.SendMemberSignResponse)
	response := func(err error) (*protos.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(net.ErrorType(err)),
			p2p.WithLogId(message.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(p2p.GetRespMessageType(message.GetHeader().GetType()), output, opts...)
		return resp, nil
	}

	var req pb.SendMemberSignRequest
	err := p2p.Unmarshal(message, &req)
	if err != nil {
		ctx.GetLog().Error("unmarshal error", "bcName", bcName, "error", err)
		return response(common.ErrParameter)
	}

	output, err = thresholdSign.updateMemberSignPart(req)
	if err != nil {
		ctx.GetLog().Error("updateMemberSignPart error", "bcName", bcName, "error", err)
		return response(err)
	}
	return response(nil)
}

func (e *Event) handleGetMessageSignPart(ctx xctx.XContext, message *protos.XuperMessage) (*protos.XuperMessage, error) {
	ctx.GetLog().Debug("handleGetMessageSignPart invoked", "message", message)

	bcName := message.Header.Bcname
	var output = new(pb.BroadcastMessageSignResponse)
	response := func(err error) (*protos.XuperMessage, error) {
		opts := []p2p.MessageOption{
			p2p.WithBCName(bcName),
			p2p.WithErrorType(net.ErrorType(err)),
			p2p.WithLogId(message.GetHeader().GetLogid()),
		}
		resp := p2p.NewMessage(p2p.GetRespMessageType(message.GetHeader().GetType()), output, opts...)
		return resp, nil
	}

	var req pb.BroadcastMessageSignRequest
	err := p2p.Unmarshal(message, &req)
	if err != nil {
		_, decompressErr := p2p.Decompress(message)
		ctx.GetLog().Error("unmarshal error",
			"bcName", bcName,
			"error", err,
			"decompress", decompressErr)
		return response(common.ErrParameter)
	}

	output, err = thresholdSign.updateMessageSignPart(ctx, req)
	if err != nil {
		ctx.GetLog().Error("updateMessageSignPart error", "bcName", bcName, "error", err)
		return response(err)
	}
	return response(nil)
}
