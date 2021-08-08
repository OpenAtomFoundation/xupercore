package evm

import (
	log15 "github.com/xuperchain/log15"
	_ "github.com/xuperchain/xupercore/bcs/contract/evm"
	_ "github.com/xuperchain/xupercore/bcs/contract/native"
	_ "github.com/xuperchain/xupercore/bcs/contract/xvm"

	"encoding/hex"
	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	_ "github.com/xuperchain/xupercore/kernel/contract/manager"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
	"io/ioutil"
	"testing"
)

//type MockLogger struct {
//	log15.Logger
//}
//
//func (*MockLogger) GetLogId() string {
//	return ""
//}
//
//func (*MockLogger) SetCommField(key string, value interface{}) {
//
//}
//func (*MockLogger) SetInfoField(key string, value interface{}) {
//
//}

func BenchmarkEVM(b *testing.B) {
	var logger = log15.New()
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
		LogDriver: &MockLogger{
			logger,
		},
	}
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()
	_, err := NewEVMProxy(m)
	if err != nil {
		b.Error(err)
		return
	}

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
	// TODO WTF
	data, err := hex.DecodeString(string((bin)))
	if err != nil {
		b.Fatal(err)
	}
	resp, err := th.Deploy("evm", "counter", "counter", data, args)
	if err != nil {
		// TODO
		b.Fatal(err)
	}
	b.Run("Benchmark", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := th.Invoke("evm", "counter", "testWrite", map[string][]byte{
				"input":       []byte(`{"_name":"xchain"}`),
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
