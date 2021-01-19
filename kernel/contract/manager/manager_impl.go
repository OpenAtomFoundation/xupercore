package manager

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
)

type managerImpl struct {
	core      contract.ChainCore
	xbridge   *bridge.XBridge
	kregistry registryImpl
}

func newManagerImpl(cfg *contract.ManagerConfig) (contract.Manager, error) {
	m := &managerImpl{
		core: cfg.Core,
	}
	xbridge, err := bridge.New(&bridge.XBridgeConfig{
		VMConfigs: map[bridge.ContractType]bridge.VMConfig{
			// bridge.TypeWasm: &bridge.WasmConfig{
			// 	Driver: "ixvm",
			// },
			// bridge.TypeNative: &bridge.NativeConfig{
			// 	Driver: "native",
			// 	Enable: true,
			// },
			// bridge.TypeEvm: &bridge.EVMConfig{
			// 	Enable: false,
			// },
			bridge.TypeKernel: &bridge.XkernelConfig{
				Driver:   "default",
				Enable:   true,
				Registry: &m.kregistry,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	m.xbridge = xbridge
	registry := &m.kregistry
	registry.RegisterKernMethod("$contract", "deployContract", m.deployContract)
	registry.RegisterKernMethod("$contract", "upgradeContract", m.deployContract)
	return m, nil
}

func (m *managerImpl) NewContext(cfg *contract.ContextConfig) (contract.Context, error) {
	return m.xbridge.NewContext(cfg)
}

func (m *managerImpl) NewStateSandbox(cfg *contract.SandboxConfig) (contract.StateSandbox, error) {
	return sandbox.NewXModelCache(cfg.XMReader)
}

func (m *managerImpl) GetKernRegistry() contract.KernRegistry {
	return &m.kregistry
}

func (m *managerImpl) deployContract(ctx contract.KContext) (*contract.Response, error) {
	resp, limit, err := m.xbridge.DeployContract(ctx)
	if err != nil {
		return nil, err
	}
	ctx.AddResourceUsed(limit)
	return resp, nil
}

func (m *managerImpl) upgradeContract(ctx contract.KContext) (*contract.Response, error) {
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
