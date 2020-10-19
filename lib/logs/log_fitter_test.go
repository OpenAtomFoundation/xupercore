package logs

import (
	"sync"
	"testing"
)

func TestInfo(t *testing.T) {
	log, err := GetLogFitter(nil)
	if err != nil {
		t.Errorf("new logger fail.err:%v", err)
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			log.Info("info1", "a", 1, "b", 2, "c", 3, "num", num)
			log.Debug("test", "a", 1, "b", 2, "c", 3, "num", num)
			log.Trace("test", "a", 1, "b", 2, "c", 3, "num", num)
			log.Info("info2", "a", 1, "b", 2, "c", 3, "num", num)
			log.SetInfoField("key1", num)
			log.SetInfoField("key2", num)
			log.Info("info3", "a", true, "b", 1, "num", num)
			log.SetInfoField("key10", num)
		}(i)
	}

	log.Info("info4", "a", 1, "b", 2, "c", 3)
	log.SetInfoField("key3", 3)
	log.SetInfoField("key4", 4)
	log.Info("info5", "a", 1, "b", 2, "c", 3)
	log.Info("info6", "a", 1, "b", 2, "c", 3)
	log.Warn("test warn", 1)
	log.Warn("test warn", 1, 2)

	wg.Wait()
	log.Debug("msg", "log_id", "123456---111111")
	log.Trace("msg")
}
