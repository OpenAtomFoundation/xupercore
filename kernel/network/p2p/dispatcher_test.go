package p2p

import (
	"testing"

	"github.com/xuperchain/xupercore/kernel/mock"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	pb "github.com/xuperchain/xupercore/protos"
)

type dispatcherCase struct {
	sub       Subscriber
	msg       *pb.XuperMessage
	stream    Stream
	regErr    error
	handleErr error
}

func TestDispatcher(t *testing.T) {
	mock.InitLogForTest()
	ecfg, _ := mock.NewEnvConfForTest()
	netCtx, _ := nctx.NewNetCtx(ecfg)

	ch := make(chan *pb.XuperMessage, 1)
	stream := &mockStream{}

	msg := NewMessage(pb.XuperMessage_GET_BLOCK, &pb.XuperMessage{},
		WithBCName("xuper"),
		WithLogId("1234567890"),
		WithVersion(MessageVersion3),
	)

	msgPostTx := NewMessage(pb.XuperMessage_POSTTX, &pb.XuperMessage{},
		WithBCName("xuper"),
		WithLogId("1234567890"),
		WithVersion(MessageVersion3),
	)

	cases := []dispatcherCase{
		{
			sub:       NewSubscriber(netCtx, pb.XuperMessage_GET_BLOCK, nil),
			msg:       msg,
			stream:    stream,
			regErr:    ErrSubscriber,
			handleErr: nil,
		},
		{
			sub:       NewSubscriber(netCtx, pb.XuperMessage_GET_BLOCK, ch),
			msg:       nil,
			stream:    stream,
			regErr:    nil,
			handleErr: ErrMessageEmpty,
		},
		{
			sub:       NewSubscriber(netCtx, pb.XuperMessage_GET_BLOCK, ch),
			msg:       msg,
			stream:    nil,
			regErr:    nil,
			handleErr: ErrStreamNil,
		},
		{
			sub:       NewSubscriber(netCtx, pb.XuperMessage_GET_BLOCK, ch),
			msg:       msgPostTx,
			stream:    stream,
			regErr:    nil,
			handleErr: ErrNotRegister,
		},
		{
			sub:       NewSubscriber(netCtx, pb.XuperMessage_GET_BLOCK, ch),
			msg:       msg,
			stream:    stream,
			regErr:    nil,
			handleErr: nil,
		},
		{
			sub:       NewSubscriber(netCtx, pb.XuperMessage_GET_BLOCK, ch),
			msg:       msg,
			stream:    stream,
			regErr:    nil,
			handleErr: ErrMessageHandled,
		},
	}

	ecfg, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := nctx.NewNetCtx(ecfg)
	dispatcher := NewDispatcher(ctx)
	for i, c := range cases {
		err := dispatcher.Register(c.sub)
		if c.regErr != nil {
			if c.regErr != err {
				t.Errorf("case[%d]: register error: %v", i, err)
			}
			continue
		}

		err = dispatcher.Register(c.sub)
		if err != ErrRegistered {
			t.Errorf("case[%d]: register error: %v", i, err)
			continue
		}

		err = dispatcher.Dispatch(c.msg, c.stream)
		if c.handleErr != err {
			//t.Errorf("case[%d]: dispatch error: %v", i, err)
			continue
		}

		err = dispatcher.UnRegister(c.sub)
		if err != nil {
			t.Errorf("case[%d]: unregister error: %v", i, err)
			continue
		}

		err = dispatcher.UnRegister(c.sub)
		if err != ErrNotRegister {
			t.Errorf("case[%d]: unregister error: %v", i, err)
			continue
		}
	}
}
