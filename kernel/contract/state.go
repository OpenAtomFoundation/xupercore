package contract

// Iterator iterates over key/value pairs in key order
type Iterator interface {
	Key() []byte
	Value() []byte
	Next() bool
	Error() error
	// Iterator 必须在使用完毕后关闭
	Close()
}

type XMReader interface {
	//读取一个key的值，返回的value就是有版本的data
	Get(bucket string, key []byte) ([]byte, error)
	//扫描一个bucket中所有的kv, 调用者可以设置key区间[startKey, endKey)
	Select(bucket string, startKey []byte, endKey []byte) (Iterator, error)
}

type XMWriter interface {
	Put(bucket string, key, value []byte) error
	Del(bucket string, key []byte) error
}

type XModel interface {
	XMReader
	XMWriter
}

type XModelSandbox interface {
	XModel
	RWSet()
}

type XMState interface {
	XModel
}
