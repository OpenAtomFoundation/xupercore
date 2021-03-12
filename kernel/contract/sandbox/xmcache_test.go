package sandbox

import (
	"math/big"
	"math/rand"
	"sort"
	"testing"

	"github.com/xuperchain/xupercore/kernel/ledger"
)

func TestXMCachePutGet(t *testing.T) {
	testCases := []struct {
		Bucket string
		Key    string
		Value  string
		Op     string
	}{
		{"b1", "k1", "v1", "put"},
		{"b1", "k1", "v1", "get"},
		{"b1", "k1", "v2", "put"},
		{"b1", "k1", "v2", "get"},
	}
	store := NewMemXModel()

	mc := NewXModelCache(store)
	for _, test := range testCases {
		switch test.Op {
		case "put":
			err := mc.Put(test.Bucket, []byte(test.Key), []byte(test.Value))
			if err != nil {
				t.Fatal(err)
			}
		case "get":
			v, err := mc.Get(test.Bucket, []byte(test.Key))
			if err != nil {
				t.Fatal(err)
			}
			if string(v) != test.Value {
				t.Errorf("expect %s got %s", test.Value, v)
			}
		}
	}
}

func TestXMCacheIterator(t *testing.T) {
	const N = 10
	const prefix = "key_"
	keys := make([]string, N)
	rnd := rand.New(rand.NewSource(0))
	for i := 0; i < N; i++ {
		key := make([]byte, 10)
		rnd.Read(key)
		keys[i] = prefix + big.NewInt(0).SetBytes(key).Text(35)
	}

	state := NewMemXModel()
	for i := 0; i < N/2; i++ {
		t.Logf("write state:%s", keys[i])
		state.Put("test", []byte(keys[i]), &ledger.VersionedData{
			RefTxid: []byte("txid"),
			PureData: &ledger.PureData{
				Bucket: "test",
				Key:    []byte(keys[i]),
				Value:  []byte(keys[i]),
			},
		})
	}
	mc := NewXModelCache(state)
	for i := N / 2; i < N; i++ {
		t.Logf("write cache:%s", keys[i])
		mc.Put("test", []byte(keys[i]), []byte(keys[i]))
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	iter, err := mc.Select("test", []byte(prefix), []byte(prefix+"\xff"))
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

	rwset := mc.RWSet()
	for _, r := range rwset.RSet {
		t.Logf("%s", r.GetPureData().GetKey())
	}
}
