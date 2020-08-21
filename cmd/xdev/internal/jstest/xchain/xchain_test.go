package xchain

import (
	"testing"

	"github.com/xuperchain/xupercore/cmd/xdev/internal/jstest"
)

func TestRunner(t *testing.T) {
	runner, err := jstest.NewRunner(&jstest.RunOption{
		InGoTest: true,
	}, NewAdapter())
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()

	err = runner.RunFile("./testdata/features.test.js")
	if err != nil {
		t.Fatal(err)
	}
}
