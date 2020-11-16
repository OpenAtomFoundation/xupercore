package p2p

import (
	"errors"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	pb "github.com/xuperchain/xupercore/kernel/network/pb"
	"testing"
)

type mockStream struct{}

func (s *mockStream) Send(msg *pb.XuperMessage) error { return nil }

type mockStreamError struct{}

func (s *mockStreamError) Send(msg *pb.XuperMessage) error { return errors.New("mock stream error") }

type mockHandler struct{}

func (h *mockHandler) Handler(ctx nctx.OperateCtx, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	msg.Header.Type = GetRespMessageType(msg.Header.Type)
	return msg, nil
}

type mockHandlerError struct{}

func (h *mockHandlerError) Handler(ctx nctx.OperateCtx, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	return nil, errors.New("mock handler error")
}

type mockHandlerNil struct{}

func (h *mockHandlerNil) Handler(ctx nctx.OperateCtx, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	return nil, nil
}

type subscriberCase struct {
	v      interface{}
	msg    *pb.XuperMessage
	stream Stream
	err    error
}

func TestSubscriber(t *testing.T) {
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

	for i, c := range cases {
		sub := NewSubscriber(nctx.MockDomainCtx(), pb.XuperMessage_GET_BLOCK, c.v, WithFrom("from"))
		if sub == nil {
			t.Logf("case[%d]: sub is nil", i)
			continue
		}

		if err := sub.HandleMessage(nctx.MockOperateCtx(), c.msg, c.stream); err != c.err {
			t.Error(err)
		}
	}
}
