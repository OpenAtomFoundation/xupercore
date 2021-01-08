package rpc

import (
	"context"

	pb "github.com/xuperchain/xupercore/example/xchain/common/xchainpb"
)

// 示例接口
func (t *RpcServ) CheckAlive(gctx context.Context, req *pb.BaseReq) (*pb.BaseResp, error) {
	// 默认响应
	resp := &pb.BaseResp{Header: t.defRespHeader(req.GetHeader())}

	// 获取请求上下文，对内传递rctx
	rctx := gctx.ValueReqCtx(gctx)

	// 处理请求
	rctx.GetLog().Debug("check alive succ")

	// 设置成功响应
	resp.GetHeader().Error = pb.XChainErrorEnum_SUCCESS
	return resp, nil
}

// 提交交易
func (t *RpcServ) SubmitTx(gctx context.Context, req *pb.SubmitTxReq) (*pb.BaseResp, error) {
	// 默认响应
	resp := &pb.BaseResp{Header: t.defRespHeader(req.GetHeader())}
	return resp, nil
}

// 合约预执行
func (t *RpcServ) PreExec(gctx context.Context, req *pb.PreExecReq) (*pb.PreExecResp, error) {
	// 默认响应
	resp := &pb.PreExecResp{Header: t.defRespHeader(req.GetHeader())}
	return resp, nil
}

// 选择utxo
func (t *RpcServ) SelectUtxo(gctx context.Context, req *pb.SelectUtxoReq) (*pb.SelectUtxoResp, error) {
	// 默认响应
	resp := &pb.SelectUtxoResp{Header: t.defRespHeader(req.GetHeader())}
	return resp, nil
}

// 查询交易信息
func (t *RpcServ) QueryTx(gctx context.Context, req *pb.QueryTxReq) (*pb.QueryTxResp, error) {
	// 默认响应
	resp := &pb.QueryTxResp{Header: t.defRespHeader(req.GetHeader())}
	return resp, nil
}

// 查询区块信息
func (t *RpcServ) QueryBlock(gctx context.Context, req *pb.QueryBlockReq) (*pb.QueryBlockResp, error) {
	// 默认响应
	resp := &pb.QueryBlockResp{Header: t.defRespHeader(req.GetHeader())}
	return resp, nil
}

// 查询区块链状态
func (t *RpcServ) QueryChainStatus(gctx context.Context,
	req *pb.QueryChainStatusReq) (*pb.QueryChainStatusResp, error) {
	// 默认响应
	resp := &pb.QueryChainStatusResp{Header: t.defRespHeader(req.GetHeader())}
	return resp, nil
}
