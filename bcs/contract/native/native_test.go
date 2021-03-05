package native

import (
	//"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	//"testing"

	//"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	_ "github.com/xuperchain/xupercore/kernel/contract/manager"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/kernel/ledger"
	//"github.com/xuperchain/xupercore/protos"
)

type testHelper struct {
	basedir string

	state   ledger.XMReader
	manager contract.Manager
}

func newTestHelper() *testHelper {
	basedir, err := ioutil.TempDir("", "native-test")
	if err != nil {
		panic(err)
	}

	state := sandbox.NewMemXModel()

	m, err := contract.CreateManager("default", &contract.ManagerConfig{
		Basedir:  basedir,
		BCName:   "xuper",
		Core:     new(fakeChainCore),
		XMReader: state,
	})
	if err != nil {
		panic(err)
	}

	th := &testHelper{
		basedir: basedir,
		manager: m,
		state:   state,
	}
	return th
}

func (t *testHelper) Compile() ([]byte, error) {
	target := filepath.Join(t.basedir, "counter.bin")
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

func (t *testHelper) Manager() contract.Manager {
	return t.manager
}

func (t *testHelper) Basedir() string {
	return t.basedir
}

func (t *testHelper) State() ledger.XMReader {
	return t.state
}

func (t *testHelper) Close() {
	os.RemoveAll(t.basedir)
}

type fakeChainCore struct {
}

// GetAccountAddress get addresses associated with account name
func (f *fakeChainCore) GetAccountAddresses(accountName string) ([]string, error) {
	panic("not implemented") // TODO: Implement
}

// VerifyContractPermission verify permission of calling contract
func (f *fakeChainCore) VerifyContractPermission(initiator string, authRequire []string, contractName string, methodName string) (bool, error) {
	panic("not implemented") // TODO: Implement
}

// VerifyContractOwnerPermission verify contract ownership permisson
func (f *fakeChainCore) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	panic("not implemented") // TODO: Implement
}

/*func TestDeployNative(t *testing.T) {
	th := newTestHelper()
	defer th.Close()

	m := th.Manager()

	bin, err := th.Compile()
	if err != nil {
		t.Fatal(err)
	}

	state, err := m.NewStateSandbox(&contract.SandboxConfig{
		XMReader: th.State(),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := m.NewContext(&contract.ContextConfig{
		Module:         "xkernel",
		ContractName:   "$contract",
		State:          state,
		ResourceLimits: contract.MaxLimits,
	})
	if err != nil {
		t.Fatal(err)
	}

	desc := &protos.WasmCodeDesc{
		Runtime:      "go",
		ContractType: "native",
	}
	descbuf, _ := proto.Marshal(desc)

	initArgs := map[string][]byte{"creator": []byte("icexin")}
	argsBuf, _ := json.Marshal(initArgs)

	resp, err := ctx.Invoke("deployContract", map[string][]byte{
		"account_name":  []byte("XC111111@xuper"),
		"contract_name": []byte("counter"),
		"contract_code": bin,
		"contract_desc": descbuf,
		"init_args":     argsBuf,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", resp)
	ctx.Release()

	ctx, err = m.NewContext(&contract.ContextConfig{
		Module:         "native",
		ContractName:   "counter",
		State:          state,
		ResourceLimits: contract.MaxLimits,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err = ctx.Invoke("Increase", map[string][]byte{
		"key": []byte("icexin"),
	})
	t.Logf("%#v", resp)
	ctx.Release()

}*/
