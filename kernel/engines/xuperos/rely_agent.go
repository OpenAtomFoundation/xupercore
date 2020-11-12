package xuperos

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/kernel/network"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
)

// 代理依赖组件实例化操作，方便mock单测和并行开发
type EngineRelyAgentImpl struct {
	engine def.Engine
}

func NewEngineRelyAgent(engine def.Engine) *EngineRelyAgentImpl {
	return &EngineRelyAgentImpl{engine}
}

// 创建并启动p2p网络
func (t *EngineRelyAgentImpl) CreateNetwork() (network.Network, error) {

	envCfg := t.engine.GetEngineCtx().EnvCfg
	netCtx, err := nctx.CreateNetCtx(envCfg.GenConfFilePath(envCfg.NetConf))
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because create network ctx failed.err:%v", err)
	}

	netHD, err := network.CreateNetwork("default", netCtx)
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because create network failed.err:%v", err)
	}

	return netHD, nil
}

func (t *EngineRelyAgentImpl) CreateLedger() (def.XLedger, error) {
	return nil, fmt.Errorf("the interface is not implemented")
}

// 代理依赖组件实例化操作，方便mock单测和并行开发
type ChainRelyAgentImpl struct {
	chain def.Chain
}

func NewChainRelyAgent(chain def.Chain) *ChainRelyAgentImpl {
	return &ChainRelyAgentImpl{chain}
}

// 创建并启动p2p网络
func (t *ChainRelyAgentImpl) CreateContract() (contract.Manager, error) {
	return contract.CreateManager("default", new(chainCore))
}

// chainCore is the implement of contract.ChainCore
type chainCore struct {
}

// BCName get blockchain name
func (c *chainCore) BCName() string {
	return ""
}

// GetAccountAddress get addresses associated with account name
func (c *chainCore) GetAccountAddresses(accountName string) ([]string, error) {
	return []string{}, nil
}

// VerifyContractPermission verify permission of calling contract
func (c *chainCore) VerifyContractPermission(initiator string, authRequire []string, contractName, methodName string) (bool, error) {
	return true, nil
}

// VerifyContractOwnerPermission verify contract ownership permisson
func (c *chainCore) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	return nil
}
