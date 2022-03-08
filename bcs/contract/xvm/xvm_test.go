package xvm

import (
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	_ "github.com/xuperchain/xupercore/kernel/contract/manager"

	"github.com/xuperchain/xupercore/kernel/contract/mock"
)

type testCase struct {
	Count     int
	wantError bool
}

func withSituation(config *contract.ContractConfig, testcases []testCase, t *testing.T) {
	th := mock.NewTestHelper(config)
	defer th.Close()

	bin, err := ioutil.ReadFile("testdata/malloc.wasm")
	if err != nil {
		t.Fatal(err)
	}
	_, err = th.Deploy("wasm", "c", "malloc", bin, map[string][]byte{})
	if err != nil {
		t.Error(err)
		return
	}
	for _, testcase := range testcases {
		resp, err := th.Invoke("wasm", "malloc", "call_malloc", map[string][]byte{
			"count": []byte(strconv.Itoa(testcase.Count)),
		})
		if testcase.wantError && (err == nil) {
			if resp.Status >= 500 {
				continue
			}
			t.Errorf("testcase %s_%d,error mismatch,want error,got nil", t.Name(), testcase.Count)
		} else if !testcase.wantError && (err != nil) {
			t.Errorf("testcase %s_%d,error mismatch,want nil,got %v", t.Name(), testcase.Count, err)
		}
	}

}
func TestXVMMemoryGrow(t *testing.T) {
	var defaultContractConfig = contract.ContractConfig{
		EnableUpgrade: true,
		Xkernel: contract.XkernelConfig{
			Enable: true,
			Driver: "default",
		},
		Wasm: contract.WasmConfig{
			Enable: true,
			Driver: "xvm",
			XVM: contract.XVMConfig{
				OptLevel: 0,
				Memory: contract.MemoryConfig{
					MemoryGrow: contract.MemoryGrowConfig{
						Enabled: false,
					},
				},
			},
		},
		LogDriver: mock.NewMockLogger(),
	}

	t.Run("DisableMemoryGrowWithoutMinimum", func(t *testing.T) {
		config := defaultContractConfig
		config.Wasm.XVM.Memory.MemoryGrow.Enabled = false
		config.Wasm.XVM.Memory.MemoryGrow.Maxmium = 65535
		testcases := []testCase{
			{
				Count:     1,
				wantError: false,
			},
			{
				Count:     32,
				wantError: true,
			},
		}
		withSituation(&config, testcases, t)

	})

	t.Run("DisableMemoryGrowWithMinimum", func(t *testing.T) {
		config := defaultContractConfig
		config.Wasm.XVM.Memory.MemoryGrow.Enabled = false
		config.Wasm.XVM.Memory.MemoryGrow.Initialize = 16
		config.Wasm.XVM.Memory.MemoryGrow.Maxmium = 65535
		testcases := []testCase{
			{
				Count:     1,
				wantError: false,
			},
			{
				Count:     3,
				wantError: false,
			},
			{
				Count:     16,
				wantError: true,
			},
		}
		withSituation(&config, testcases, t)
	})

	t.Run("MemoryGrowWithLimit", func(t *testing.T) {
		config := defaultContractConfig
		config.Wasm.XVM.Memory.MemoryGrow.Enabled = true
		config.Wasm.XVM.Memory.MemoryGrow.Maxmium = 28
		testcases := []testCase{
			{
				Count:     1,
				wantError: false,
			},
			{
				Count:     3,
				wantError: false,
			},
			{
				Count:     5,
				wantError: false,
			},
			{
				Count:     11,
				wantError: false,
			},
			{
				Count:     12,
				wantError: false,
			},
			{
				Count:     17,
				wantError: false,
			},
			{
				Count:     30,
				wantError: true,
			},
		}
		withSituation(&config, testcases, t)
	})
	// this should not appear in production environment
	t.Run("MemoryGrowWithoutLimit", func(t *testing.T) {
		config := defaultContractConfig
		config.Wasm.XVM.Memory.MemoryGrow.Enabled = true
		config.Wasm.XVM.Memory.MemoryGrow.Maxmium = 65535
		testcases := []testCase{
			{
				Count:     1,
				wantError: false,
			},
			{
				Count:     3,
				wantError: false,
			},
			{
				Count:     10,
				wantError: false,
			},

			{
				Count:     25,
				wantError: false,
			},
			{
				Count:     26,
				wantError: false,
			},
			{
				Count:     27,
				wantError: false,
			},
			{
				Count:     33,
				wantError: true,
			},
		}
		withSituation(&config, testcases, t)
	})
}
