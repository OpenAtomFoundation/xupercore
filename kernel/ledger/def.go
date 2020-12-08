// 账本约束数据结构定义
package ledger

type XMSnapshotReader interface {
	Get(bucket string, key []byte) ([]byte, error)
}
