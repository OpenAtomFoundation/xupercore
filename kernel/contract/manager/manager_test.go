package manager

import (
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
)

func TestCreate(t *testing.T) {
	_, err := contract.CreateManager("default", &contract.ManagerConfig{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateSandbox(t *testing.T) {
	m, err := contract.CreateManager("default", &contract.ManagerConfig{})
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
	m, err := contract.CreateManager("default", &contract.ManagerConfig{})
	if err != nil {
		t.Fatal(err)
	}

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
