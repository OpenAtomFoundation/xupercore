package native

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/protos"
)

type processManager struct {
	cfg       *contract.NativeConfig
	basedir   string
	chainAddr string
	mutex     sync.Mutex
	contracts map[string]*contractProcess
}

func newProcessManager(cfg *contract.NativeConfig, basedir string, chainAddr string) (*processManager, error) {
	return &processManager{
		cfg:       cfg,
		basedir:   basedir,
		chainAddr: chainAddr,
		contracts: make(map[string]*contractProcess),
	}, nil
}

func (p *processManager) makeProcess(name string, desc *protos.WasmCodeDesc, code []byte) (*contractProcess, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	hash := nativeCodeHash(name, desc)
	process, ok := p.contracts[hash]
	if ok {
		process.Stop()
	}
	delete(p.contracts, hash)

	processDir := filepath.Join(p.basedir, name)
	err := os.MkdirAll(processDir, 0755)
	if err != nil {
		return nil, err
	}
	contractFile := nativeCodeFileName(desc)
	processBin := filepath.Join(processDir, contractFile)
	err = ioutil.WriteFile(processBin, code, 0755)
	if err != nil {
		return nil, err
	}

	process, err = newContractProcess(p.cfg, name, processDir, p.chainAddr, desc)
	if err != nil {
		return nil, err
	}

	err = process.Start()
	if err != nil {
		return nil, err
	}
	p.contracts[hash] = process

	return process, nil
}

func (p *processManager) lookupProcess(name string, desc *protos.WasmCodeDesc) (*contractProcess, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	hash := nativeCodeHash(name, desc)
	process, ok := p.contracts[hash]
	if !ok {
		return nil, false
	}
	return process, true
}

func (p *processManager) GetProcess(name string, cp bridge.ContractCodeProvider) (*contractProcess, error) {
	desc, err := cp.GetContractCodeDesc(name)
	if err != nil {
		return nil, err
	}

	process, ok := p.lookupProcess(name, desc)
	if ok {
		return process, nil
	}

	code, err := cp.GetContractCode(name)
	if err != nil {
		return nil, err
	}

	process, err = p.makeProcess(name, desc, code)
	if err != nil {
		return nil, err
	}
	return process, nil
}

func nativeCodeHash(name string, desc *protos.WasmCodeDesc) string {
	return name + hex.EncodeToString(desc.GetDigest())
}

func nativeCodeFileName(desc *protos.WasmCodeDesc) string {
	var suffix string
	switch desc.GetRuntime() {
	case "java":
		suffix = ".jar"
	}
	hash := hex.EncodeToString(desc.GetDigest()[0:3])
	return "nativecode-" + hash + suffix
}
