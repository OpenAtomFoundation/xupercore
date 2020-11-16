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

// Iterator iterates over key/value pairs in key order
type Iterator interface {
	Key() []byte
	Value() []byte
	Next() bool
	Error() error
	// Iterator 必须在使用完毕后关闭
	Close()
}
