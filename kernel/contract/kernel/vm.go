package kernel

import (
	"github.com/xuperchain/xupercore/contractsdk/go/pb"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)

type kernvm struct {
	registry contract.KernRegistry
}

func newKernvm(config *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	return &kernvm{}, nil
}

// CreateInstance instances a wasm virtual machine instance which can run a single contract call
func (k *kernvm) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	method, err := k.registry.GetKernMethod(ctx.ContractName, ctx.Method)
	if err != nil {
		return nil, err
	}
	return newKernInstance(ctx, method), nil
}

func (k *kernvm) RemoveCache(name string) {
}

type kernInstance struct {
	ctx    *bridge.Context
	kctx   *kcontextImpl
	method contract.KernMethod
}

func newKernInstance(ctx *bridge.Context, method contract.KernMethod) *kernInstance {
	return &kernInstance{
		ctx:    ctx,
		kctx:   newKContext(ctx),
		method: method,
	}
}

func (k *kernInstance) Exec() error {
	resp, err := k.method(k.kctx)
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
