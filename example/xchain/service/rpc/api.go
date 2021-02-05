package rpc

import (
	"context"
	"math/big"

	sctx "github.com/xuperchain/xupercore/example/xchain/common/context"
	pb "github.com/xuperchain/xupercore/example/xchain/common/xchainpb"
	"github.com/xuperchain/xupercore/example/xchain/models"
	ecom "github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/utils"
)

// 注意：
// 1.rpc接口响应resp不能为nil，必须实例化
// 2.rpc接口响应err必须为ecom.Error类型的标准错误，没有错误响应err=nil
// 3.rpc接口不需要关注resp.Header，由拦截器根据err统一设置
// 4.rpc接口可以调用log库提供的SetInfoField方法附加输出到ending log

// 示例接口
func (t *RpcServ) CheckAlive(gctx context.Context, req *pb.BaseReq) (*pb.BaseResp, error) {
	// 默认响应
	resp := &pb.BaseResp{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	rctx.GetLog().Debug("check alive succ")
	return resp, nil
}

// 提交交易
func (t *RpcServ) SubmitTx(gctx context.Context, req *pb.SubmitTxReq) (*pb.BaseResp, error) {
	// 默认响应
	resp := &pb.BaseResp{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	// 校验参数
	if req == nil || req.GetTx() == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, ecom.ErrParameter
	}

	// 提交交易
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	err = handle.SubmitTx(req.GetTx())
	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("txid", utils.F(req.GetTxid()))
	return resp, err
}

// 合约预执行
func (t *RpcServ) PreExec(gctx context.Context, req *pb.PreExecReq) (*pb.PreExecResp, error) {
	// 默认响应
	resp := &pb.PreExecResp{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	// 校验参数
	if req == nil || req.GetBcname() == "" || len(req.GetRequests()) < 1 {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, ecom.ErrParameter
	}

	// 预执行
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	res, err := handle.PreExec(req.GetRequests(), req.GetInitiator(), req.GetAuthRequire())
	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("initiator", req.GetInitiator())
	// 设置响应
	if err == nil {
		resp.Bcname = req.GetBcname()
		resp.Response = res
	}

	return resp, err
}

// 选择utxo
func (t *RpcServ) SelectUtxo(gctx context.Context, req *pb.SelectUtxoReq) (*pb.SelectUtxoResp, error) {
	// 默认响应
	resp := &pb.SelectUtxoResp{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	// 校验参数
	if req == nil || req.GetBcname() == "" || req.GetAddress() == "" || req.GetTotalNeed() == "" {
		return resp, ecom.ErrParameter
	}
	totalNeed, ok := new(big.Int).SetString(req.GetTotalNeed(), 10)
	if !ok {
		return resp, ecom.ErrParameter
	}

	// 选择utxo
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	res, err := handle.SelectUtxo(req.GetAddress(), totalNeed, req.GetNeedLock(), false)
	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("address", req.GetAddress())
	rctx.GetLog().SetInfoField("need_amount", req.GetTotalNeed())
	// 设置响应
	if err == nil {
		resp.UtxoList = res.UtxoList
		resp.TotalAmount = res.TotalSelected
	}

	return resp, err
}

// 查询交易信息
func (t *RpcServ) QueryTx(gctx context.Context, req *pb.QueryTxReq) (*pb.QueryTxResp, error) {
	// 默认响应
	resp := &pb.QueryTxResp{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	// 校验参数
	if req == nil || req.GetBcname() == "" || len(req.GetTxid()) < 1 {
		return resp, ecom.ErrParameter
	}

	// 查询交易
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	res, err := handle.QueryTx(req.GetTxid())
	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("txid", utils.F(req.GetTxid()))
	// 设置响应
	if err == nil {
		resp.Status = res.GetStatus()
		resp.Distance = res.GetDistance()
		resp.Tx = res.GetTx()
	}

	return resp, err
}

// 查询区块信息
func (t *RpcServ) QueryBlock(gctx context.Context, req *pb.QueryBlockReq) (*pb.QueryBlockResp, error) {
	// 默认响应
	resp := &pb.QueryBlockResp{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	// 校验参数
	if req == nil || req.GetBcname() == "" || len(req.GetBlockId()) < 1 {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, ecom.ErrParameter
	}

	// 查询区块
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	res, err := handle.QueryBlock(req.GetBlockId(), req.GetNeedContent())
	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("block_id", utils.F(req.GetBlockId()))
	// 设置响应
	if err == nil {
		resp.Status = res.GetStatus()
		resp.Block = res.GetBlock()
	}

	return resp, err
}

// 查询区块链状态
func (t *RpcServ) QueryChainStatus(gctx context.Context,
	req *pb.QueryChainStatusReq) (*pb.QueryChainStatusResp, error) {
	// 默认响应
	resp := &pb.QueryChainStatusResp{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	// 校验参数
	if req == nil || req.GetBcname() == "" {
		return resp, ecom.ErrParameter
	}

	// 预执行
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	res, err := handle.QueryChainStatus(req.GetNeedBranchBlock())
	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	// 设置响应
	if err == nil {
		resp.Bcname = req.GetBcname()
		resp.LedgerMeta = res.LedgerMeta
		resp.UtxoMeta = res.UtxoMeta
		resp.BranchBlockId = res.BranchIds
	}

	return resp, err
}
