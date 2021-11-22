package xvm

import (
	"os"
	"testing"
)

func TestSymbol(t *testing.T) {
	f, err := os.Open("/Users/chenfengjin/baidu/xuperchain/output/data/blockchain/xuper/xvm/counter/code.so")
	if err != nil {
		t.Error(err)
	}
	if err := Symbols(f); err != nil {
		t.Error(err)
	}

}
