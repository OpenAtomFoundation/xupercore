package sandbox

import (
	"testing"
)

func TestMultiIterator(t *testing.T) {
	const bucket = "test"
	frontKVs := [][2]string{
		{"k1", "v"},
		{"k3", "v"},
		{"k2", "v"},
	}
	backKVs := [][2]string{
		{"k1", "v1"},
		{"k4", "v"},
		{"k5", "v"},
		{"k3", "v1"},
	}
	frontState := NewMemXModel()
	backState := NewMemXModel()
	for _, kv := range frontKVs {
		putVersionedData(frontState, bucket, []byte(kv[0]), []byte(kv[1]))
	}
	for _, kv := range backKVs {
		putVersionedData(backState, bucket, []byte(kv[0]), []byte(kv[1]))
	}

	iter := newMultiIterator(frontState.NewIterator(), backState.NewIterator())

	expected := [][2]string{
		{"k1", "v"},
		{"k2", "v"},
		{"k3", "v"},
		{"k4", "v"},
		{"k5", "v"},
	}
	i := 0
	for iter.Next() {
		t.Logf("%s", iter.Key())
		if string(iter.Key()) != expected[i][0] {
			t.Fatalf("expect %s got %s", expected[i][0], iter.Key())
		}
		if string(iter.Value().GetPureData().GetValue()) != expected[i][1] {
			t.Fatalf("expect %s got %s", expected[i][1], iter.Value().GetPureData().GetValue())
		}
		i++
	}
	if i != len(expected) {
		t.Fatalf("expect iter %d iterms got %d", len(expected), i)
	}
}
