package xmodel

import (
	"testing"

	kledger "github.com/xuperchain/xupercore/kernel/ledger"
)

func TestEqual(t *testing.T) {
	testCases := map[string]struct {
		pd     []*kledger.PureData
		vpd    []*kledger.PureData
		expect bool
	}{
		"testEqual": {
			expect: true,
			pd: []*kledger.PureData{
				{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
			vpd: []*kledger.PureData{
				{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
		},
		"testNotEqual": {
			expect: false,
			pd: []*kledger.PureData{
				{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
			vpd: []*kledger.PureData{
				{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				{
					Bucket: "bucket3",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
		},
		"testNotEqual2": {
			expect: false,
			pd: []*kledger.PureData{
				{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
			vpd: []*kledger.PureData{
				{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value3"),
				},
			},
		},
	}

	for k, v := range testCases {
		res := Equal(v.pd, v.vpd)
		t.Log(res)
		if res != v.expect {
			t.Error(k, "error", "expect", v.expect, "actual", res)
		}
	}
}
