package xvm

import (
	"fmt"
	"os"
	"testing"

	"github.com/xuperchain/xvm/compile"
)

func TestSymbol(t *testing.T) {
	t.Run("AOT", func(t *testing.T) {
		cases := map[string]bool{
			"current": true,
			"legacy":  true,
		}
		var err error

		for testCase, want := range cases {
			outpath := fmt.Sprintf("testdata/counter_%s.so", testCase)
			defer os.RemoveAll(outpath)

			cfg := &compile.Config{
				Wasm2cPath: "wasm2c",
			}
			err = compile.CompileNativeLibrary(cfg, outpath, "testdata/counter_current.wasm")
			if err != nil {
				t.Error(err)
				return
			}
			got, err := isLegacyAOT(outpath)
			if err != nil {
				t.Error(err)
				return
			}
			if got != want {
				t.Errorf("file %s not match,want %v,got %v\n", testCase, want, got)
			}
		}

	})
	t.Run("Interp", func(t *testing.T) {
		cases := map[string]bool{
			"current": false,
			"legacy":  true,
		}
		for testCase, want := range cases {
			codebuf, err := os.ReadFile(fmt.Sprintf("testdata/counter_%s.wasm", testCase))
			if err != nil {
				t.Error(err)
				return
			}
			got, err := isLegacyInterp(codebuf)
			if err != nil {
				t.Error(err)
				return
			}
			if got != want {
				t.Errorf("file %s not match,want %v,got %v\n", testCase, want, got)
			}

		}

	})
}
