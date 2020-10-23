package rpc

import (
	"errors"

	"github.com/xuperchain/xupercore/kernel/engines"
	"github.com/xuperchain/xupercore/lib/logs"
	sconf "github.com/xuperchain/xupercore/server/config"

	"google.golang.org/grpc"
)

// rpc server启停控制管理
type RpcServMG struct {
	scfg    *sconf.ServConf
	engine  engines.BCEngine
	log     logs.Logger
	rpcServ *RpcServer
	servHD  *grpc.Server
	isInit  bool
	exitCh  chan error
}

func NewRpcServMG(scfg *sconf.ServConf, engine engines.BCEngine) *RpcServMG {
	log := logs.NewLogger("", SubModName)
	return &RpcServMG{
		scfg:    scfg,
		engine:  engine,
		log:     log,
		rpcServ: NewRpcServ(engine, log),
		exitCh:  make(chan error),
	}
}

// 启动rpc服务
func (t *RpcServMG) Run() <-chan error {
	if !t.isInit {
		t.exitCh <- errors.New("RpcServMG not init")
		return t.exitCh
	}

	// 启动rpc server，阻塞直到退出
	err := t.RunRpcServ()
	if err != nil {
		t.logger.Error("grpc server abnormal exit.err:%v", err)
	}
	t.exitCh <- err

	return t.exitCh
}

// 退出rpc服务，释放相关资源
func (t *RpcServMG) Exit() {
	if !t.isInit {
		return
	}

	t.StopRpcServ()
}

func (t *RpcServMG) RunRpcServ() error {

}

func (t *RpcServMG) StopRpcServ() {
	if t.servHD != nil {
		// 优雅关闭grpc server
		t.servHD.GracefulStop()
	}
}
