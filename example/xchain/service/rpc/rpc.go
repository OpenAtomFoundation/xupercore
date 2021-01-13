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
		handler grpc.UnaryHandler) (interface{}, error) {

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
		// 根据err自动设置响应错误码，err需要是定义的标准err，否则会响应为未知错误
		stdErr := ecom.ErrSuccess
		respRes, err := handler(ctx, req)
		if err != nil {
			stdErr = ecom.CastError(err)
		}
		// 根据错误统一设置header，对外统一响应err=nil，通过Header.ErrCode判断
		respHeader := &pb.RespHeader{
			LogId:   reqHeader.GetLogId(),
			ErrCode: int64(stdErr.Code),
			ErrMsg:  stdErr.Msg,
			TraceId: t.genTraceId(),
		}
		// 通过反射设置header到response
		header := reflect.ValueOf(respRes).Elem().FieldByName("Header")
		if header.IsValid() && header.IsNil() && header.CanSet() {
			header.Set(reflect.ValueOf(respHeader))
		}

		// output ending log
		// 可以通过log库提供的SetInfoField方法附加输出到ending log
		logFields = append(logFields, "status", stdErr.Status, "err_code", stdErr.Code,
			"err_msg", stdErr.Msg, "cost_time", reqCtx.GetTimer().Print())
		reqCtx.GetLog().Info("request done", logFields...)

		return respRes, nil
	}
}

func (t *RpcServ) defReqHeader() *pb.ReqHeader {
	return &pb.ReqHeader{
		LogId:    utils.GenLogId(),
		SelfName: "unknow",
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

// 生成包含机器host和请求时间的AES加密字符串，方便问题定位
func (t *RpcServ) genTraceId() string {
	return "127.0.0.1"
}
