package xuperos

import (
    lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/pb"
    "github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
    "github.com/xuperchain/xupercore/kernel/network/p2p"
    "github.com/xuperchain/xupercore/protos"
)

func GetBlock(ctx *common.EngineCtx, request *protos.XuperMessage, opts ...p2p.OptionFunc) (*lpb.InternalBlock, error) {
    responses, err := ctx.Net.SendMessageWithResponse(ctx, request, opts...)
    if err != nil {
        return nil, err
    }

    for _, response := range responses {
        if response.GetHeader().GetErrorType() != protos.XuperMessage_SUCCESS {
            continue
        }

        var block *lpb.InternalBlock
        err := p2p.Unmarshal(response, block)
        if err != nil {
            ctx.GetLog().Warn("GetBlock unmarshal error", "error", err)
            continue
        }

        return block, nil
    }

    return nil, common.ErrNetworkNoResponse
}
