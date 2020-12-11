package xuperos

import (
    "context"
    "github.com/xuperchain/xuperchain/core/pb"
    "github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
    "github.com/xuperchain/xupercore/kernel/network/p2p"
    netPB "github.com/xuperchain/xupercore/kernel/network/pb"
)

func (t *NetEvent) GetBlock(ctx context.Context, in *pb.BlockID, opts ...p2p.OptionFunc) (*pb.Block, error) {
    if in == nil || in.GetBcname() == "" || len(in.GetBlockid()) <= 0 {
        return nil, def.ErrMessageParam
    }

    msgOpts := []p2p.MessageOption {
        p2p.WithBCName(in.GetBcname()),
        p2p.WithLogId(in.GetHeader().GetLogid()),
    }
    msg := p2p.NewMessage(netPB.XuperMessage_GET_BLOCK, in, msgOpts...)

    engCtx := t.engine.Context()
    responses, err := engCtx.Net.SendMessageWithResponse(ctx, msg, opts...)
    if err != nil {
        t.log.Warn("GetBlock error", "error", err)
        return nil, err
    }

    for _, response := range responses {
        if response.GetHeader().GetErrorType() != netPB.XuperMessage_SUCCESS {
            continue
        }

        var block pb.Block
        err := p2p.Unmarshal(response, &block)
        if err != nil {
            t.log.Warn("GetBlock unmarshal error", "error", err)
            continue
        }

        return &block, nil
    }

    return nil, def.ErrNoResponse
}
