package context

import (
	"testing"

	"github.com/xuperchain/xupercore/kernel/mock"
)

func TestNewNetCtx(t *testing.T) {
	mock.InitLogForTest()

	ecfg, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}

	octx, err := NewNetCtx(ecfg)
	if err != nil {
		t.Fatal(err)
	}
	octx.XLog.Debug("test NewNetCtx succ", "cfg", octx.P2PConf)
}
