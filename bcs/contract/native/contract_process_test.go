package native

import (
	"os/exec"
	"testing"
	"time"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/protos"
)

func TestCommandNotFound(t *testing.T) {

	t.Run("testDocker", func(t *testing.T) {
		if resp, err := exec.Command("docker", "info").CombinedOutput(); err != nil {
			t.Skip(string(resp))
		}
		pm, err := newContractProcess(&contract.NativeConfig{
			Driver:      "native",
			StopTimeout: 5,
			Enable:      true,
			Docker: contract.NativeDockerConfig{
				Enable:    true,
				ImageName: "ubuntu:18.04:",
				Cpus:      1,
				Memory:    "1G",
			},
		}, "xchain-test", "/tmp", "", &protos.WasmCodeDesc{
			Runtime: "java",
			Digest:  []byte("nativetest"),
		})

		if err != nil {
			t.Error(err)
		}

		process, err := pm.makeNativeProcess()

		err = process.Start()
		defer process.Stop(time.Second)
		if err == nil {
			t.Error("expect error,get nil")
		}
		t.Log(err)
	})

	t.Run("testDockerOpenJDK", func(t *testing.T) {
		if resp, err := exec.Command("docker", "info").CombinedOutput(); err != nil {
			t.Skip(string(resp))
		}
		if resp, err := exec.Command("docker", []string{"image", "inspect", "openjdk:8u292-slim-buster"}...).CombinedOutput(); err != nil {
			t.Skip((string(resp)))
		}
		cp, err := newContractProcess(&contract.NativeConfig{
			Driver:      "native",
			StopTimeout: 5,
			Enable:      true,
			Docker: contract.NativeDockerConfig{
				Enable:    true,
				ImageName: "openjdk:8u292-slim-buster",
				Cpus:      1,
				Memory:    "1G",
			},
		}, "xchain-test", "/tmp", "", &protos.WasmCodeDesc{
			Runtime: "java",
			Digest:  []byte("nativetest"),
		})
		process, err := cp.makeNativeProcess()
		if err != nil {
			t.Error(err)
		}

		defer process.Stop(time.Second)

		if err != nil {
			t.Error(err)
		}

		err = process.Start()
		if err != nil {
			t.Error(err)
		}
		//t.Log(err)
	})
}
