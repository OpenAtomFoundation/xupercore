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

func newManagerImpl(core contract.ChainCore) (contract.Manager, error) {
	return &managerImpl{
		core: core,
	}, nil
}

func (m *managerImpl) NewContext(cfg *contract.ContextConfig) (contract.Context, error) {
	return m.xbridge.NewContext(cfg)
}

func (m *managerImpl) deployContract(ctx *kernel.KContext) (*contract.Response, error) {
	// m.xbridge.DeployContract()
	return nil, nil
}

func (m *managerImpl) upgradeContract(ctx *kernel.KContext) (*contract.Response, error) {
	// m.xbridge.UpgradeContract()
	return nil, nil
}

func init() {
	contract.Register("default", newManagerImpl)
}
