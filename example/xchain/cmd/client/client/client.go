package client

import (
	"context"
	"fmt"

	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	"github.com/xuperchain/xupercore/example/xchain/common/xchainpb"
	"github.com/xuperchain/xupercore/lib/utils"

	"google.golang.org/grpc"
)

type XchainClient struct {
	xclient xchainpb.XchainClient
}

func NewXchainClient() (*XchainClient, error) {
	conn, err := grpc.Dial(global.GFlagHost, grpc.WithInsecure(), grpc.WithMaxMsgSize(64<<20-1))
	if err != nil {
		return nil, err
	}

	client := &XchainClient{
		xclient: xchainpb.NewXchainClient(conn),
	}

	return client, nil
}

func (t *XchainClient) QueryTx(txId string) (*xchainpb.QueryTxResp, error) {
	req := &xchainpb.QueryTxReq{
		Header: t.genReqHeader(),
		Bcname: global.GFlagBCName,
		Txid:   utils.DecodeId(txId),
	}

	ctx := context.TODO()
	resp, err := t.xclient.QueryTx(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetHeader().GetErrCode() != 0 {
		return nil, fmt.Errorf("ErrCode:%d ErrMsg:%s LogId:%s TraceId:%s", resp.GetHeader().GetErrCode(),
			resp.GetHeader().GetErrMsg(), resp.GetHeader().GetLogId(), resp.GetHeader().GetTraceId())
	}

	return resp, nil
}

func (t *XchainClient) genReqHeader() *xchainpb.ReqHeader {
	return &xchainpb.ReqHeader{
		LogId:    utils.GenLogId(),
		SelfName: "xchain-cli",
	}
}
