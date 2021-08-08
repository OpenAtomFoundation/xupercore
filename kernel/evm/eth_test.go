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

type MockLogger struct {
	log15.Logger
}

func (*MockLogger) GetLogId() string {
	return ""
}

func (*MockLogger) SetCommField(key string, value interface{}) {

}
func (*MockLogger) SetInfoField(key string, value interface{}) {

}

func TestEVM1(t *testing.T) {
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
		t.Error(err)
		return
	}

	bin, err := ioutil.ReadFile("testdata/counter.bin")
	if err != nil {
		t.Error(err)
		return
	}
	abi, err := ioutil.ReadFile("testdata/counter.abi")
	if err != nil {
		t.Error(err)
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
		t.Fatal(err)
	}
	resp, err := th.Deploy("evm", "counter", "counter", data, args)
	if err != nil {
		// TODO
		t.Fatal(err)
	}

	t.Run("SendTransaction", func(t *testing.T) {
		resp, err = th.Invoke("xkernel", "$evm", "SendTransaction", map[string][]byte{
			"desc": data,
		})
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("SendRawTransaction", func(t *testing.T) {
		resp, err = th.Invoke("xkernel", "$evm", "SendTransaction", map[string][]byte{
			"desc": data,
		})
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("ContractCall", func(t *testing.T) {
		resp, err = th.Invoke("xkernel", "$evm", "SendTransaction", map[string][]byte{
			"desc": data,
		})
		if err != nil {
			t.Error(err)
			return
		}
	})
	// TODO
	_ = resp
}