package contract

import (
	"encoding/json"
	log15 "github.com/xuperchain/log15"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
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
	resp, err := th.Invoke("xkernel", "$evm", "proxy", map[string][]byte{
		"desc": data,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", resp.Body)
}
