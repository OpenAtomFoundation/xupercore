package rpc

import (
	"context"

	"github.com/xuperchain/xupercore/server/pb"
)

// 示例接口
func (t *RpcServ) CheckAlive(gctx context.Context, req *pb.BaseReq) (*pb.BaseResp, error) {
	// 默认响应
	resp := &pb.BaseResp{Header: t.defRespHeader(req.GetHeader())}

	// 初始化请求上下文以及记录请求日志
	rctx, err := t.access(gctx, req.GetHeader(), "from", req.GetHeader().GetSelfName())
	if err != nil {
		resp.GetHeader().Error = pb.XChainErrorEnum_UNKNOW_ERROR
		t.log.Error("request access porc failed", "log_id", req.GetHeader().GetLogId(), "err", err)
		return resp, nil
	}
	defer t.ending(rctx, resp.GetHeader(), "status", "running")

	// 处理请求
	rctx.GetLog().Debug("check alive succ")

	// 设置成功响应
	resp.GetHeader().Error = pb.XChainErrorEnum_SUCCESS
	return resp, nil
}
