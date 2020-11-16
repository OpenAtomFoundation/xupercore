package leveldb

import (
	pt "path"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/xuperchain/xupercore/lib/storage/config"
	"github.com/xuperchain/xupercore/lib/storage/s3"
)

// Open opens an instance of LDB with parameters (ldb path and other options)
func (ldb *LDBDatabase) OpenCloud(path string, options map[string]interface{}) error {
	setDefaultOptions(options)
	cache := options["cache"].(int)
	fds := options["fds"].(int)
	cfg := config.NewCloudStorageConfig()
	//cloud storage
	s3opt := levels3.OpenOption{
		Bucket:        cfg.Bucket,
		Path:          pt.Join(cfg.Path, path),
		Ak:            cfg.Ak,
		Sk:            cfg.Sk,
		Region:        cfg.Region,
		Endpoint:      cfg.Endpoint,
		LocalCacheDir: cfg.LocalCacheDir,
	}
	st, err := levels3.NewS3Storage(s3opt)
	if err != nil {
		return err
	}
	db, err := leveldb.Open(st, &opt.Options{
		OpenFilesCacheCapacity: fds,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		//db, err = leveldb.Recover(store, nil)
		return err
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return err
	}
	ldb.fn = path
	ldb.db = db
	return nil
}
