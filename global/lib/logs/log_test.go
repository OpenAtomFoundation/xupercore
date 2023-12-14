package logs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/OpenAtomFoundation/xupercore/global/lib/logs/config"
)

func BenchmarkLogging(b *testing.B) {
	tmpdir, err := ioutil.TempDir("", "xchain-log")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	conf := config.GetDefLogConf()
	conf.Console = false
	log, err := OpenLog(conf, tmpdir)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	logHandle = log

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l, _ := NewLogger("", "test")
			l.Info("test logging benchmark", "key1", "k1", "key2", "k2")
		}
	})
}
