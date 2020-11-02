package xuperos

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
)

type NetEvent struct {
	engine def.Engine
}

func NewNetEvent(engine def.Engine) (*NetEvent, error) {
	if engine == nil {
		return nil, fmt.Errorf("new net event failed because param error")
	}

	obj := &NetEvent{
		engine: engine,
	}

	// 注册监听事件
	err := obj.Subscriber()
	if err != nil {
		return nil, fmt.Errorf("new net event failed because register subscriber error.err:%v", err)
	}

	return obj, nil
}

func (t *NetEvent) Start() {

}

func (t *NetEvent) Stop() {

}

func (t *NetEvent) Subscriber() error {
	return fmt.Errorf("the interface is not implemented")
}
