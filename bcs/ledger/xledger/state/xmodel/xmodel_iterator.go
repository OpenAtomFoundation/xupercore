package xmodel

import (
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	kledger "github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

// XMIterator data structure for XModel Iterator
type XMIterator struct {
	bucket string
	iter   kvdb.Iterator
	model  *XModel
	err    error
}

// Data get data pointer to VersionedData for XMIterator
func (di *XMIterator) Value() *kledger.VersionedData {
	version := di.iter.Value()
	verData, err := di.model.fetchVersionedData(di.bucket, string(version))
	di.err = err
	return verData
}

// Next check if next element exist
func (di *XMIterator) Next() bool {
	return di.iter.Next()
}

// First ...
func (di *XMIterator) First() bool {
	return di.iter.First()
}

// Key get key for XMIterator
func (di *XMIterator) Key() []byte {
	tablePrefixLen := len(pb.ExtUtxoTablePrefix)
	kvdbKey := di.iter.Key()
	return kvdbKey[tablePrefixLen:len(kvdbKey)]
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
}
