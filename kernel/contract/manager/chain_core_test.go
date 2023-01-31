package manager

// we put bridge feature test here as a compromise of cycle import
import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/big"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	log15 "github.com/xuperchain/log15"
	_ "github.com/xuperchain/xupercore/bcs/contract/native"
	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
)

const (
	callerContractName = "caller"
)

func compile(th *mock.TestHelper) ([]byte, error) {
	target := filepath.Join(th.Basedir(), "counter.bin")
	goModCmd := exec.Command("go", "mod", "tidy")
	goModCmd.Dir = "testdata"
	err := goModCmd.Run()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("go", "build", "-o", target)
	cmd.Dir = "testdata"
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s:%s", err, out)
	}
	bin, err := ioutil.ReadFile(target)
	if err != nil {
		return nil, err
	}
	return bin, nil
}

func TestBridgeFeatures(t *testing.T) {
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
		LogDriver: &mock.MockLogger{
			logger,
		},
	}
	buffer := bytes.NewBuffer([]byte{})
	logger.SetHandler(log15.StreamHandler(buffer, log15.LogfmtFormat()))

	th := mock.NewTestHelper(contractConfig)
	defer th.Close()

	bin, err := compile(th)
	if err != nil {
		t.Fatal(err)
	}

	_, err = th.Deploy("native", "go", mock.FeaturesContractName, bin, map[string][]byte{
		"creator": []byte("icexin"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Logging", func(t *testing.T) {
		resp, err := th.Invoke("native", "features", "Logging", map[string][]byte{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buffer.String(), "log from contract") {
			t.Error(resp.Status, resp.Body, resp.Message)
		}
	})
	t.Run("Transfer", func(t *testing.T) {
		resp, err := th.Invoke("native", "features", "Transfer", map[string][]byte{
			"to":        []byte(mock.ContractAccount2),
			"amount":    []byte(big.NewInt(100).String()),
			"initiator": []byte(mock.ContractAccount),
		})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Status > 400 {
			t.Error(resp.Message)
		}
		uwset := th.UTXOState().WSet
		{
			if new(big.Int).SetBytes(uwset[0].Amount).Int64() != 100 || string(uwset[0].ToAddr) != mock.ContractAccount2 {
				t.Error("transfer error")
			}
		}
		{
			if new(big.Int).SetBytes(uwset[1].Amount).Int64() != 9899 || string(uwset[1].ToAddr) != mock.FeaturesContractName {
				fmt.Println(new(big.Int).SetBytes(uwset[0].Amount).Int64())
				fmt.Println(string(uwset[0].ToAddr))
				t.Error("transfer error")
			}
		}

	})
	t.Run("QueryTx", func(t *testing.T) {
		resp, err := th.Invoke("native", "features", "QueryTx", map[string][]byte{})
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(string(resp.Body))
	})
	t.Run("QueryBlock", func(t *testing.T) {
		resp, err := th.Invoke("native", "features", "QueryBlock", map[string][]byte{})
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(string(resp.Body))
	})
}

func TestContractCall(t *testing.T) {
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
		LogDriver: &mock.MockLogger{
			logger,
		},
	}
	buffer := bytes.NewBuffer([]byte{})
	logger.SetHandler(log15.StreamHandler(buffer, log15.LogfmtFormat()))

	th := mock.NewTestHelper(contractConfig)
	defer th.Close()

	bin, err := compile(th)
	if err != nil {
		t.Fatal(err)
	}

	_, err = th.Deploy("native", "go", mock.FeaturesContractName, bin, map[string][]byte{
		"creator": []byte("xchain"),
	})

	_, err = th.Deploy("native", "go", callerContractName, bin, map[string][]byte{})
	if err != nil {
		t.Error(err)
	}

	t.Run("DirectCall", func(t *testing.T) {
		resp, err := th.Invoke("native", mock.FeaturesContractName, "Caller", map[string][]byte{})
		if err != nil {
			t.Error(err)
			return
		}
		if string(resp.Body) != mock.ContractAccount {
			t.Errorf("want %s, got %s", mock.ContractAccount, string(resp.Body))
			return
		}
	})

	t.Run("ContractCall", func(t *testing.T) {
		resp, err := th.Invoke("native", mock.FeaturesContractName, "Invoke", map[string][]byte{
			"contract": []byte(callerContractName),
			"method":   []byte("Caller"),
		})
		if err != nil {
			t.Error(err)
			return
		}

		if string(resp.Body) != mock.FeaturesContractName {
			t.Errorf("want %s,got %s\n", mock.FeaturesContractName, string(resp.Body))
		}
	})
	t.Run("RecursiongCall", func(t *testing.T) {
		resp, err := th.Invoke("native", mock.FeaturesContractName, "Invoke", map[string][]byte{
			"contract": []byte(mock.FeaturesContractName),
			"method":   []byte("Caller"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if resp.Status < contract.StatusErrorThreshold {
			t.Error("recursive contract call not permitted")
		}
	})
}
