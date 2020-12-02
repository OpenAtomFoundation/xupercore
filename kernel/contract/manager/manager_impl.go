package manager

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
)

type managerImpl struct {
	core    contract.ChainCore
	xbridge *bridge.XBridge
}

func newManagerImpl(cfg *contract.ManagerConfig) (contract.Manager, error) {
	m := &managerImpl{
		core: cfg.Core,
	}
	registry := m.core.KernRegistry().(kernel.Registry)
	registry.RegisterKernMethod("contract", "deployContract", m.deployContract)
	registry.RegisterKernMethod("contract", "upgradeContract", m.deployContract)
	return m, nil
}

func (m *managerImpl) NewContext(cfg *contract.ContextConfig) (contract.Context, error) {
	return m.xbridge.NewContext(cfg)
}

func (m *managerImpl) NewStateSandbox(r contract.XMStateReader) (contract.XMStateSandbox, error) {
	return nil, nil
}

func (m *managerImpl) deployContract(ctx kernel.KContext) (*contract.Response, error) {
	resp, limit, err := m.xbridge.DeployContract(ctx)
	if err != nil {
		return nil, err
	}
	ctx.AddResourceUsed(limit)
	return resp, nil
}

func (m *managerImpl) upgradeContract(ctx kernel.KContext) (*contract.Response, error) {
	resp, limit, err := m.xbridge.UpgradeContract(ctx)
	if err != nil {
		return nil, err
	}
	ctx.AddResourceUsed(limit)
	return resp, nil
}

func init() {
	contract.Register("default", newManagerImpl)
}
