// badger wrapper plugin
// so
package badgerdb

import (
	"fmt"
	"os"

	log "github.com/xuperchain/log15"

	"github.com/OpenAtomFoundation/xupercore/global/lib/storage/kvdb"
	"github.com/dgraph-io/badger/v3"
)

// BadgerDatabase define db backend based on badger
type BadgerDatabase struct {
	fn  string     // filename of db
	db  *badger.DB // db instance
	log log.Logger // logger instance
}

func NewKVDBInstance(param *kvdb.KVParameter) (kvdb.Database, error) {
	var baseDB kvdb.Database = new(BadgerDatabase)
	err := baseDB.Open(param.GetDBPath(), map[string]interface{}{
		"cache":     param.GetMemCacheSize(),
		"fds":       param.GetFileHandlersCacheSize(),
		"dataPaths": param.GetOtherPaths(),
	})
	if err != nil {
		return nil, err
	}

	return baseDB, nil
}

func init() {
	kvdb.Register("badger", NewKVDBInstance)
}

// Path returns the path to the database directory
func (bdb *BadgerDatabase) Path() string {
	return bdb.fn
}

func (bdb *BadgerDatabase) Open(path string, options map[string]interface{}) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
	}
	logger := log.New("database", path)
	bdb.fn = path
	opts := badger.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path
	opts.SyncWrites = false
	opts.ValueThreshold = 256
	opts.CompactL0OnClose = true
	db, err := badger.Open(opts)
	if err != nil {
		log.Warn("badger open failed", "path", path, "err", err)
		return err
	}
	bdb.db = db
	bdb.log = logger
	return nil
}

func (bdb *BadgerDatabase) Close() {
	err := bdb.db.Close()
	if err == nil {
		bdb.log.Info("database closed")
	} else {
		bdb.log.Error("failed to close database", "err", err)
	}
}

func (bdb *BadgerDatabase) Put(key []byte, value []byte) error {
	wb := bdb.db.NewWriteBatch()
	defer wb.Cancel()
	err := wb.SetEntry(badger.NewEntry(key, value).WithMeta(0))
	if err != nil {
		return err
	}
	return wb.Flush()
}

func (bdb *BadgerDatabase) Delete(key []byte) error {
	wb := bdb.db.NewWriteBatch()
	defer wb.Cancel()
	return wb.Delete(key)
}

func (bdb *BadgerDatabase) Get(key []byte) ([]byte, error) {
	var ival []byte
	err := bdb.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		ival, err = item.ValueCopy(nil)
		return err
	})
	return ival, err
}

func (bdb *BadgerDatabase) Has(key []byte) (bool, error) {
	var exist bool = false
	err := bdb.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err != nil {
			return err
		} else {
			exist = true
		}
		return err
	})
	// align with leveldb, if the key doesn't exist, leveldb returns nil
	if kvdb.ErrNotFound(err) {
		err = nil
	}
	return exist, err
}

func (bdb *BadgerDatabase) NewBatch() kvdb.Batch {
	return &BadgerBatch{db: bdb.db, b: bdb.db.NewWriteBatch(), keys: map[string]bool{}}
}

type BadgerBatch struct {
	db      *badger.DB
	b       *badger.WriteBatch
	size    int
	keys    map[string]bool
	discard bool
}

func (b *BadgerBatch) Put(key, value []byte) error {
	if b.discard {
		b.b = b.db.NewWriteBatch()
		b.discard = false
	}
	err := b.b.SetEntry(badger.NewEntry(key, value).WithMeta(0))
	if err != nil {
		return err
	}
	b.size += len(value)
	return nil
}

func (b *BadgerBatch) Delete(key []byte) error {
	if b.discard {
		b.b = b.db.NewWriteBatch()
		b.discard = false
	}
	err := b.b.Delete(key)
	if err != nil {
		return err
	}
	b.size += len(key)
	return nil
}

func (b *BadgerBatch) PutIfAbsent(key, value []byte) error {
	if b.discard {
		b.b = b.db.NewWriteBatch()
		b.discard = false
	}
	if !b.keys[string(key)] {
		err := b.b.SetEntry(badger.NewEntry(key, value).WithMeta(0))
		if err != nil {
			return err
		}
		b.size += len(value)
		b.keys[string(key)] = true
		return nil
	}
	return fmt.Errorf("duplicated key in batch, (HEX) %x", key)
}

func (b *BadgerBatch) Exist(key []byte) bool {
	return b.keys[string(key)]
}

func (b *BadgerBatch) Write() error {
	defer func() {
		b.b.Cancel()
		b.discard = true
	}()
	return b.b.Flush()
}

func (b *BadgerBatch) ValueSize() int {
	return b.size
}

func (b *BadgerBatch) Reset() {
	b.b = b.db.NewWriteBatch()
	b.keys = make(map[string]bool)
	b.size = 0
}

func (bdb *BadgerDatabase) NewIteratorWithPrefix(prefix []byte) kvdb.Iterator {
	iteratorOptions := badger.IteratorOptions{
		PrefetchValues: true,
		PrefetchSize:   100,
		Reverse:        false,
		AllVersions:    false,
		Prefix:         prefix,
	}
	return NewBadgerIterator(bdb.db, iteratorOptions, true, false, prefix, []byte("00"))
}

func (bdb *BadgerDatabase) NewIteratorWithRange(start []byte, limit []byte) kvdb.Iterator {
	opt := badger.IteratorOptions{
		PrefetchValues: true,
		PrefetchSize:   100,
		Reverse:        false,
		AllVersions:    false,
	}
	return NewBadgerIterator(bdb.db, opt, false, true, start, limit)
}
