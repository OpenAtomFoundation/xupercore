package kernel

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)

type kcontextImpl struct {
	ctx *bridge.Context
	contract.StateSandbox
	used, limit contract.Limits
}

func newKContext(ctx *bridge.Context) *kcontextImpl {
	return &kcontextImpl{
		ctx:          ctx,
		limit:        ctx.ResourceLimits,
		StateSandbox: ctx.State,
	}
}

// 交易相关数据
func (k *kcontextImpl) Args() map[string][]byte {
	return k.ctx.Args
}

func (k *kcontextImpl) Initiator() string {
	return k.ctx.Initiator
}

func (k *kcontextImpl) AuthRequire() []string {
	return k.ctx.AuthRequire
}

func (k *kcontextImpl) AddResourceUsed(delta contract.Limits) {
	k.used.Add(delta)
}

func (k *kcontextImpl) ResourceLimit() contract.Limits {
	return k.limit
}
