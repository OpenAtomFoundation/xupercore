package manager

import (
	"fmt"
	"sync"

	"github.com/xuperchain/xupercore/kernel/contract"
)

type shortcut struct {
	OldMethod string
	Contract  string
	Method    string
}

type registryImpl struct {
	mutex     sync.Mutex
	methods   map[string]map[string]contract.KernMethod
	shortcuts map[string]shortcut
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

func (r *registryImpl) RegisterShortcut(oldmethod, contract, method string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.shortcuts == nil {
		r.shortcuts = make(map[string]shortcut)
	}
	_, ok := r.shortcuts[oldmethod]
	if ok {
		panic(fmt.Sprintf("kernel method shortcut for '%s' exists", oldmethod))
	}
	r.shortcuts[oldmethod] = shortcut{
		OldMethod: oldmethod,
		Contract:  contract,
		Method:    method,
	}
}

func (r *registryImpl) getShortcut(method string) (shortcut, error) {
	sc, ok := r.shortcuts[method]
	if !ok {
		return shortcut{}, fmt.Errorf("kernel method for `%s' not found", method)
	}
	return sc, nil
}

func (r *registryImpl) GetKernMethod(ctract, method string) (contract.KernMethod, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if ctract == "" {
		sc, err := r.getShortcut(method)
		if err != nil {
			return nil, err
		}
		ctract, method = sc.Contract, sc.Method
	}
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
