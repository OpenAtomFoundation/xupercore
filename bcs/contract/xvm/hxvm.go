package xvm

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)

type HXVM struct {
	ixvm  bridge.InstanceCreator
	aotvm bridge.InstanceCreator
}
type HVMInstance struct {
	iinstance   bridge.Instance
	aotinstance bridge.Instance
	aotReady    bool
}

func (i *HVMInstance) Exec() error {
	//1.AOT ready 直接用AOT
	// 2. AOT 未 Ready 非初始化方法直接用ixvm
	//TODO
	// 确定 method,包括部署和升级
	// 部署可能失败，后续有重试
	// 考虑交易验证的清醒
	// 初始化方法会重复执行
	method := ""
	// 并发问题
	if i.aotReady {
		return i.aotinstance.Exec()
	}
	if method != "initialize" {
		return i.iinstance.Exec()
	}
	// 初始化才使用hvm
	// TODO 后续可以考虑用 JIT 方案 ????
	// TODO
	ready := make(chan<- bool, 1)
	go func() {
		i.aotinstance.Exec()
		i.aotReady = true
		//TODO
		//	清理ixvm 相关资源
		//考虑 gas 消耗 问题
	}()
	return i.iinstance.Exec()

}
func (i *HVMInstance) ResourceUsed() contract.Limits {
	// TODO
	return contract.Limits{}
}
func (i *HVMInstance) Release() {
	//	 TODO
}
func (i *HVMInstance) Abort(msg string) {
	//TODO
}
func newHXVMCreator(creatorConfig *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	ixvm, err := newXVMInterpCreator(creatorConfig)
	if err != nil {
		return nil, err
	}
	aotvm, err := newXVMCreator(creatorConfig)
	if err != nil {
		return nil, err
	}
	return &HXVM{
		ixvm:  ixvm,
		aotvm: aotvm,
	}, nil
}

func (vm *HXVM) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	iinstance, err := vm.ixvm.CreateInstance(ctx, cp)
	if err != nil {
		return nil, err
	}
	aotInstance, err := vm.ixvm.CreateInstance(ctx, cp)
	if err != nil {
		return nil, err
	}
	return &HVMInstance{
		iinstance:   iinstance,
		aotinstance: aotInstance,
	}, nil
}

func (vm *HXVM) RemoveCache(name string) {
	// TODO
}

func init() {
	bridge.Register(bridge.TypeWasm, "hxvm", newHXVMCreator)
}
