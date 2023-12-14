package logs

import (
	"bufio"
	"sync"
	"time"

	log "github.com/xuperchain/log15"
)

func mustBufferFileHandler(path string, fmtr log.Format, interval int, backupCount int) log.Handler {
	h, err := bufferFileHandler(path, fmtr, interval, backupCount)
	if err != nil {
		panic(err)
	}
	return h
}

type syncWriter struct {
	mutex sync.Mutex
	w     *bufio.Writer
}

func newSyncWriter(w *bufio.Writer) *syncWriter {
	return &syncWriter{
		w: w,
	}
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mutex.Lock()
	n, err := s.w.Write(p)
	s.mutex.Unlock()
	return n, err
}

func (s *syncWriter) Flush() {
	s.mutex.Lock()
	s.w.Flush()
	s.mutex.Unlock()
}

func bufferFileHandler(path string, fmtr log.Format, interval int, backupCount int) (log.Handler, error) {
	f, err := log.NewTimeRotateWriter(path, interval, backupCount)
	if err != nil {
		return nil, err
	}
	w := newSyncWriter(bufio.NewWriter(f))

	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			w.Flush()
		}
	}()

	h := log.FuncHandler(func(r *log.Record) error {
		buf := fmtr.Format(r)
		_, err = w.Write(buf)
		return err
	})
	return h, nil
}
