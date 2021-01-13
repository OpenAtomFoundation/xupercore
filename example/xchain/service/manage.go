package service

import (
	"fmt"

	sconf "github.com/xuperchain/xupercore/example/xchain/common/config"
	"github.com/xuperchain/xupercore/example/xchain/common/def"
	"github.com/xuperchain/xupercore/example/xchain/service/rpc"
	"github.com/xuperchain/xupercore/kernel/engines"
	"github.com/xuperchain/xupercore/lib/logs"
)

// 由于需要同时启动多个服务组件，采用注册机制管理
type ServCom interface {
	Run() error
	Exit()
}

// 各server组件运行控制
type ServMG struct {
	scfg    *sconf.ServConf
	log     logs.Logger
	servers []ServCom
}

func NewServMG(scfg *sconf.ServConf, engine engines.BCEngine) (*ServMG, error) {
	if scfg == nil || engine == nil {
		return nil, fmt.Errorf("param error")
	}

	log, _ := logs.NewLogger("", def.SubModName)
	obj := &ServMG{
		scfg:    scfg,
		log:     log,
		servers: make([]ServCom, 0),
	}

	// 实例化rpc服务
	rpcServ, err := rpc.NewRpcServMG(scfg, engine)
	if err != nil {
		return nil, err
	}
	obj.servers = append(obj.servers, rpcServ)

	return obj, nil
}

// 启动rpc服务
func (t *ServMG) Run() error {
	ch := make(chan error, 0)
	defer close(ch)

	for _, serv := range t.servers {
		// 启动各个service
		go func(s ServCom) {
			ch <- s.Run()
		}(serv)
	}

	// 监听各个service状态
	exitCnt := 0
	for {
		if exitCnt >= len(t.servers) {
			break
		}

		select {
		case <-ch:
			exitCnt++
		}
	}

	return nil
}

// 退出rpc服务，释放相关资源，需要幂等
func (t *ServMG) Exit() {
	for _, serv := range t.servers {
		// 触发各service退出
		go func(s ServCom) {
			s.Exit()
		}(serv)
	}
}
