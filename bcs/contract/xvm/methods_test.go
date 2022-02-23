package xvm

import (
	"os"
	"testing"

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
			_, exist := symbols["_export_"+symbol]
			if exist != shouldExist {
				t.Errorf("symbol %s not match,want %v,got %v\n", symbol, shouldExist, exist)
			}
		}
	})

}
