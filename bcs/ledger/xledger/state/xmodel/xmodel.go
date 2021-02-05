package xmodel

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	kledger "github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/cache"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/protos"
)

const (
	bucketExtUTXOCacheSize = 1024

	// TransientBucket is the name of bucket that only appears in tx output set
	// but does't persists in xmodel
	TransientBucket = "$transient"
)

var (
	contractUtxoInputKey  = []byte("ContractUtxo.Inputs")
	contractUtxoOutputKey = []byte("ContractUtxo.Outputs")
)

// XModel xmodel data structure
type XModel struct {
	ledger          *ledger.Ledger
	stateDB         kvdb.Database
	unconfirmTable  kvdb.Database
	extUtxoTable    kvdb.Database
	extUtxoDelTable kvdb.Database
	logger          logs.Logger
	batchCache      *sync.Map
	lastBatch       kvdb.Batch
	// extUtxoCache caches per bucket key-values using version as key
	extUtxoCache sync.Map // map[string]*LRUCache
}

// NewXuperModel new an instance of XModel
func NewXModel(sctx *context.StateCtx, stateDB kvdb.Database) (*XModel, error) {
	return &XModel{
		ledger:          sctx.Ledger,
		stateDB:         stateDB,
		unconfirmTable:  kvdb.NewTable(stateDB, pb.UnconfirmedTablePrefix),
		extUtxoTable:    kvdb.NewTable(stateDB, pb.ExtUtxoTablePrefix),
		extUtxoDelTable: kvdb.NewTable(stateDB, pb.ExtUtxoDelTablePrefix),
		logger:          sctx.XLog,
		batchCache:      &sync.Map{},
	}, nil
}

func (s *XModel) CreateSnapshot(blkId []byte) (kledger.XMReader, error) {
	// 查询快照区块高度
	blkInfo, err := s.ledger.QueryBlockHeader(blkId)
	if err != nil {
		return nil, fmt.Errorf("query block header fail.block_id:%s, err:%v",
			hex.EncodeToString(blkId), err)
	}

	xms := &xModSnapshot{
		xmod:      s,
		logger:    s.logger,
		blkHeight: blkInfo.Height,
		blkId:     blkId,
	}
	return xms, nil
}

func (s *XModel) CreateXMSnapshotReader(blkId []byte) (kledger.XMSnapshotReader, error) {
	xMReader, err := s.CreateSnapshot(blkId)
	if err != nil {
		return nil, err
	}

	return NewXMSnapshotReader(xMReader), nil
}

func (s *XModel) updateExtUtxo(tx *pb.Transaction, batch kvdb.Batch) error {
	for offset, txOut := range tx.TxOutputsExt {
		if txOut.Bucket == TransientBucket {
			continue
		}
		bucketAndKey := makeRawKey(txOut.Bucket, txOut.Key)
		valueVersion := MakeVersion(tx.Txid, int32(offset))
		if isDelFlag(txOut.Value) {
			putKey := append([]byte(pb.ExtUtxoDelTablePrefix), bucketAndKey...)
			delKey := append([]byte(pb.ExtUtxoTablePrefix), bucketAndKey...)
			batch.Delete(delKey)
			batch.Put(putKey, []byte(valueVersion))
			s.logger.Trace("    xmodel put gc", "putkey", string(putKey), "version", valueVersion)
			s.logger.Trace("    xmodel del", "delkey", string(delKey), "version", valueVersion)
		} else {
			putKey := append([]byte(pb.ExtUtxoTablePrefix), bucketAndKey...)
			batch.Put(putKey, []byte(valueVersion))
			s.logger.Trace("    xmodel put", "putkey", string(putKey), "version", valueVersion)
		}
		if len(tx.Blockid) > 0 {
			s.batchCache.Store(string(bucketAndKey), valueVersion)
		}
		s.bucketCacheStore(txOut.Bucket, valueVersion, &kledger.VersionedData{
			RefTxid:   tx.Txid,
			RefOffset: int32(offset),
			PureData: &kledger.PureData{
				Key:    txOut.Key,
				Value:  txOut.Value,
				Bucket: txOut.Bucket,
			},
		})
	}
	return nil
}

// DoTx running a transaction and update extUtxoTable
func (s *XModel) DoTx(tx *pb.Transaction, batch kvdb.Batch) error {
	if len(tx.Blockid) > 0 {
		s.cleanCache(batch)
	}
	err := s.verifyInputs(tx)
	if err != nil {
		return err
	}
	err = s.verifyOutputs(tx)
	if err != nil {
		return err
	}
	err = s.updateExtUtxo(tx, batch)
	if err != nil {
		return err
	}
	return nil
}

// UndoTx rollback a transaction and update extUtxoTable
func (s *XModel) UndoTx(tx *pb.Transaction, batch kvdb.Batch) error {
	s.cleanCache(batch)
	inputVersionMap := map[string]string{}
	for _, txIn := range tx.TxInputsExt {
		rawKey := string(makeRawKey(txIn.Bucket, txIn.Key))
		version := GetVersionOfTxInput(txIn)
		inputVersionMap[rawKey] = version
	}
	for _, txOut := range tx.TxOutputsExt {
		if txOut.Bucket == TransientBucket {
			continue
		}
		bucketAndKey := makeRawKey(txOut.Bucket, txOut.Key)
		previousVersion := inputVersionMap[string(bucketAndKey)]
		if previousVersion == "" {
			delKey := append([]byte(pb.ExtUtxoTablePrefix), bucketAndKey...)
			batch.Delete(delKey)
			s.logger.Trace("    undo xmodel del", "delkey", string(delKey))
			s.batchCache.Store(string(bucketAndKey), "")
		} else {
			verData, err := s.fetchVersionedData(txOut.Bucket, previousVersion)
			if err != nil {
				return err
			}
			if isDelFlag(verData.PureData.Value) { //previous version is del
				putKey := append([]byte(pb.ExtUtxoDelTablePrefix), bucketAndKey...)
				batch.Put(putKey, []byte(previousVersion))
				delKey := append([]byte(pb.ExtUtxoTablePrefix), bucketAndKey...)
				batch.Delete(delKey)
				s.logger.Trace("    undo xmodel put gc", "putkey", string(putKey), "prever", previousVersion)
				s.logger.Trace("    undo xmodel del", "del key", string(delKey), "prever", previousVersion)
			} else {
				putKey := append([]byte(pb.ExtUtxoTablePrefix), bucketAndKey...)
				batch.Put(putKey, []byte(previousVersion))
				s.logger.Trace("    undo xmodel put", "putkey", string(putKey), "prever", previousVersion)
				if isDelFlag(txOut.Value) { //current version is del
					delKey := append([]byte(pb.ExtUtxoDelTablePrefix), bucketAndKey...)
					batch.Delete(delKey) //remove garbage in gc table
				}
			}
			s.batchCache.Store(string(bucketAndKey), previousVersion)
		}
	}
	return nil
}

func (s *XModel) fetchVersionedData(bucket, version string) (*kledger.VersionedData, error) {
	value, ok := s.bucketCacheGet(bucket, version)
	if ok {
		return value, nil
	}
	txid, offset, err := parseVersion(version)
	if err != nil {
		return nil, err
	}
	tx, _, err := s.queryTx(txid)
	if err != nil {
		return nil, err
	}
	if offset >= len(tx.TxOutputsExt) {
		return nil, fmt.Errorf("xmodel.Get failed, offset overflow: %d, %d", offset, len(tx.TxOutputsExt))
	}
	txOutputs := tx.TxOutputsExt[offset]
	value = &kledger.VersionedData{
		RefTxid:   txid,
		RefOffset: int32(offset),
		PureData: &kledger.PureData{
			Key:    txOutputs.Key,
			Value:  txOutputs.Value,
			Bucket: txOutputs.Bucket,
		},
	}
	s.bucketCacheStore(bucket, version, value)
	return value, nil
}

// GetUncommited get value for specific key, return the value with version, even it is in batch cache
func (s *XModel) GetUncommited(bucket string, key []byte) (*kledger.VersionedData, error) {
	rawKey := makeRawKey(bucket, key)
	cacheObj, cacheHit := s.batchCache.Load(string(rawKey))
	if cacheHit {
		version := cacheObj.(string)
		if version == "" {
			return makeEmptyVersionedData(bucket, key), nil
		}
		return s.fetchVersionedData(bucket, version)
	}
	return s.Get(bucket, key)
}

// GetFromLedger get data directely from ledger
func (s *XModel) GetFromLedger(txin *protos.TxInputExt) (*kledger.VersionedData, error) {
	if txin.RefTxid == nil {
		return makeEmptyVersionedData(txin.Bucket, txin.Key), nil
	}
	version := MakeVersion(txin.RefTxid, txin.RefOffset)
	return s.fetchVersionedData(txin.Bucket, version)
}

// Get get value for specific key, return value with version
func (s *XModel) Get(bucket string, key []byte) (*kledger.VersionedData, error) {
	rawKey := makeRawKey(bucket, key)
	version, err := s.extUtxoTable.Get(rawKey)
	if err != nil {
		if kvdb.ErrNotFound(err) {
			//从回收站Get, 因为这个utxo可能是被删除了，RefTxid需要引用
			version, err = s.extUtxoDelTable.Get(rawKey)
			if err != nil {
				if kvdb.ErrNotFound(err) {
					return makeEmptyVersionedData(bucket, key), nil
				}
				return nil, err
			}
			return s.fetchVersionedData(bucket, string(version))
		}
		return nil, err
	}
	return s.fetchVersionedData(bucket, string(version))
}

// GetWithTxStatus likes Get but also return tx status information
func (s *XModel) GetWithTxStatus(bucket string, key []byte) (*kledger.VersionedData, bool, error) {
	data, err := s.Get(bucket, key)
	if err != nil {
		return nil, false, err
	}
	exists, err := s.ledger.HasTransaction(data.RefTxid)
	if err != nil {
		return nil, false, err
	}
	return data, exists, nil
}

// Select select all kv from a bucket, can set key range, left closed, right opend
func (s *XModel) Select(bucket string, startKey []byte, endKey []byte) (kledger.XMIterator, error) {
	rawStartKey := makeRawKey(bucket, startKey)
	rawEndKey := makeRawKey(bucket, endKey)
	iter := &XMIterator{
		bucket: bucket,
		iter:   s.extUtxoTable.NewIteratorWithRange(rawStartKey, rawEndKey),
		model:  s,
	}
	return iter, nil
}

func (s *XModel) queryTx(txid []byte) (*pb.Transaction, bool, error) {
	unconfirmTx, err := queryUnconfirmTx(txid, s.unconfirmTable)
	if err != nil {
		if !kvdb.ErrNotFound(err) {
			return nil, false, err
		}
	} else {
		return unconfirmTx, false, nil
	}
	confirmedTx, err := s.ledger.QueryTransaction(txid)
	if err != nil {
		return nil, false, err
	}
	return confirmedTx, true, nil
}

// QueryTx query transaction including unconfirmed table and confirmed table
func (s *XModel) QueryTx(txid []byte) (*pb.Transaction, bool, error) {
	tx, status, err := s.queryTx(txid)
	if err != nil {
		return nil, status, err
	}
	return tx, status, nil
}

// QueryBlock query block from ledger
func (s *XModel) QueryBlock(blockid []byte) (*pb.InternalBlock, error) {
	block, err := s.ledger.QueryBlock(blockid)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// CleanCache clear batchCache and lastBatch
func (s *XModel) CleanCache() {
	s.cleanCache(nil)
}

func (s *XModel) cleanCache(newBatch kvdb.Batch) {
	if newBatch != s.lastBatch {
		s.batchCache = &sync.Map{}
		s.lastBatch = newBatch
	}
}

func (s *XModel) bucketCache(bucket string) *cache.LRUCache {
	icache, ok := s.extUtxoCache.Load(bucket)
	if ok {
		return icache.(*cache.LRUCache)
	}
	cache := cache.NewLRUCache(bucketExtUTXOCacheSize)
	s.extUtxoCache.Store(bucket, cache)
	return cache
}

func (s *XModel) bucketCacheStore(bucket, version string, value *kledger.VersionedData) {
	cache := s.bucketCache(bucket)
	cache.Add(version, value)
}

func (s *XModel) bucketCacheGet(bucket, version string) (*kledger.VersionedData, bool) {
	cache := s.bucketCache(bucket)
	value, ok := cache.Get(version)
	if !ok {
		return nil, false
	}
	return value.(*kledger.VersionedData), true
}

// BucketCacheDelete gen write key with perfix
func (s *XModel) BucketCacheDelete(bucket, version string) {
	cache := s.bucketCache(bucket)
	cache.Del(version)
}

// GenWriteKeyWithPrefix gen write key with perfix
func GenWriteKeyWithPrefix(txOutputExt *protos.TxOutputExt) string {
	bucket := txOutputExt.GetBucket()
	key := txOutputExt.GetKey()
	baseWriteSetKey := bucket + fmt.Sprintf("%s", key)
	return pb.ExtUtxoTablePrefix + baseWriteSetKey
}

// ParseContractUtxoInputs parse contract utxo inputs from tx write sets
func ParseContractUtxoInputs(tx *pb.Transaction) ([]*protos.TxInput, error) {
	var (
		utxoInputs []*protos.TxInput
		extInput   []byte
	)
	for _, out := range tx.GetTxOutputsExt() {
		if out.GetBucket() != TransientBucket {
			continue
		}
		if bytes.Equal(out.GetKey(), contractUtxoInputKey) {
			extInput = out.GetValue()
		}
	}
	if extInput != nil {
		err := UnmsarshalMessages(extInput, &utxoInputs)
		if err != nil {
			return nil, err
		}
	}
	return utxoInputs, nil
}
