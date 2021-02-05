package manager

import (
	"fmt"
	"sync"

	"github.com/xuperchain/xupercore/kernel/contract"
)

type registryImpl struct {
	mutex   sync.Mutex
	methods map[string]map[string]contract.KernMethod
}

func (r *registryImpl) RegisterKernMethod(ctract, method string, handler contract.KernMethod) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.methods == nil {
		r.methods = make(map[string]map[string]contract.KernMethod)
	}
	contractMap, ok := r.methods[ctract]
	if !ok {
		contractMap = make(map[string]contract.KernMethod)
		r.methods[ctract] = contractMap
	}
	_, ok = contractMap[method]
	if ok {
		panic(fmt.Sprintf("kernel method `%s' for `%s' exists", method, ctract))
	}
	contractMap[method] = handler
}

func (r *registryImpl) GetKernMethod(ctract, method string) (contract.KernMethod, error) {
	contractMap, ok := r.methods[ctract]
	if !ok {
		return nil, fmt.Errorf("kernel contract '%s' not found", ctract)
	}
	contractMethod, ok := contractMap[method]
	if !ok {
		return nil, fmt.Errorf("kernel method '%s' for '%s' not exists", method, ctract)
	}
	return contractMethod, nil
}
