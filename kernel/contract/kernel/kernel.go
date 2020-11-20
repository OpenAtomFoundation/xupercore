package kernel

import "github.com/xuperchain/xupercore/kernel/contract"

type KContext interface {
	// 交易相关数据
	Args() map[string][]byte
	Initiator() string
	AuthRequire() []string

	// 状态修改接口
	contract.XMState

	AddResourceUsed(delta contract.Limits)
	ResourceLimit() contract.Limits
}
