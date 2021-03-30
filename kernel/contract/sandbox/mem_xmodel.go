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
	return newTreeRangeIterator(m.tree, rawStartKey, rawEndKey), nil
}

func (m *MemXModel) NewIterator() ledger.XMIterator {
	return newTreeIterator(m.tree)
}

// treeIterator 把tree的Iterator转换成XMIterator
type treeIterator struct {
	tree *redblacktree.Tree
	iter *redblacktree.Iterator
	end  []byte
	err  error

	iterDone bool
}

func newTreeIterator(tree *redblacktree.Tree) ledger.XMIterator {
	iter := tree.Iterator()
	return &treeIterator{
		tree: tree,
		iter: &iter,
	}
}

func newTreeRangeIterator(tree *redblacktree.Tree, start, end []byte) ledger.XMIterator {
	var iter redblacktree.Iterator
	if start == nil {
		iter = tree.Iterator()
	} else {
		startNode, ok := tree.Floor(start)
		if !ok {
			iter = tree.Iterator()
		} else {
			iter = tree.IteratorAt(startNode)
		}
	}

	return &treeIterator{
		tree: tree,
		iter: &iter,
		end:  end,
	}
}

func (t *treeIterator) Next() bool {
	if t.iterDone {
		return false
	}

	if t.iter == nil {
		t.iterDone = true
		return false
	}
	if t.err != nil {
		t.iterDone = true
		return false
	}

	if !t.iter.Next() {
		t.iterDone = true
		return false
	}
	rawkey := t.iter.Key()

	if t.end != nil && t.tree.Comparator(rawkey, t.end) >= 0 {
		t.iterDone = true
		return false
	}
	return true
}

// 如果迭代器结束则Key返回nil
func (t *treeIterator) Key() []byte {
	if t.iterDone {
		return nil
	}
	return t.Value().GetPureData().GetKey()
}

// 如果迭代器结束则Value返回nil
func (t *treeIterator) Value() *ledger.VersionedData {
	if t.iterDone {
		return nil
	}
	return t.iter.Value().(*ledger.VersionedData)
}

func (t *treeIterator) Error() error {
	return t.err
}

func (t *treeIterator) Close() {
	t.iterDone = true
}

func treeCompare(a, b interface{}) int {
	ka := a.([]byte)
	kb := b.([]byte)
	return bytes.Compare(ka, kb)
}
