package native

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/protos"
	"testing"
)

func TestCommandNotFound(t *testing.T) {
	t.Run("testHost", func(t *testing.T) {
		process, err := newContractProcess(&contract.NativeConfig{
			Driver:      "native",
			StopTimeout: 5,
			Enable:      true,
			Docker: contract.NativeDockerConfig{
				Enable: false,
			},
		}, "nil", "/tmp", "", &protos.WasmCodeDesc{
			Runtime: "java",
			Digest:  []byte("nativetest"),
		})

		if err != nil {
			t.Error(err)
		}

		if err := process.Start(); err == nil {
			t.Error("expect error,get nil")
		}
		t.Log(err)
	})

	t.Run("testDocker", func(t *testing.T) {
		process, err := newContractProcess(&contract.NativeConfig{
			Driver:      "native",
			StopTimeout: 5,
			Enable:      true,
			Docker: contract.NativeDockerConfig{
				Enable:    true,
				ImageName: "docker.io/centos:7.5.1804",
				Cpus:      1,
				Memory:    "1G",
			},
		}, "nil", "/tmp", "", &protos.WasmCodeDesc{
			Runtime: "java",
			Digest:  []byte("nativetest"),
		})

		if err != nil {
			t.Error(err)
		}

		err = process.Start()
		if err == nil {
			t.Error("expect error,get nil")
		}
		t.Log(err)
	})

	t.Run("testDockerOpenJDK", func(t *testing.T) {
		process, err := newContractProcess(&contract.NativeConfig{
			Driver:      "native",
			StopTimeout: 5,
			Enable:      true,
			Docker: contract.NativeDockerConfig{
				Enable:    true,
				ImageName: "openjdk:8u292-slim-buster",
				Cpus:      1,
				Memory:    "1G",
			},
		}, "nil", "/tmp", "", &protos.WasmCodeDesc{
			Runtime: "java",
			Digest:  []byte("nativetest"),
		})

		if err != nil {
			t.Error(err)
		}

		err = process.Start()
		if err == nil {
			t.Error("expect error,get nil")
		}
		t.Log(err)
	})
}
