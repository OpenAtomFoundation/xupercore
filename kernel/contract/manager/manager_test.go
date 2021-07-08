package manager

import (
	"testing"

	log15 "github.com/xuperchain/log15"
	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
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

var contractConfig = &contract.ContractConfig{
	Xkernel: contract.XkernelConfig{
		Enable: true,
		Driver: "default",
	},
	LogDriver: &MockLogger{},
}

func TestCreate(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
}

func TestCreateSandbox(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()

	r := sandbox.NewMemXModel()
	state, err := m.NewStateSandbox(&contract.SandboxConfig{
		XMReader: r,
	})
	if err != nil {
		t.Fatal(err)
	}
	state.Put("test", []byte("key"), []byte("value"))
	if string(state.RWSet().WSet[0].Value) != "value" {
		t.Error("unexpected value")
	}
}

func TestInvoke(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()

	m.GetKernRegistry().RegisterKernMethod("$hello", "Hi", new(helloContract).Hi)

	resp, err := th.Invoke("xkernel", "$hello", "Hi", map[string][]byte{
		"name": []byte("xuper"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", resp.Body)
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
