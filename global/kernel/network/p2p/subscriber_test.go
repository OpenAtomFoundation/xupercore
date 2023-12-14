package p2p

import (
	"context"
	"errors"
	"testing"

	xctx "github.com/OpenAtomFoundation/xupercore/global/kernel/common/xcontext"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/mock"
	nctx "github.com/OpenAtomFoundation/xupercore/global/kernel/network/context"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/network/def"
	"github.com/OpenAtomFoundation/xupercore/global/lib/logs"
	"github.com/OpenAtomFoundation/xupercore/global/lib/timer"
	pb "github.com/OpenAtomFoundation/xupercore/global/protos"
)

type mockStream struct{}

func (s *mockStream) Send(msg *pb.XuperMessage) error { return nil }

type mockStreamError struct{}

func (s *mockStreamError) Send(msg *pb.XuperMessage) error { return errors.New("mock stream error") }

type mockHandler struct{}

func (h *mockHandler) Handler(ctx context.Context, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	msg.Header.Type = GetRespMessageType(msg.Header.Type)
	return msg, nil
}

type mockHandlerError struct{}

func (h *mockHandlerError) Handler(ctx context.Context, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	return nil, errors.New("mock handler error")
}

type mockHandlerNil struct{}

func (h *mockHandlerNil) Handler(ctx context.Context, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	return nil, nil
}

type subscriberCase struct {
	v      interface{}
	msg    *pb.XuperMessage
	stream Stream
	err    error
}

func TestSubscriber(t *testing.T) {
	mock.InitLogForTest()

	msg := NewMessage(pb.XuperMessage_GET_BLOCK, &pb.XuperMessage{},
		WithBCName("xuper"),
		WithLogId("1234567890"),
		WithVersion(MessageVersion3),
	)
	msg.Header.From = "from"

	cases := []subscriberCase{
		{
			v:   nil,
			err: nil,
		},
		{
			msg:    msg,
			v:      make(chan *pb.XuperMessage, 1),
			stream: &mockStream{},
			err:    nil,
		},
		{
			msg:    msg,
			v:      &mockHandler{},
			stream: &mockStreamError{},
			err:    ErrStreamSendError,
		},
		{
			msg:    msg,
			v:      &mockHandlerError{},
			stream: &mockStream{},
			err:    ErrHandlerError,
		},
		{
			msg:    msg,
			v:      &mockHandlerNil{},
			stream: &mockStream{},
			err:    ErrResponseNil,
		},
	}

	ecfg, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := nctx.NewNetCtx(ecfg)

	for i, c := range cases {
		sub := NewSubscriber(ctx, pb.XuperMessage_GET_BLOCK, c.v, WithFilterFrom("from"))
		if sub == nil {
			t.Logf("case[%d]: sub is nil", i)
			continue
		}

		log, _ := logs.NewLogger("", def.SubModName)
		rctx := &xctx.BaseCtx{
			XLog:  log,
			Timer: timer.NewXTimer(),
		}
		if err := sub.HandleMessage(rctx, c.msg, c.stream); err != c.err {
			t.Error(err)
		}
	}
}
