package client

import (
	"context"
	"fmt"
	"math/big"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"
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

func (t *XchainClient) SubmitTx(tx *xldgpb.Transaction) (*xchainpb.BaseResp, error) {
	ctx := context.TODO()
	req := &xchainpb.SubmitTxReq{
		Header: t.genReqHeader(),
		Bcname: global.GFlagBCName,
		Txid:   tx.Txid,
		Tx:     tx,
	}
	resp, err := t.xclient.SubmitTx(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetHeader().GetErrCode() != 0 {
		return nil, fmt.Errorf("ErrCode:%d ErrMsg:%s LogId:%s TraceId:%s", resp.GetHeader().GetErrCode(),
			resp.GetHeader().GetErrMsg(), resp.GetHeader().GetLogId(), resp.GetHeader().GetTraceId())
	}

	return resp, nil
}

func (t *XchainClient) PreExec() (*xchainpb.PreExecResp, error) {
	return nil, fmt.Errorf("not impl")
}

func (t *XchainClient) SelectUtxo(need *big.Int) (*xchainpb.SelectUtxoResp, error) {
	addr, err := global.LoadAccount(global.GFlagCrypto, global.GFlagKeys)
	if err != nil {
		return nil, fmt.Errorf("load account info failed.KeyPath:%s Err:%v", global.GFlagKeys, err)
	}

	req := &xchainpb.SelectUtxoReq{
		Header:    t.genReqHeader(),
		Bcname:    global.GFlagBCName,
		Address:   addr.Address,
		TotalNeed: need.String(),
		NeedLock:  true,
	}

	ctx := context.TODO()
	resp, err := t.xclient.SelectUtxo(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetHeader().GetErrCode() != 0 {
		return nil, fmt.Errorf("ErrCode:%d ErrMsg:%s LogId:%s TraceId:%s", resp.GetHeader().GetErrCode(),
			resp.GetHeader().GetErrMsg(), resp.GetHeader().GetLogId(), resp.GetHeader().GetTraceId())
	}

	return resp, nil
}

func (t *XchainClient) QueryBlock(blockId string) (*xchainpb.QueryBlockResp, error) {
	req := &xchainpb.QueryBlockReq{
		Header:      t.genReqHeader(),
		Bcname:      global.GFlagBCName,
		BlockId:     utils.DecodeId(blockId),
		NeedContent: true,
	}

	ctx := context.TODO()
	resp, err := t.xclient.QueryBlock(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetHeader().GetErrCode() != 0 {
		return nil, fmt.Errorf("ErrCode:%d ErrMsg:%s LogId:%s TraceId:%s", resp.GetHeader().GetErrCode(),
			resp.GetHeader().GetErrMsg(), resp.GetHeader().GetLogId(), resp.GetHeader().GetTraceId())
	}

	return resp, nil
}

func (t *XchainClient) QueryChainStatus() (*xchainpb.QueryChainStatusResp, error) {
	req := &xchainpb.QueryChainStatusReq{
		Header:          t.genReqHeader(),
		Bcname:          global.GFlagBCName,
		NeedBranchBlock: true,
	}

	ctx := context.TODO()
	resp, err := t.xclient.QueryChainStatus(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetHeader().GetErrCode() != 0 {
		return nil, fmt.Errorf("ErrCode:%d ErrMsg:%s LogId:%s TraceId:%s", resp.GetHeader().GetErrCode(),
			resp.GetHeader().GetErrMsg(), resp.GetHeader().GetLogId(), resp.GetHeader().GetTraceId())
	}

	return resp, nil
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
		SelfName: xdef.CmdLineName,
	}
}
