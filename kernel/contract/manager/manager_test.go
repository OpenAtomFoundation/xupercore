package manager

import (
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/xuperchain/xupercore/bcs/contract/native"
	_ "github.com/xuperchain/xupercore/bcs/contract/xvm"
	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
)

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
func TestCreate(t *testing.T) {
	tmpdir, _ := ioutil.TempDir("", "contract-test")
	defer os.RemoveAll(tmpdir)

	_, err := contract.CreateManager("default", &contract.ManagerConfig{
		Basedir:  tmpdir,
		BCName:   "xuper",
		Core:     new(fakeChainCore),
		XMReader: sandbox.NewMemXModel(),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateSandbox(t *testing.T) {
	tmpdir, _ := ioutil.TempDir("", "contract-test")
	defer os.RemoveAll(tmpdir)

	m, err := contract.CreateManager("default", &contract.ManagerConfig{
		Basedir:  tmpdir,
		BCName:   "xuper",
		Core:     new(fakeChainCore),
		XMReader: sandbox.NewMemXModel(),
	})
	if err != nil {
		t.Fatal(err)
	}
	r := sandbox.NewMemXModel()
	state, err := m.NewStateSandbox(&contract.SandboxConfig{
		XMReader: r,
	})
	if err != nil {
		t.Fatal(err)
	}
	state.Put("test", []byte("key"), []byte("value"))
	t.Logf("%v", state.RWSet())
}

func TestInvoke(t *testing.T) {
	tmpdir, _ := ioutil.TempDir("", "contract-test")
	defer os.RemoveAll(tmpdir)

	m, err := contract.CreateManager("default", &contract.ManagerConfig{
		Basedir:  tmpdir,
		BCName:   "xuper",
		Core:     new(fakeChainCore),
		XMReader: sandbox.NewMemXModel(),
	})
	m.GetKernRegistry().RegisterKernMethod("$hello", "Hi", new(helloContract).Hi)

	r := sandbox.NewMemXModel()
	state, err := m.NewStateSandbox(&contract.SandboxConfig{
		XMReader: r,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := m.NewContext(&contract.ContextConfig{
		Module:       "xkernel",
		ContractName: "$hello",
		State:        state,
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ctx.Invoke("Hi", map[string][]byte{
		"name": []byte("xuper"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", resp.Body)
	t.Logf("%v", state.RWSet().WSet[0])
}

type helloContract struct {
}

func (h *helloContract) Hi(ctx contract.KContext) (*contract.Response, error) {
	name := ctx.Args()["name"]
	ctx.Put("test", []byte("k1"), []byte("v1"))
	return &contract.Response{
		Body: []byte("hello " + string(name)),
	}, nil
}
