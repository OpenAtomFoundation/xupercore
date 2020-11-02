package xuperos

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
)

// 代理依赖组件实例化操作，方便mock单测和并行开发
type RelyAgentMock struct {
	engine def.Engine
}

func MockRelyAgent(engine def.Engine) *RelyAgentMock {
	return &RelyAgentMock{engine}
}

func (t *RelyAgentMock) CreateNetwork() (XNetwork, error) {
	return nil, fmt.Errorf("the interface is not implemented")
}
