package contract

import (
	"github.com/xuperchain/xupercore/kernel/ledger"
)

// // XMIterator iterates over key/value pairs in key order
// type XMIterator interface {
// 	Key() []byte
// 	Value() *VersionedData
// 	Next() bool
// 	Error() error
// 	// Iterator 必须在使用完毕后关闭
// 	Close()
// }

// // XMReader 为账本的XModel的读接口集合，
// // 合约通过XMReader构造StateSandbox，从而生成读写集
// type XMReader interface {
// 	//读取一个key的值，返回的value就是有版本的data
// 	Get(bucket string, key []byte) (*VersionedData, error)
// 	//扫描一个bucket中所有的kv, 调用者可以设置key区间[startKey, endKey)
// 	Select(bucket string, startKey []byte, endKey []byte) (XMIterator, error)
// }

type SandboxConfig struct {
	XMReader ledger.XMReader
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

// XMState 对XuperBridge暴露对XModel的读写接口，不同于XMReader，
// Get和Select方法得到的不是VersionedData，而是[]byte
type XMState interface {
	Get(bucket string, key []byte) ([]byte, error)
	//扫描一个bucket中所有的kv, 调用者可以设置key区间[startKey, endKey)
	Select(bucket string, startKey []byte, endKey []byte) (Iterator, error)
	Put(bucket string, key, value []byte) error
	Del(bucket string, key []byte) error
}

// XMState 对XuperBridge暴露对账本的UTXO操作能力
type UTXOState interface {
}

// CrossQueryState 对XuperBridge暴露对跨链只读合约的操作能力
type CrossQueryState interface {
}

// State 抽象了链的状态机接口，合约通过State里面的方法来修改状态。
type State interface {
	XMState
	UTXOState
	CrossQueryState
}

// StateSandbox 在沙盒环境里面执行状态修改操作，最终生成读写集
type StateSandbox interface {
	State
	RWSet() *RWSet
}

// type PureData struct {
// 	Bucket string
// 	Key    []byte
// 	Value  []byte
// }

// type VersionedData struct {
// 	PureData  *PureData
// 	RefTxid   []byte
// 	RefOffset int32
// }

type RWSet struct {
	RSet []*ledger.VersionedData
	WSet []*ledger.PureData
}
