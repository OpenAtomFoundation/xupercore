package leveldb

import (
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"math/rand"
	"testing"
)

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// 产生随机字符串
func RandBytes(n int) []byte {
	b := make([]byte, n)
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return b
}

func makeDB() (kvdb.Database, error) {
	kvParam := &kvdb.KVParameter{
		DBPath:                "./leveldb",
		KVEngineType:          "leveldb",
		StorageType:           "single",
		MemCacheSize:          128,
		FileHandlersCacheSize: 1024,
	}
	return NewKVDBInstance(kvParam)
}

func BenchmarkLdbBatch_Put(b *testing.B) {
	db, err := makeDB()
	if err != nil {
		b.Errorf("NewKVDBInstance error: %s", err)
		return
	}
	defer db.Close()

	key := RandBytes(64)
	value := RandBytes(1024)

	keys := make([][]byte, 5)
	for i := 0; i < b.N; i++ {
		for k := 0; k < 10; k++ {
			db.Get(keys[k%5])
		}

		batch := db.NewBatch()

		if i > 0 {
			batch.Delete(keys[1])
			batch.Delete(keys[3])
		}

		for j := 0; j < 5; j++ {
			key = RandBytes(64)
			value = RandBytes(1024)
			batch.Put(key, value)

			keys[j] = key
		}
		batch.Write()
	}
}

func BenchmarkLdbBatch_ParallelPut(b *testing.B) {
	db, err := makeDB()
	if err != nil {
		b.Errorf("NewKVDBInstance error: %s", err)
		return
	}
	defer db.Close()

	key := RandBytes(64)
	value := RandBytes(1024)

	keys := make([][]byte, 5)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := int(rand.Int63n(10240))
			db.Get(key)
			batch := db.NewBatch()

			if i > 0 {
				batch.Delete(keys[1])
				batch.Delete(keys[3])
			}

			for j := 0; j < 5; j++ {
				key[(i+j)%64] = 'K'
				value[(i+j)%1024] = 'V'
				batch.Put(key, value)

				keys[j] = key
			}
			batch.Write()
		}
	})
}

func BenchmarkLdbBatch_Get(b *testing.B) {
	db, err := makeDB()
	if err != nil {
		b.Errorf("NewKVDBInstance error: %s", err)
		return
	}
	defer db.Close()

	key := RandBytes(64)
	value := RandBytes(1024)
	db.Put(key, value)
	for i := 0; i < b.N; i++ {
		db.Get(key)
	}
}

func BenchmarkLdbBatch_Get_Parallel(b *testing.B) {
	db, err := makeDB()
	if err != nil {
		b.Errorf("NewKVDBInstance error: %s", err)
		return
	}
	defer db.Close()

	key := RandBytes(64)
	value := RandBytes(1024)
	db.Put(key, value)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			db.Get(key)
		}
	})
}

func BenchmarkLdbBatch_GetNotExist(b *testing.B) {
	db, err := makeDB()
	if err != nil {
		b.Errorf("NewKVDBInstance error: %s", err)
		return
	}
	defer db.Close()

	key := RandBytes(64)
	for i := 0; i < b.N; i++ {
		db.Get(key)
	}
}

func BenchmarkLdbBatch_ParallelGetNotExist(b *testing.B) {
	db, err := makeDB()
	if err != nil {
		b.Errorf("NewKVDBInstance error: %s", err)
		return
	}
	defer db.Close()

	key := RandBytes(64)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			db.Get(key)
		}
	})
}
