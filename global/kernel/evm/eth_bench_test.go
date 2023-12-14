package evm

import (
	_ "github.com/OpenAtomFoundation/xupercore/global/bcs/contract/evm"
	_ "github.com/OpenAtomFoundation/xupercore/global/bcs/contract/native"
	_ "github.com/OpenAtomFoundation/xupercore/global/bcs/contract/xvm"

	"encoding/hex"
	"io/ioutil"
	"testing"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract"
	_ "github.com/OpenAtomFoundation/xupercore/global/kernel/contract"
	_ "github.com/OpenAtomFoundation/xupercore/global/kernel/contract/kernel"
	_ "github.com/OpenAtomFoundation/xupercore/global/kernel/contract/manager"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract/mock"
)

func BenchmarkEVM(b *testing.B) {
	var contractConfig = &contract.ContractConfig{
		EnableUpgrade: true,
		Xkernel: contract.XkernelConfig{
			Enable: true,
			Driver: "default",
		},
		Native: contract.NativeConfig{
			Enable: true,
			Driver: "native",
		},
		EVM: contract.EVMConfig{
			Enable: true,
			Driver: "evm",
		},
		LogDriver: mock.NewMockLogger(),
	}
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()

	bin, err := ioutil.ReadFile("testdata/counter.bin")
	if err != nil {
		b.Error(err)
		return
	}
	abi, err := ioutil.ReadFile("testdata/counter.abi")
	if err != nil {
		b.Error(err)
		return
	}
	args := map[string][]byte{
		"contract_abi": abi,
		"input":        bin,
		"jsonEncoded":  []byte("false"),
	}
	data, err := hex.DecodeString(string((bin)))
	if err != nil {
		b.Fatal(err)
	}
	resp, err := th.Deploy("evm", "counter", "counter", data, args)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("Benchmark", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := th.Invoke("evm", "counter", "increase", map[string][]byte{
				"input":       []byte(`{"key":"xchain"}`),
				"jsonEncoded": []byte("true"),
			})
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
	_ = resp

}
