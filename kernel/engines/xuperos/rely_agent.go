package xuperos

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	//"github.com/xuperchain/xupercore/kernel/network"
)

// 代理依赖组件实例化操作，方便mock单测和并行开发
type RelyAgentImpl struct {
	engine def.Engine
}

func NewRelyAgent(engine def.Engine) *RelyAgentImpl {
	return &RelyAgentImpl{engine}
}

func (t *RelyAgentImpl) CreateNetwork() (def.XNetwork, error) {
	/*
			envCfg := t.engine.GetEngineCtx().EnvCfg
		    netCtx, err := nctx.CreateNetCtx(envCfg.GenConfFilePath(envCfg.NetConf))
		    if err != nil {
		        return nil, fmt.Errorf("create engine ctx failed because create network ctx failed.err:%v", err)
		    }
		    netHD, err := network.CreateNetwork(netCtx)
		    if err != nil {
		        return nil, fmt.Errorf("create engine ctx failed because create network failed.err:%v", err)
		    }
	*/

	return nil, fmt.Errorf("the interface is not implemented")
}

func (t *RelyAgentImpl) CreateLedger() (def.XLedger, error) {
	return nil, fmt.Errorf("the interface is not implemented")
}
