package sandbox

import (
	"crypto/rand"
	"math/big"
	"sort"
	"testing"

	"github.com/xuperchain/xupercore/kernel/ledger"
)

func TestXModelIterator(t *testing.T) {
	const N = 10
	keys := make([]string, N)
	for i := 0; i < N; i++ {
		key := make([]byte, 10)
		rand.Read(key)
		keys[i] = big.NewInt(0).SetBytes(key).Text(35)
	}

	m := NewMemXModel()
	for i := 0; i < N; i++ {
		putVersionedData(m, "test", []byte(keys[i]), []byte(keys[i]))
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	iter := m.NewIterator()

	i := 0
	for iter.Next() {
		if compareBytes([]byte(keys[i]), iter.Key()) != 0 {
			t.Fatalf("not equal: %s %s", keys[i], iter.Key())
		}
		i++
	}
	if i != N {
		t.Fatalf("expect iter %d iterms got %d", N, i)
	}
}

func TestXModelRangeIterator(t *testing.T) {
	const N = 10
	const prefix = "key_"
	keys := make([]string, N)
	for i := 0; i < N; i++ {
		key := make([]byte, 10)
		rand.Read(key)
		keys[i] = prefix + big.NewInt(0).SetBytes(key).Text(35)
	}

	m := NewMemXModel()
	for i := 0; i < N; i++ {
		putVersionedData(m, "test", []byte(keys[i]), []byte(keys[i]))
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	iter, err := m.Select("test", []byte(keys[0]), []byte(keys[N-1]))
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for iter.Next() {
		if compareBytes([]byte(keys[i]), iter.Key()) != 0 {
			t.Fatalf("not equal: %s %s", keys[i], iter.Key())
		}
		i++
	}
	if i != N-1 {
		t.Fatalf("expect iter %d iterms got %d", N, i)
	}
}

func expectNextKey(t *testing.T, iter ledger.XMIterator, expect string) {
	ok := iter.Next()
	if !ok {
		t.Fatal("expect next ok, go eof")
	}
	key := iter.Key()
	if string(key) != expect {
		t.Fatalf("expect next key:%s, got %s", expect, key)
	}
}

func TestXModelIteratorStartAndEnd(t *testing.T) {
	m := NewMemXModel()
	putVersionedData(m, "tess", []byte("1"), []byte("1"))
	putVersionedData(m, "test", []byte("1"), []byte("1"))
	putVersionedData(m, "test", []byte("2"), []byte("2"))
	putVersionedData(m, "test", []byte("3"), []byte("2"))
	putVersionedData(m, "test1", []byte("1"), []byte("1"))
	t.Run("nil start", func(tt *testing.T) {
		iter, err := m.Select("test", nil, nil)
		if err != nil {
			tt.Fatal(err)
		}
		expectNextKey(t, iter, "1")
	})
	t.Run("non nil start", func(tt *testing.T) {
		iter, err := m.Select("test", []byte("2"), nil)
		if err != nil {
			tt.Fatal(err)
		}
		expectNextKey(t, iter, "2")
	})
	t.Run("nil end", func(tt *testing.T) {
		iter, err := m.Select("test", []byte("2"), nil)
		if err != nil {
			tt.Fatal(err)
		}
		expectNextKey(tt, iter, "2")
		expectNextKey(tt, iter, "3")
		if iter.Next() {
			tt.Error("expect end")
		}
	})
	t.Run("non nil end", func(tt *testing.T) {
		iter, err := m.Select("test", []byte("2"), []byte("3"))
		if err != nil {
			tt.Fatal(err)
		}
		expectNextKey(tt, iter, "2")
		if iter.Next() {
			tt.Error("expect end")
		}
	})

}
