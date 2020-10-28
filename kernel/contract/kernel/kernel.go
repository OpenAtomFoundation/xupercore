package kernel

import "github.com/xuperchain/xupercore/kernel/contract"

type KContext interface {
	// 交易相关数据
	Args() map[string][]byte
	Initiator() string
	AuthRequire() []string

	// 状态修改接口
	PutObject(bucket string, key []byte, value []byte) error
	GetObject(bucket string, key []byte) ([]byte, error)
	DeleteObject(bucket string, key []byte) error
	NewIterator(bucket string, start, limit []byte) Iterator

	AddResourceUsed(delta contract.Limits)
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
