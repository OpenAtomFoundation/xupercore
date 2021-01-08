package rpc

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"

	sctx "github.com/xuperchain/xupercore/example/xchain/common/context"
	pb "github.com/xuperchain/xupercore/example/xchain/common/xchainpb"
	ecom "github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type RpcServ struct {
	engine ecom.Engine
	log    logs.Logger
}

func NewRpcServ(engine ecom.Engine, log logs.Logger) *RpcServ {
	return &RpcServ{
		engine: engine,
		log:    log,
	}
}

// UnaryInterceptor provides a hook to intercept the execution of a unary RPC on the server.
func (t *RpcServ) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {

		// panic recover
		defer func() {
			if e := recover(); e != nil {
				t.log.Error("Rpc server happen panic.", "error", e, "rpc_method", info.FullMethod)
			}
		}()

		// set request header
		type HeaderInterface interface {
			GetHeader() *pb.ReqHeader
		}
		if req.(HeaderInterface).GetHeader() == nil {
			header := reflect.ValueOf(req).Elem().FieldByName("Header")
			if header.IsValid() && header.IsNil() && header.CanSet() {
				header.Set(reflect.ValueOf(t.defReqHeader()))
			}
		}
		if req.(HeaderInterface).GetHeader().GetLogId() == "" {
			req.(HeaderInterface).GetHeader().LogId = utils.GenLogId()
		}
		reqHeader := req.(HeaderInterface).GetHeader()

		// set request context
		reqCtx, _ := t.createReqCtx(ctx, reqHeader)
		ctx = sctx.WithReqCtx(ctx, reqCtx)

		// output access log
		logFields := make([]interface{}, 0)
		logFields = append(logFields, "from", reqHeader.GetSelfName(),
			"client_ip", reqCtx.GetClientIp(), "rpc_method", info.FullMethod)
		reqCtx.GetLog().Trace("access request", logFields...)

		// handle request
		// 忽略err，强制响应nil，统一通过resp中的错误码标识错误
		resp, _ := handler(ctx, req)

		// output ending log
		// 可以通过log库提供的SetInfoField方法附加输出到ending log
		logFields = append(logFields, "error", respHeader.GetError(),
			"cost_time", reqCtx.GetTimer().Print())
		rctx.GetLog().Info("request done", logFields...)

		return resp, nil
	}
}

func (t *RpcServ) defReqHeader() *pb.ReqHeader {
	return &pb.ReqHeader{
		LogId:    utils.GenLogId(),
		SelfName: "unknow",
	}
}

func (t *RpcServ) defRespHeader(rHeader *pb.ReqHeader) *pb.RespHeader {
	return &pb.RespHeader{
		LogId:   rHeader.GetLogId(),
		Error:   pb.XChainErrorEnum_UNKNOW_ERROR,
		TraceId: utils.GetHostName(),
	}
}

func (t *RpcServ) createReqCtx(gctx context.Context, reqHeader *pb.ReqHeader) (sctx.ReqCtx, error) {
	// 获取客户端ip
	clientIp, err := t.getClietIP(gctx)
	if err != nil {
		t.log.Error("access proc failed because get client ip failed", "error", err)
		return nil, fmt.Errorf("get client ip failed")
	}

	// 创建请求上下文
	rctx, err := sctx.NewReqCtx(t.engine, reqHeader.GetLogId(), clientIp)
	if err != nil {
		t.log.Error("access proc failed because create request context failed", "error", err)
		return nil, fmt.Errorf("create request context failed")
	}

	return rctx, nil
}

func (t *RpcServ) getClietIP(gctx context.Context) (string, error) {
	pr, ok := peer.FromContext(gctx)
	if !ok {
		return "", fmt.Errorf("create peer form context failed")
	}

	if pr.Addr == nil || pr.Addr == net.Addr(nil) {
		return "", fmt.Errorf("get client_ip failed because peer.Addr is nil")
	}

	addrSlice := strings.Split(pr.Addr.String(), ":")
	return addrSlice[0], nil
}
