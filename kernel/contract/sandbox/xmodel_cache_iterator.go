// Package xmodel CacheIterator is a merged iterator model cache
package sandbox

import (
	"bytes"

	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/xuperchain/xupercore/kernel/contract"
)

type dir int

const (
	// 迭代器被释放
	dirReleased dir = iota - 1
	// start of iterator
	dirSOI
	// end of iterator
	dirEOI
	// 正向迭代
	dirForward
)

type setType string

const (
	setTypeNext  = "Next"
	setTypeFirst = "First"
)

// XMCacheIterator 返回XModelCache的迭代器, 需要对inputsCache、outputsCache和model中的iter进行merge
// 当XMCache可以穿透时需要进行3路merge，当XModelCache不可以穿透时需要进行2路merge
// 当3路迭代时从model中取出的key需要存入inputCache
type XMCacheIterator struct {
	mIter     contract.XMIterator // index: 2
	iters     []iterator.Iterator // index:0 是mcOutIter; index:1 是mcInIter
	cmp       comparer.Comparer
	keys      [][]byte
	markedKey map[string]bool
	index     int
	dir       dir
	err       error
	mc        *XMCache
}

// NewXModelCacheIterator new an instance of XModel Cache iterator
func (mc *XMCache) NewXModelCacheIterator(bucket string, startKey []byte, endKey []byte, cmp comparer.Comparer) (*XMCacheIterator, error) {
	rawStartKey := makeRawKey(bucket, startKey)
	rawEndKey := makeRawKey(bucket, endKey)
	var iters []iterator.Iterator
	mcoi := mc.outputsCache.NewIterator(&util.Range{Start: rawStartKey, Limit: rawEndKey})
	iters = append(iters, mcoi)
	mcii := mc.inputsCache.NewIterator(&util.Range{Start: rawStartKey, Limit: rawEndKey})
	iters = append(iters, mcii)
	var mi contract.XMIterator
	if mc.isPenetrate {
		var err error
		mi, err = mc.model.Select(bucket, startKey, endKey)
		if err != nil {
			return nil, err
		}
	}
	return &XMCacheIterator{
		mIter:     mi,
		mc:        mc,
		iters:     iters,
		cmp:       cmp,
		keys:      make([][]byte, 3),
		markedKey: make(map[string]bool),
	}, nil
}

// Data get data pointer to VersionedData for XMCacheIterator
func (mci *XMCacheIterator) Value() []byte {
	if mci.err != nil || mci.dir == dirReleased {
		return nil
	}
	// TODO:
	switch mci.index {
	case 2:
		return mci.mIter.Value().PureData.Value
	case 0, 1:
		return mci.data(mci.iters[mci.index]).PureData.Value
	default:
		return nil
	}
}

func (mci *XMCacheIterator) data(iter iterator.Iterator) *contract.VersionedData {
	val := iter.Value()
	return mci.mc.getRawData(val)
}

// Next get next XMCacheIterator
func (mci *XMCacheIterator) Next() bool {
	if mci.dir == dirEOI || mci.err != nil {
		return false
	} else if mci.dir == dirReleased {
		mci.err = iterator.ErrIterReleased
		return false
	}

	switch mci.dir {
	case dirSOI:
		return mci.First()
	}

	if !mci.setMciKeys(mci.index, setTypeNext) {
		return false
	}
	return mci.next()
}

func (mci *XMCacheIterator) next() bool {
	var key []byte
	if mci.dir == dirForward {
		key = mci.keys[mci.index]
	}
	for x, tkey := range mci.keys {
		if tkey != nil && (key == nil || mci.cmp.Compare(tkey, key) < 0) {
			key = tkey
			mci.index = x
		}
	}
	if key == nil {
		mci.dir = dirEOI
		return false
	}

	if mci.markedKey[string(key)] {
		return mci.Next()
	}
	mci.markedKey[string(key)] = true
	mci.dir = dirForward
	return true
}

// First get the first XMCacheIterator
func (mci *XMCacheIterator) First() bool {
	if mci.err != nil {
		return false
	} else if mci.dir == dirReleased {
		mci.err = iterator.ErrIterReleased
		return false
	}
	if mci.setMciKeys(0, setTypeFirst) && mci.setMciKeys(1, setTypeFirst) && mci.setMciKeys(2, setTypeFirst) {
		mci.dir = dirSOI
		return mci.next()
	}
	return false
}

// Key get key for XMCacheIterator
func (mci *XMCacheIterator) Key() []byte {
	if mci.err != nil || mci.dir == dirReleased {
		return nil
	}
	switch mci.index {
	case 0, 1:
		return mci.iters[mci.index].Key()
	case 2:
		if mci.mc.isPenetrate {
			return mci.mIter.Key()
		}
		return nil
	}
	return nil
}

func (mci *XMCacheIterator) Error() error {
	return mci.err
}

// Release release the XMCacheIterator
func (mci *XMCacheIterator) Close() {
	if mci.dir == dirReleased {
		return
	}
	mci.dir = dirReleased
	if mci.mIter != nil {
		mci.mIter.Close()
	}
	for _, it := range mci.iters {
		it.Release()
	}
	mci.keys = nil
	mci.iters = nil
}

func (mci *XMCacheIterator) setMciKeys(index int, st setType) bool {
	switch index {
	case 0, 1:
		return mci.setMciCiKey(index, st)
	case 2:
		return mci.setMciMiKey(st)
	default:
		return false
	}
}

func (mci *XMCacheIterator) setMciCiKey(index int, st setType) bool {
	mci.keys[index] = nil
	if st == setTypeFirst {
		isFirst := mci.iters[index].First()
		if isFirst {
			for {
				if mci.iters[index].Error() != nil {
					mci.err = mci.iters[index].Error()
					return false
				}
				key := mci.iters[index].Key()
				if mci.mc.isDel(key) {
					if mci.iters[index].Next() {
						continue
					}
					return true
				}
				mci.keys[index] = key
				break
			}
			return true
		}
	} else if st == setTypeNext {
		isNext := mci.iters[index].Next()
		if isNext {
			for {
				if mci.iters[index].Error() != nil {
					mci.err = mci.iters[index].Error()
					return false
				}
				key := mci.iters[index].Key()
				if mci.mc.isDel(key) {
					if mci.iters[index].Next() {
						continue
					}
					return true
				}
				mci.keys[index] = key
				break
			}
			return true
		}
	}
	return true
}

func (mci *XMCacheIterator) setMciMiKey(st setType) bool {
	mci.keys[2] = nil
	if !mci.mc.isPenetrate {
		return true
	}
	if st == setTypeFirst {
		// TODO:
		var isFirst bool
		// isFirst := mci.mIter.First()
		if isFirst {
			for {
				if mci.mIter.Error() != nil {
					mci.err = mci.mIter.Error()
					return false
				}
				key := mci.mIter.Key()
				if mci.mc.isDel(key) {
					if mci.mIter.Next() {
						continue
					}
					return true
				}
				mci.keys[2] = key
				break
			}
			err := mci.mc.setInputCache(mci.keys[2])
			if err != nil {
				return false
			}
			return true
		}
	} else if st == setTypeNext {
		isNext := mci.mIter.Next()
		if isNext {
			for {
				if mci.mIter.Error() != nil {
					mci.err = mci.mIter.Error()
					return false
				}
				key := mci.mIter.Key()
				if mci.mc.isDel(key) {
					if mci.mIter.Next() {
						continue
					}
					return true
				}
				mci.keys[2] = key
				break
			}
			err := mci.mc.setInputCache(mci.keys[2])
			if err != nil {
				return false
			}
			return true
		}
	}
	return true
}

type multiIterator struct {
	front contract.XMIterator
	back  contract.XMIterator

	frontEnd bool
	backEnd  bool

	key   []byte
	value *contract.VersionedData
}

func newMultiIterator(front, back contract.XMIterator) contract.XMIterator {
	return nil
}

func (m *multiIterator) Key() []byte {
	k1, k2 := m.front.Key(), m.back.Key()
	ret := bytes.Compare(k1, k2)
	switch ret {
	case 0, -1:
		return k1
	case 1:
		return k2
	default:
		return nil
	}
}

func (m *multiIterator) Value() *contract.VersionedData {
	panic("not implemented") // TODO: Implement
}

func (m *multiIterator) Next() bool {
	if m.frontEnd && m.backEnd {
		return false
	}
	return true
}

func (m *multiIterator) Error() error {
	panic("not implemented") // TODO: Implement
}

// Iterator 必须在使用完毕后关闭
func (m *multiIterator) Close() {
	panic("not implemented") // TODO: Implement
}

// compareBytes like bytes.Compare but treats nil as max value
func compareBytes(k1, k2 []byte) int {
	if k1 == nil && k2 == nil {
		return 0
	}
	if k1 == nil {
		return 1
	}
	if k2 == nil {
		return -1
	}
	return bytes.Compare(k1, k2)
}

type memIterator struct {
	xmc  *XMCache
	iter iterator.Iterator
}

func (m *memIterator) Key() []byte {
	return m.iter.Key()
}

func (m *memIterator) Value() *contract.VersionedData {
	return m.xmc.getRawData(m.iter.Value())
}

func (m *memIterator) Next() bool {
	return m.iter.Next()
}

func (m *memIterator) Error() error {
	return m.Error()
}

// Iterator 必须在使用完毕后关闭
func (m *memIterator) Close() {
	m.iter.Release()
}
