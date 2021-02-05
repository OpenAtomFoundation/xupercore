package sandbox

import (
	"bytes"
	"errors"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"
)

type MemXModel struct {
	tree *redblacktree.Tree
}

func XMReaderFromRWSet(rwset *contract.RWSet) ledger.XMReader {
	m := NewMemXModel()
	for _, r := range rwset.RSet {
		m.Put(r.PureData.Bucket, r.PureData.Key, r)
	}
	return m
}

func NewMemXModel() *MemXModel {
	tree := redblacktree.NewWith(treeCompare)
	return &MemXModel{
		tree: tree,
	}
}

//读取一个key的值，返回的value就是有版本的data
func (m *MemXModel) Get(bucket string, key []byte) (*ledger.VersionedData, error) {
	buKey := makeRawKey(bucket, key)
	v, ok := m.tree.Get(buKey)
	if !ok {
		return nil, ErrNotFound
	}
	return v.(*ledger.VersionedData), nil
}

func (m *MemXModel) Put(bucket string, key []byte, value *ledger.VersionedData) error {
	buKey := makeRawKey(bucket, key)
	m.tree.Put(buKey, value)
	return nil
}

//扫描一个bucket中所有的kv, 调用者可以设置key区间[startKey, endKey)
func (m *MemXModel) Select(bucket string, startKey []byte, endKey []byte) (ledger.XMIterator, error) {
	if compareBytes(startKey, endKey) >= 0 {
		return nil, errors.New("bad select range")
	}
	rawStartKey := makeRawKey(bucket, startKey)
	rawEndKey := makeRawKey(bucket, endKey)
	return newTreeIterator(m.tree, rawStartKey, rawEndKey), nil
}

func (m *MemXModel) NewIterator(startKey []byte, endKey []byte) (ledger.XMIterator, error) {
	if startKey != nil && compareBytes(startKey, endKey) >= 0 {
		return nil, errors.New("bad iterator range")
	}
	return newTreeIterator(m.tree, startKey, endKey), nil
}

// treeIterator 把tree的Iterator转换成XMIterator
type treeIterator struct {
	tree *redblacktree.Tree
	iter *redblacktree.Iterator
	end  []byte
	key  []byte
	err  error
}

func newTreeIterator(tree *redblacktree.Tree, start, end []byte) ledger.XMIterator {
	var iter redblacktree.Iterator
	if start == nil {
		iter = tree.Iterator()
	} else {
		startNode, ok := tree.Floor(start)
		if !ok {
			return new(treeIterator)
		}
		iter = tree.IteratorAt(startNode)
	}

	return &treeIterator{
		tree: tree,
		iter: &iter,
		end:  end,
	}
}

func (t *treeIterator) Next() bool {
	if t.iter == nil {
		return false
	}
	if t.err != nil {
		return false
	}

	if !t.iter.Next() {
		return false
	}
	rawkey := t.iter.Key()

	if t.end != nil && t.tree.Comparator(rawkey, t.end) >= 0 {
		return false
	}
	_, key, err := parseRawKey(rawkey.([]byte))
	if err != nil {
		t.err = err
		return false
	}
	t.key = key
	return true
}

func (t *treeIterator) Key() []byte {
	if t.iter == nil {
		return nil
	}
	return t.key
}

func (t *treeIterator) Value() *ledger.VersionedData {
	if t.iter == nil {
		return nil
	}
	return t.iter.Value().(*ledger.VersionedData)
}

func (t *treeIterator) Error() error {
	return t.err
}

func (t *treeIterator) Close() {
	t.iter = nil
}

func treeCompare(a, b interface{}) int {
	ka := a.([]byte)
	kb := b.([]byte)
	return bytes.Compare(ka, kb)
}
