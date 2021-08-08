package mock

import (
	log15 "github.com/xuperchain/log15"
)

type MockLogger struct {
	log15.Logger
}

func (*MockLogger) GetLogId() string {
	return ""
}

func (*MockLogger) SetCommField(key string, value interface{}) {

}
func (*MockLogger) SetInfoField(key string, value interface{}) {

}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		log15.New(),
	}
}
