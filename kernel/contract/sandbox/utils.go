package sandbox

import (
	"bytes"
	"fmt"

	"github.com/xuperchain/xupercore/kernel/ledger"
)

// BucketSeperator separator between bucket and raw key
const BucketSeperator = "/"

// DelFlag delete flag
const DelFlag = "\x00"

func makeRawKey(bucket string, key []byte) []byte {
	k := append([]byte(bucket), []byte(BucketSeperator)...)
	return append(k, key...)
}

func parseRawKey(rawKey []byte) (string, []byte, error) {
	idx := bytes.Index(rawKey, []byte(BucketSeperator))
	if idx < 0 {
		return "", nil, fmt.Errorf("parseRawKey failed, invalid raw key:%s", string(rawKey))
	}
	bucket := string(rawKey[:idx])
	key := rawKey[idx+1:]
	return bucket, key, nil
}

// IsEmptyVersionedData check if VersionedData is empty
func IsEmptyVersionedData(vd *ledger.VersionedData) bool {
	return vd.RefTxid == nil && vd.RefOffset == 0
}

func IsDelFlag(value []byte) bool {
	return bytes.Equal([]byte(DelFlag), value)
}

// helper for test
func putVersionedData(state *MemXModel, bucket string, key []byte, value []byte) {
	state.Put(bucket, key, &ledger.VersionedData{
		RefTxid: []byte("txid"),
		PureData: &ledger.PureData{
			Bucket: bucket,
			Key:    key,
			Value:  value,
		},
	})
}
