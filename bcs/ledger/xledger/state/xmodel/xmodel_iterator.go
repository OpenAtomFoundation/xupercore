package xmodel

import (
	kledger "github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

// XMIterator data structure for XModel Iterator
type XMIterator struct {
	bucket string
	iter   kvdb.Iterator
	model  *XModel
	value  *kledger.VersionedData
	err    error
}

// Data get data pointer to VersionedData for XMIterator
func (di *XMIterator) Value() *kledger.VersionedData {
	return di.value
}

// Next check if next element exist
func (di *XMIterator) Next() bool {
	ok := di.iter.Next()
	if !ok {
		return false
	}
	version := di.iter.Value()
	verData, err := di.model.fetchVersionedData(di.bucket, string(version))
	if err != nil {
		di.err = err
		return false
	}
	di.value = verData
	return true
}

// Key get key for XMIterator
func (di *XMIterator) Key() []byte {
	v := di.Value()
	if v == nil {
		return nil
	}
	return v.GetPureData().GetKey()
}

// Error return error info for XMIterator
func (di *XMIterator) Error() error {
	kverr := di.iter.Error()
	if kverr != nil {
		return kverr
	}
	return di.err
}

// Release release XMIterator
func (di *XMIterator) Close() {
	di.iter.Release()
	di.value = nil
}
