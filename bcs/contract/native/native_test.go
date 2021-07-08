package native

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	_ "github.com/xuperchain/xupercore/kernel/contract/manager"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
)

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
}

func compile(th *mock.TestHelper) ([]byte, error) {
	target := filepath.Join(th.Basedir(), "counter.bin")
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

func TestNativeDeploy(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()

	bin, err := compile(th)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := th.Deploy("native", "go", "counter", bin, map[string][]byte{
		"creator": []byte("icexin"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", resp)
}

func TestNativeInvoke(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()

	bin, err := compile(th)
	if err != nil {
		t.Fatal(err)
	}

	_, err = th.Deploy("native", "go", "counter", bin, map[string][]byte{
		"creator": []byte("icexin"),
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := th.Invoke("native", "counter", "increase", map[string][]byte{
		"key": []byte("k1"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("body:%s", resp.Body)
}

func TestNativeUpgrade(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()

	bin, err := compile(th)
	if err != nil {
		t.Fatal(err)
	}

	_, err = th.Deploy("native", "go", "counter", bin, map[string][]byte{
		"creator": []byte("icexin"),
	})
	if err != nil {
		t.Fatal(err)
	}

	err = th.Upgrade("counter", bin)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNativeDocker(t *testing.T) {
	const imageName = "centos:7.5.1804"
	_, err := exec.Command("docker", "inspect", imageName).CombinedOutput()
	if err != nil {
		t.Skip("docker available")
		return
	}
	cfg := *contractConfig
	cfg.Native.Docker = contract.NativeDockerConfig{
		Enable:    true,
		ImageName: imageName,
		Cpus:      1,
		Memory:    "1G",
	}

	th := mock.NewTestHelper(&cfg)
	defer th.Close()

	bin, err := compile(th)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := th.Deploy("native", "go", "counter", bin, map[string][]byte{
		"creator": []byte("icexin"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", resp)
}
