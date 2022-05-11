package xvm

import (
	"path/filepath"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)

type HXVMCreator struct {
	tier0Creator *xvmInterpCreator
	tier1Creator *xvmCreator
	tier2Creator *xvmCreator
}

// TODO @chenfengjin
// 1. 添加 tier up 时的错误处理
// 2. 避免重复编译，以节省计算资源
// 3. 增加对应的 prometheus metric
func newHXVMCreator(creatorConfig *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	baseDir := creatorConfig.Basedir

	tier0Config := *creatorConfig
	tier1Config := *creatorConfig
	tier2Config := *creatorConfig

	tier0Config.Basedir = filepath.Join(baseDir, "tier0")
	tier1Config.Basedir = filepath.Join(baseDir, "tier1")
	tier2Config.Basedir = filepath.Join(baseDir, "tier2")

	tier1Config.VMConfig = &contract.WasmConfig{
		XVM: contract.XVMConfig{
			OptLevel: 0,
		},
	}
	tier2Config.VMConfig = &contract.WasmConfig{
		XVM: contract.XVMConfig{
			OptLevel: 2,
		},
	}

	tier0Creator, err := newXVMInterpCreator(&tier0Config)
	if err != nil {
		return nil, err
	}
	tier1Creator, err := newXVMCreator(&tier1Config)
	if err != nil {
		return nil, err
	}

	tier2Creator, err := newXVMCreator(&tier2Config)
	if err != nil {
		return nil, err
	}

	creator := &HXVMCreator{
		tier0Creator: tier0Creator.(*xvmInterpCreator),
		tier1Creator: tier1Creator.(*xvmCreator),
		tier2Creator: tier2Creator.(*xvmCreator),
	}
	return creator, nil
}

func (creator *HXVMCreator) tierUp1(ctx *bridge.Context, cp bridge.ContractCodeProvider) {
	creator.tier1Creator.CreateInstance(ctx, cp)
}

func (creator *HXVMCreator) tierUp2(ctx *bridge.Context, cp bridge.ContractCodeProvider) {
	creator.tier2Creator.CreateInstance(ctx, cp)
}

func (creator *HXVMCreator) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	codeDesc, err := cp.GetContractCodeDesc(ctx.ContractName)
	if err != nil {
		return nil, err
	}

	if _, find := creator.tier2Creator.cm.lookupDiskCache(ctx.ContractName, codeDesc); find {
		return creator.tier2Creator.CreateInstance(ctx, cp)
	}

	if _, find := creator.tier1Creator.cm.lookupDiskCache(ctx.ContractName, codeDesc); find {
		go creator.tierUp2(ctx, cp)
		return creator.tier1Creator.CreateInstance(ctx, cp)
	}

	instance, err := creator.tier0Creator.CreateInstance(ctx, cp)
	if err != nil {
		return nil, err
	}

	go creator.tierUp1(ctx, cp)
	return instance, err

}

func (creator *HXVMCreator) RemoveCache(name string) {
	creator.tier0Creator.RemoveCache(name)
	creator.tier1Creator.RemoveCache(name)
	creator.tier2Creator.RemoveCache(name)
}

func init() {
	bridge.Register(bridge.TypeWasm, "hxvm", newHXVMCreator)
}
