package vm

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
)

type kcontextImpl struct {
	ctx         *bridge.Context
	used, limit contract.Limits
}

func newKContext(ctx *bridge.Context) *kcontextImpl {
	return &kcontextImpl{
		ctx: ctx,
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

// 状态修改接口
func (k *kcontextImpl) PutObject(bucket string, key []byte, value []byte) error {
	return k.ctx.Cache.Put(bucket, key, value)
}

func (k *kcontextImpl) GetObject(bucket string, key []byte) ([]byte, error) {
	return k.ctx.Cache.Get(bucket, key)
}

func (k *kcontextImpl) DeleteObject(bucket string, key []byte) error {
	return k.ctx.Cache.Delete(bucket, key)
}

func (k *kcontextImpl) NewIterator(bucket string, start []byte, limit []byte) (kernel.Iterator, error) {
	return k.ctx.Cache.Select(bucket, start, limit)
}

func (k *kcontextImpl) AddResourceUsed(delta contract.Limits) {
	k.used.Add(delta)
}

func (k *kcontextImpl) ResourceLimit() contract.Limits {
	return k.limit
}
