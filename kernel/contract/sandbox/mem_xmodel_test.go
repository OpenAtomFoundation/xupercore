package sandbox

import (
	"crypto/rand"
	"math/big"
	"sort"
	"testing"
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

	iter, err := m.Select("test", []byte(prefix), []byte(prefix+"\xff"))
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
	if i != N {
		t.Fatalf("expect iter %d iterms got %d", N, i)
	}
}
