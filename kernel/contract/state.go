package contract

import (
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/protos"
)

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

type ContractEventState interface {
	AddEvent(events ...*protos.ContractEvent)
}

// State 抽象了链的状态机接口，合约通过State里面的方法来修改状态。
type State interface {
	XMState
	UTXOState
	CrossQueryState
	ContractEventState
}

// StateSandbox 在沙盒环境里面执行状态修改操作，最终生成读写集
type StateSandbox interface {
	State
	// Flush将缓存的UTXO，CrossQuery等内存状态写入到读写集
	// 没有调用Flush只能得到KV数据的读写集
	Flush() error
	RWSet() *RWSet
}

type RWSet struct {
	RSet []*ledger.VersionedData
	WSet []*ledger.PureData
}
