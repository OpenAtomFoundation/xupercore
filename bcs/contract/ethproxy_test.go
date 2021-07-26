package contract

import (
	"encoding/json"
	log15 "github.com/xuperchain/log15"
	_ "github.com/xuperchain/xupercore/bcs/contract/evm"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
	"io/ioutil"
	"testing"
)

type evmtransaction struct {
}

func TestEVM(t *testing.T) {
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
	//m := th.Manager()

	//m.GetKernRegistry().RegisterKernMethod("$evm", "proxy", new(helloContract).Hi)
	txData := evmtransaction{}
	data, err := json.Marshal(txData)
	if err != nil {
		t.Error(err)
		return
	}
	//prepare env
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
	}
	resp, err := th.Deploy("evm", "counter", "evmcounter", bin, args)
	if err != nil {
		t.Fatal(err)
	}

	// unit test
	resp, err = th.Invoke("xkernel", "$evm", "proxy", map[string][]byte{
		"desc": data,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", resp.Body)
}
