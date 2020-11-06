package p2p

import (
    nctx "github.com/xuperchain/xupercore/kernel/network/context"
    pb "github.com/xuperchain/xupercore/kernel/network/pb"
    "testing"
)

type dispatcherCase struct {
    sub Subscriber
    msg *pb.XuperMessage
    stream Stream
    regErr error
    handleErr error
}

func TestDispatcher(t *testing.T) {
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
            sub: NewSubscriber(nctx.MockDomainCtx(), pb.XuperMessage_GET_BLOCK, nil),
            msg: msg,
            stream: stream,
            regErr: ErrSubscriber,
            handleErr: nil,
        },
        {
            sub: NewSubscriber(nctx.MockDomainCtx(), pb.XuperMessage_GET_BLOCK, ch),
            msg: nil,
            stream: stream,
            regErr: nil,
            handleErr: ErrMessageEmpty,
        },
        {
            sub: NewSubscriber(nctx.MockDomainCtx(), pb.XuperMessage_GET_BLOCK, ch),
            msg: msg,
            stream: nil,
            regErr: nil,
            handleErr: ErrStreamNil,
        },
        {
            sub: NewSubscriber(nctx.MockDomainCtx(), pb.XuperMessage_GET_BLOCK, ch),
            msg: msgPostTx,
            stream: stream,
            regErr: nil,
            handleErr: ErrNotRegister,
        },
        {
            sub: NewSubscriber(nctx.MockDomainCtx(), pb.XuperMessage_GET_BLOCK, ch),
            msg: msg,
            stream: stream,
            regErr: nil,
            handleErr: nil,
        },
        {
            sub: NewSubscriber(nctx.MockDomainCtx(), pb.XuperMessage_GET_BLOCK, ch),
            msg: msg,
            stream: stream,
            regErr: nil,
            handleErr: ErrMessageHandled,
        },
    }

    dispatcher := NewDispatcher()
    for i, c := range cases {
        err := dispatcher.Register(c.sub)
        if c.regErr != nil {
            if c.regErr != err{
                t.Errorf("case[%d]: register error: %v", i, err)
            }
            continue
        }

        err = dispatcher.Register(c.sub)
        if err != ErrRegistered {
            t.Errorf("case[%d]: register error: %v", i, err)
            continue
        }

        err = dispatcher.Dispatch(nctx.MockOperateCtx(), c.msg, c.stream)
        if c.handleErr != err {
            t.Errorf("case[%d]: dispatch error: %v", i, err)
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
