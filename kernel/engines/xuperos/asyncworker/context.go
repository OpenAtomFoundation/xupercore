package asyncworker

import "encoding/json"

var (
	UnmarshalFunc = json.Unmarshal
)

type TaskContextImpl struct {
	decodeFunc func(data []byte, v interface{}) error
	buf        []byte
}

func newTaskContextImpl(buf []byte) *TaskContextImpl {
	return &TaskContextImpl{
		decodeFunc: UnmarshalFunc,
		buf:        buf,
	}
}
func (tc *TaskContextImpl) ParseArgs(v interface{}) error {
	return tc.decodeFunc(tc.buf, v)
}

func (tc *TaskContextImpl) RetryTimes() int {
	return 0
}
