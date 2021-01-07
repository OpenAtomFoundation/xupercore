package rpc

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"

	sctx "github.com/xuperchain/xupercore/example/xchain/common/context"
	pb "github.com/xuperchain/xupercore/example/xchain/common/xchainpb"
	edef "github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type RpcServ struct {
	engine edef.Engine
	log    logs.Logger
}

func NewRpcServ(engine edef.Engine, log logs.Logger) *RpcServ {
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

		// handle request
		return handler(ctx, req)
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

// 请求处理前处理，考虑到各接口个性化记录日志，没有使用拦截器
// others必须是KV格式，K为string
func (t *RpcServ) access(gctx context.Context, reqHeader *pb.ReqHeader,
	others ...interface{}) (sctx.ReqCtx, error) {
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

	// 输出access log
	logFields := make([]interface{}, 0)
	logFields = append(logFields, "from", reqHeader.GetSelfName(), "client_ip", clientIp)
	logFields = append(logFields, others...)
	rctx.GetLog().Trace("received request", logFields...)

	return rctx, nil
}

// 请求完成后处理
// others必须是KV格式，K为string
func (t *RpcServ) ending(rctx sctx.ReqCtx, respHeader *pb.RespHeader, others ...interface{}) {
	// 输出ending log
	logFields := make([]interface{}, 0)
	logFields = append(logFields, "error", respHeader.GetError(), "cost_time", rctx.GetTimer().Print())
	logFields = append(logFields, others...)
	rctx.GetLog().Info("request done", logFields...)
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
