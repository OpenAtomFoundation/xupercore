package xvm

import (
	"time"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)

type HXVMCreator struct {
	creator  bridge.InstanceCreator
	lastSeen time.Time
	count    int64
	config   *bridge.InstanceCreatorConfig
}

func newHXVMCreator(creatorConfig *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	ixvm, err := newXVMInterpCreator(creatorConfig)
	if err != nil {
		return nil, err
	}
	creator := &HXVMCreator{
		creator: ixvm,
	}
	go creator.tierUp1()
	return creator, nil
}

func (creator *HXVMCreator) tierUp1() {
	config := creator.config
	aotxvm, err := newXVMCreator(creator.config)
	config.VMConfig = &contract.WasmConfig{
		XVM: contract.XVMConfig{
			OptLevel: 0,
		},
	}
	if err != nil {

	}
	creator.creator = aotxvm
}

func (creator *HXVMCreator) tierUp2() {
	config := creator.config
	config.VMConfig = &contract.WasmConfig{
		XVM: contract.XVMConfig{
			OptLevel: 3,
		},
	}
	aotxvm, err := newXVMCreator(creator.config)
	if err != nil {

	}
	creator.creator = aotxvm
}

func (vm *HXVMCreator) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	if time.Now().Sub(vm.lastSeen) < time.Minute {
		if vm.count > 10 {
			go vm.tierUp2()
		}
	} else {
		vm.lastSeen = time.Now()
	}
	return vm.creator.CreateInstance(ctx, cp)
}

func (vm *HXVMCreator) RemoveCache(name string) {
	vm.creator.RemoveCache(name)
}

func init() {
	bridge.Register(bridge.TypeWasm, "hxvm", newHXVMCreator)
}
