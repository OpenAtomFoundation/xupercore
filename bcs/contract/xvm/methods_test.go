package xvm

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/xuperchain/wagon/wasm"
	"github.com/xuperchain/xvm/compile"
)

func TestSymbol(t *testing.T) {
	t.Run("Current", func(t *testing.T) {
		cases := map[string]bool{
			"initialize": true,
			"increase":   true,
		}
		var err error

		outpath := "testdata/counter_current.so"
		defer os.RemoveAll(outpath)

		cfg := &compile.Config{
			Wasm2cPath: "wasm2c",
		}
		err = compile.CompileNativeLibrary(cfg, outpath, "testdata/counter_current.wasm")
		if err != nil {
			t.Error(err)
			return
		}

		symbols, err := resolveSymbols(outpath)
		if err != nil {
			t.Error(err)
			return
		}
		for symbol, shouldExist := range cases {
			_, exist := symbols[exportSymbolPrefix+symbol]
			if exist != shouldExist {
				t.Errorf("symbol %s not match,want %v,got %v\n", symbol, shouldExist, exist)
			}
		}
	})

	t.Run("Legacy", func(t *testing.T) {
		cases := map[string]bool{
			"initialize": false,
			"increase":   false,
		}
		var err error
		outpath := "testdata/counter_legacy.so"
		defer os.RemoveAll(outpath)

		cfg := &compile.Config{
			Wasm2cPath: "wasm2c",
		}
		err = compile.CompileNativeLibrary(cfg, outpath, "testdata/counter_legacy.wasm")
		if err != nil {
			t.Error(err)
			return
		}

		symbols, err := resolveSymbols(outpath)
		if err != nil {
			t.Error(err)
			return
		}
		for symbol, shouldExist := range cases {
			_, exist := symbols[exportSymbolPrefix+symbol]
			if exist != shouldExist {
				t.Errorf("symbol %s not match,want %v,got %v\n", symbol, shouldExist, exist)
			}
		}
	})

	t.Run("Interp", func(t *testing.T) {
		cases := map[string]bool{
			"current": true,
			"legacy":  false,
		}
		for testCase, want := range cases {
			content, err := os.ReadFile(fmt.Sprintf("testdata/counter_%s.wasm", testCase))
			if err != nil {
				t.Error(err)
				return
			}
			current := false

			module, err := wasm.DecodeModule(bytes.NewBuffer(content))
			if err != nil {
				t.Error(err)
				return
			}
			if module.Import != nil {
				for _, entry := range module.Export.Entries {
					if entry.FieldStr == currentContractMethodInitialize {
						current = true
					}
				}
			}

			if current != want {
				t.Errorf("file %s not match,want %v,got %v\n", testCase, want, current)
			}
		}
	})

}
