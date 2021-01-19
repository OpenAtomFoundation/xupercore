package rpc

import (
	"errors"
	"fmt"
	"net"
	"sync"

	sconf "github.com/xuperchain/xupercore/example/xchain/common/config"
	"github.com/xuperchain/xupercore/example/xchain/common/def"
	pb "github.com/xuperchain/xupercore/example/xchain/common/xchainpb"
	"github.com/xuperchain/xupercore/kernel/engines"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos"
	ecom "github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// rpc server启停控制管理
type RpcServMG struct {
	scfg      *sconf.ServConf
	engine    ecom.Engine
	log       logs.Logger
	rpcServ   *RpcServ
	servHD    *grpc.Server
	tlsServHD *grpc.Server
	isInit    bool
	exitOnce  *sync.Once
}

func NewRpcServMG(scfg *sconf.ServConf, engine engines.BCEngine) (*RpcServMG, error) {
	if scfg == nil || engine == nil {
		return nil, fmt.Errorf("param error")
	}
	xosEngine, err := xuperos.EngineConvert(engine)
	if err != nil {
		return nil, fmt.Errorf("not xuperos engine")
	}

	log, _ := logs.NewLogger("", def.SubModName)
	obj := &RpcServMG{
		scfg:     scfg,
		engine:   xosEngine,
		log:      log,
		rpcServ:  NewRpcServ(engine.(ecom.Engine), log),
		isInit:   true,
		exitOnce: &sync.Once{},
	}

	return obj, nil
}

// 启动rpc服务
func (t *RpcServMG) Run() error {
	if !t.isInit {
		return errors.New("RpcServMG not init")
	}

	t.log.Trace("run grpc server")

	// 启动rpc server，阻塞直到退出
	err := t.runRpcServ()
	if err != nil {
		t.log.Error("grpc server abnormal exit", "err", err)
		return err
	}

	t.log.Trace("grpc server exit")
	return nil
}

// 退出rpc服务，释放相关资源，需要幂等
func (t *RpcServMG) Exit() {
	if !t.isInit {
		return
	}

	t.exitOnce.Do(func() {
		t.stopRpcServ()
	})
}

// 启动rpc服务，阻塞直到退出
func (t *RpcServMG) runRpcServ() error {
	rpcOptions := make([]grpc.ServerOption, 0)
	unaryInterceptors := make([]grpc.UnaryServerInterceptor, 0)
	unaryInterceptors = append(unaryInterceptors, t.rpcServ.UnaryInterceptor())
	rpcOptions = append(rpcOptions,
		middleware.WithUnaryServerChain(unaryInterceptors...),
		grpc.MaxRecvMsgSize(t.scfg.MaxRecvMsgSize),
		grpc.ReadBufferSize(t.scfg.ReadBufSize),
		grpc.InitialWindowSize(t.scfg.InitWindowSize),
		grpc.InitialConnWindowSize(t.scfg.InitConnWindowSize),
		grpc.WriteBufferSize(t.scfg.WriteBufSize),
	)

	t.servHD = grpc.NewServer(rpcOptions...)
	pb.RegisterXchainServer(t.servHD, t.rpcServ)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", t.scfg.RpcPort))
	if err != nil {
		t.log.Error("failed to listen", "err", err.Error())
		return fmt.Errorf("failed to listen")
	}

	reflection.Register(t.servHD)
	if err := t.servHD.Serve(lis); err != nil {
		t.log.Error("failed to serve", "err", err.Error())
		return err
	}

	t.log.Trace("rpc server exit")
	return nil
}

// 需要幂等
func (t *RpcServMG) stopRpcServ() {
	if t.servHD != nil {
		// 优雅关闭grpc server
		t.servHD.GracefulStop()
	}
}
