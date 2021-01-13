package manager

import (
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
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
