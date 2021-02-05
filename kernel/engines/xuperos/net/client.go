package xuperos

import (
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/protos"
)

func (t *NetEvent) GetBlock(ctx xctx.XContext, request *protos.XuperMessage) (*lpb.InternalBlock, error) {
	var block lpb.InternalBlock
	if err := p2p.Unmarshal(request, &block); err != nil {
		ctx.GetLog().Warn("handleNewBlockID Unmarshal request error", "error", err)
		return nil, common.ErrParameter
	}

	msgOpts := []p2p.MessageOption{
		p2p.WithBCName(request.Header.Bcname),
		p2p.WithLogId(request.Header.Bcname),
	}
	msg := p2p.NewMessage(protos.XuperMessage_GET_BLOCK, &block, msgOpts...)
	responses, err := t.engine.Context().Net.SendMessageWithResponse(ctx, msg, p2p.WithPeerIDs([]string{request.GetHeader().GetFrom()}))
	if err != nil {
		return nil, common.ErrSendMessageFailed
	}

	for _, response := range responses {
		if response.GetHeader().GetErrorType() != protos.XuperMessage_SUCCESS {
			ctx.GetLog().Warn("GetBlock response error", "errorType", response.GetHeader().GetErrorType(), "from", response.GetHeader().GetFrom())
			continue
		}

		var block lpb.InternalBlock
		err := p2p.Unmarshal(response, &block)
		if err != nil {
			ctx.GetLog().Warn("GetBlock unmarshal error", "error", err, "from", response.GetHeader().GetFrom())
			continue
		}

		return &block, nil
	}

	return nil, common.ErrNetworkNoResponse
}
