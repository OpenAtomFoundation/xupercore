package geth

import (
	"os"

	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/xuperchain/xupercore/protos"
)

type mockContractCodeProvider struct {
}

func (m mockContractCodeProvider) GetContractCodeDesc(name string) (*protos.WasmCodeDesc, error) {
	panic("implement me")
}

func (m mockContractCodeProvider) GetContractCode(name string) ([]byte, error) {
	switch name {
	case "counter", "storage":
		return binCode(name)
	default:
		return minimumCode(), nil
	}
}

func (m mockContractCodeProvider) GetContractAbi(name string) ([]byte, error) {
	return nil, nil
}

func (m mockContractCodeProvider) GetContractCodeFromCache(name string) ([]byte, error) {
	panic("implement me")
}

func (m mockContractCodeProvider) GetContractAbiFromCache(name string) ([]byte, error) {
	panic("implement me")
}

func binCode(name string) ([]byte, error) {
	return os.ReadFile("testdata/" + name + ".bin")
}

func minimumCode() []byte {
	return []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x0, byte(vm.SSTORE)}
}
