package xuperos

import (
	"fmt"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/network"
)

// 代理依赖组件实例化操作，方便mock单测和并行开发
type RelyAgentMock struct {
	engine common.Engine
}

func MockRelyAgent(engine common.Engine) *RelyAgentMock {
	return &RelyAgentMock{engine}
}

func (t *RelyAgentMock) CreateNetwork() (network.Network, error) {
	return nil, fmt.Errorf("the interface is not implemented")
}
