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
		m.Put("test", []byte(keys[i]), nil)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	iter, err := m.NewIterator(nil, nil)
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
}
