package xmodel

import (
	"bytes"
	"sort"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	kledger "github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"

	"github.com/golang/protobuf/proto"
)

// KVEngineType KV storage type
const KVEngineType = "default"

// BucketSeperator separator between bucket and raw key
const BucketSeperator = "/"

// DelFlag delete flag
const DelFlag = "\x00"

func isDelFlag(value []byte) bool {
	return bytes.Equal([]byte(DelFlag), value)
}

// MakeRawKey make key with bucket and raw key
func MakeRawKey(bucket string, key []byte) []byte {
	return makeRawKey(bucket, key)
}

func makeRawKey(bucket string, key []byte) []byte {
	k := append([]byte(bucket), []byte(BucketSeperator)...)
	return append(k, key...)
}

func queryUnconfirmTx(txid []byte, table kvdb.Database) (*pb.Transaction, error) {
	pbBuf, findErr := table.Get(txid)
	if findErr != nil {
		return nil, findErr
	}
	tx := &pb.Transaction{}
	pbErr := proto.Unmarshal(pbBuf, tx)
	if pbErr != nil {
		return nil, pbErr
	}
	return tx, nil
}

func saveUnconfirmTx(tx *pb.Transaction, batch kvdb.Batch) error {
	buf, err := proto.Marshal(tx)
	if err != nil {
		return err
	}
	rawKey := append([]byte(pb.UnconfirmedTablePrefix), []byte(tx.Txid)...)
	batch.Put(rawKey, buf)
	return nil
}

// 快速对写集合排序
type pdSlice []*kledger.PureData

// newPdSlice new a slice instance for PureData
func newPdSlice(vpd []*kledger.PureData) pdSlice {
	s := make([]*kledger.PureData, len(vpd))
	copy(s, vpd)
	return s
}

// Len length of slice of PureData
func (pds pdSlice) Len() int {
	return len(pds)
}

// Swap swap two pureData elements in a slice
func (pds pdSlice) Swap(i, j int) {
	pds[i], pds[j] = pds[j], pds[i]
}

// Less compare two pureData elements with pureData's key in a slice
func (pds pdSlice) Less(i, j int) bool {
	rawKeyI := makeRawKey(pds[i].GetBucket(), pds[i].GetKey())
	rawKeyJ := makeRawKey(pds[j].GetBucket(), pds[j].GetKey())
	ret := bytes.Compare(rawKeyI, rawKeyJ)
	if ret == 0 {
		// 注: 正常应该无法走到这个逻辑，因为写集合中的key一定是唯一的
		return bytes.Compare(pds[i].GetValue(), pds[j].GetValue()) < 0
	}
	return ret < 0
}

func equal(pd, vpd *kledger.PureData) bool {
	rawKeyI := makeRawKey(pd.GetBucket(), pd.GetKey())
	rawKeyJ := makeRawKey(vpd.GetBucket(), vpd.GetKey())
	ret := bytes.Compare(rawKeyI, rawKeyJ)
	if ret != 0 {
		return false
	}
	return bytes.Equal(pd.GetValue(), vpd.GetValue())
}

// Equal check if two PureData object equal
func Equal(pd, vpd []*kledger.PureData) bool {
	if len(pd) != len(vpd) {
		return false
	}
	pds := newPdSlice(pd)
	vpds := newPdSlice(vpd)
	sort.Sort(pds)
	sort.Sort(vpds)
	for i, v := range pds {
		if equal(v, vpds[i]) {
			continue
		}
		return false
	}
	return true
}
