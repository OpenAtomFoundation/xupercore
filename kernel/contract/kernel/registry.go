package kernel

import (
	"fmt"
	"sync"

	"github.com/xuperchain/xupercore/kernel/contract"
)

type Registry interface {
	RegisterKernMethod(contract, method string, handler KernMethod)
	GetKernMethod(contract, method string) (KernMethod, error)
}

type KernMethod func(ctx KContext) (*contract.Response, error)

var (
	DefaultRegistry Registry = &registryImpl{}
)

type registryImpl struct {
	mutex   sync.Mutex
	methods map[string]map[string]KernMethod
}

func (r *registryImpl) RegisterKernMethod(contract, method string, handler KernMethod) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.methods == nil {
		r.methods = make(map[string]map[string]KernMethod)
	}
	contractMap, ok := r.methods[contract]
	if !ok {
		contractMap = make(map[string]KernMethod)
		r.methods[contract] = contractMap
	}
	_, ok = contractMap[method]
	if ok {
		panic(fmt.Sprintf("kernel method %s for %s exists", method, contract))
	}
	contractMap[method] = handler
}

func (r *registryImpl) GetKernMethod(contract, method string) (KernMethod, error) {
	contractMap, ok := r.methods[contract]
	if !ok {
		return nil, fmt.Errorf("kernel contract %s not found", contract)
	}
	contractMethod, ok := contractMap[method]
	if !ok {
		return nil, fmt.Errorf("kernel method %s for %s exists", method, contract)
	}
	return contractMethod, nil
}

func RegisterKernMethod(contract, method string, handler KernMethod) {
	DefaultRegistry.RegisterKernMethod(contract, method, handler)
}
