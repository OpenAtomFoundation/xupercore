package kernel

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
)

type kernvm struct {
	registry contract.KernRegistry
	config   *bridge.InstanceCreatorConfig
}

func newKernvm(config *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	return &kernvm{
		registry: config.VMConfig.(*contract.XkernelConfig).Registry,
		config:   config,
	}, nil
}

// CreateInstance instances a wasm virtual machine instance which can run a single contract call
func (k *kernvm) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	return newKernInstance(ctx, k.config.SyscallService, k.registry), nil
}

func (k *kernvm) RemoveCache(name string) {
}

type kernInstance struct {
	ctx      *bridge.Context
	kctx     *kcontextImpl
	registry contract.KernRegistry
}

func newKernInstance(ctx *bridge.Context, syscall *bridge.SyscallService, registry contract.KernRegistry) *kernInstance {
	return &kernInstance{
		ctx:      ctx,
		kctx:     newKContext(ctx, syscall),
		registry: registry,
	}
}

func (k *kernInstance) Exec() error {
	method, err := k.registry.GetKernMethod(k.ctx.ContractName, k.ctx.Method)
	if err != nil {
		return err
	}

	resp, err := method(k.kctx)
	if err != nil {
		return err
	}
	k.ctx.Output = &pb.Response{
		Status:  int32(resp.Status),
		Message: resp.Message,
		Body:    resp.Body,
	}
	return nil
}

func (k *kernInstance) ResourceUsed() contract.Limits {
	return k.kctx.used
}

func (k *kernInstance) Release() {
}

func (k *kernInstance) Abort(msg string) {
}

func init() {
	bridge.Register("xkernel", "default", newKernvm)
}
