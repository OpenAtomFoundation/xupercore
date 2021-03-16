package sandbox

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/xmodel"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/protos"
)

var (
	// ErrHasDel is returned when key was marked as del
	ErrHasDel = errors.New("Key has been mark as del")
	// ErrNotFound is returned when key is not found
	ErrNotFound = errors.New("Key not found")
)

var (
	contractUtxoInputKey  = []byte("ContractUtxo.Inputs")
	contractUtxoOutputKey = []byte("ContractUtxo.Outputs")
	crossQueryInfosKey    = []byte("CrossQueryInfos")
	contractEventKey      = []byte("contractEvent")
)

var (
	_ contract.StateSandbox = (*XMCache)(nil)
)

// UtxoVM manages utxos
type UtxoVM interface {
	SelectUtxos(fromAddr string, fromPubKey string, totalNeed *big.Int, needLock, excludeUnconfirmed bool) ([]*protos.TxInput, [][]byte, *big.Int, error)
}

// XMCache data structure for XModel Cache
type XMCache struct {
	// Key: bucket_key; Value: VersionedData
	inputsCache *MemXModel // bucket -> {k1:v1, k2:v2}
	// Key: bucket_key; Value: PureData
	outputsCache *MemXModel

	model ledger.XMReader

	// utxoCache       *UtxoCache
	// crossQueryCache *CrossQueryCache
	events []*protos.ContractEvent
}

// NewXModelCache new an instance of XModel Cache
func NewXModelCache(model ledger.XMReader) *XMCache {
	return &XMCache{
		model:        model,
		inputsCache:  NewMemXModel(),
		outputsCache: NewMemXModel(),
		// utxoCache:       NewUtxoCache(utxovm),
		// crossQueryCache: NewCrossQueryCache(),
	}
}

// Get 读取一个key的值，返回的value就是有版本的data
func (xc *XMCache) Get(bucket string, key []byte) ([]byte, error) {
	// Level1: get from outputsCache
	data, err := xc.getFromOuputsCache(bucket, key)
	if err != nil && err != ErrNotFound {
		return nil, err
	}

	if err == nil {
		return data.PureData.Value, nil
	}

	// Level2: get and set from inputsCache
	verData, err := xc.getAndSetFromInputsCache(bucket, key)
	if err != nil {
		return nil, err
	}
	if IsEmptyVersionedData(verData) {
		return nil, ErrNotFound
	}
	if IsDelFlag(verData.GetPureData().GetValue()) {
		return nil, ErrHasDel
	}
	return verData.GetPureData().GetValue(), nil
}

// Level1 读取，从outputsCache中读取
func (xc *XMCache) getFromOuputsCache(bucket string, key []byte) (*ledger.VersionedData, error) {
	data, err := xc.outputsCache.Get(bucket, key)
	if err != nil {
		return nil, err
	}

	if IsDelFlag(data.PureData.Value) {
		return nil, ErrHasDel
	}
	return data, nil
}

// Level2 读取，从inputsCache中读取, 读取不到的情况下，如果isPenetrate为true，会更深一层次从model里读取，并且会将内容填充到readSets中
func (xc *XMCache) getAndSetFromInputsCache(bucket string, key []byte) (*ledger.VersionedData, error) {
	data, err := xc.inputsCache.Get(bucket, key)
	if err == nil {
		return data, nil
	}
	if err != nil && err != ErrNotFound {
		return nil, err
	}

	if err == ErrNotFound {
		data, err = xc.model.Get(bucket, key)
		if err != nil {
			return nil, err
		}
		xc.inputsCache.Put(bucket, key, data)
	}
	return data, nil
}

// Put put a pair of <key, value> into XModel Cache
func (xc *XMCache) Put(bucket string, key []byte, value []byte) error {
	_, err := xc.getFromOuputsCache(bucket, key)
	if err != nil && err != ErrNotFound && err != ErrHasDel {
		return err
	}

	val := &ledger.VersionedData{
		PureData: &ledger.PureData{
			Key:    key,
			Value:  value,
			Bucket: bucket,
		},
	}
	if bucket != TransientBucket {
		// put 前先强制get一下
		xc.Get(bucket, key)
	}
	return xc.outputsCache.Put(bucket, key, val)
}

// Del delete one key from outPutCache, marked its value as `DelFlag`
func (xc *XMCache) Del(bucket string, key []byte) error {
	return xc.Put(bucket, key, []byte(DelFlag))
}

// Select select all kv from a bucket, can set key range, left closed, right opend
// When xc.isPenetrate equals true, three-way merge, When xc.isPenetrate equals false, two-way merge
func (xc *XMCache) Select(bucket string, startKey []byte, endKey []byte) (contract.Iterator, error) {
	return xc.newXModelCacheIterator(bucket, startKey, endKey)
}

// newXModelCacheIterator new an instance of XModel Cache iterator
func (mc *XMCache) newXModelCacheIterator(bucket string, startKey []byte, endKey []byte) (contract.Iterator, error) {
	iter, _ := mc.outputsCache.Select(bucket, startKey, endKey)
	outputIter := iter

	iter, _ = mc.inputsCache.Select(bucket, startKey, endKey)
	inputIter := newStripDelIterator(iter)

	backendIter, err := mc.model.Select(bucket, startKey, endKey)
	if err != nil {
		return nil, err
	}
	backendIter = newStripDelIterator(
		newRsetIterator(bucket, backendIter, mc),
	)
	// return newContractIterator(backendIter), nil

	// 优先级顺序 outputIter -> inputIter -> backendIter
	// 意味着如果一个key在三个迭代器里面同时出现，优先级高的会覆盖优先级底的
	multiIter := newMultiIterator(inputIter, backendIter)
	multiIter = newMultiIterator(outputIter, multiIter)
	return newContractIterator(multiIter), nil
}

// GetRWSets get read/write sets
func (xc *XMCache) RWSet() *contract.RWSet {
	readSet := xc.getReadSets()
	writeSet := xc.getWriteSets()

	return &contract.RWSet{
		RSet: readSet,
		WSet: writeSet,
	}
}

func (xc *XMCache) getReadSets() []*ledger.VersionedData {
	var readSets []*ledger.VersionedData
	iter := xc.inputsCache.NewIterator()
	defer iter.Close()
	for iter.Next() {
		val := iter.Value()
		readSets = append(readSets, val)
	}
	return readSets
}

func (xc *XMCache) getWriteSets() []*ledger.PureData {
	var writeSets []*ledger.PureData
	iter := xc.outputsCache.NewIterator()
	defer iter.Close()
	for iter.Next() {
		val := iter.Value()
		writeSets = append(writeSets, val.PureData)
	}
	return writeSets
}

// // Transfer transfer tokens using utxo
// func (xc *XMCache) Transfer(from, to string, amount *big.Int) error {
// 	return xc.utxoCache.Transfer(from, to, amount)
// }

// // GetUtxoRWSets returns the inputs and outputs of utxo
// func (xc *XMCache) GetUtxoRWSets() ([]*pb.TxInput, []*pb.TxOutput) {
// 	return xc.utxoCache.GetRWSets()
// }

// // putUtxos put utxos to TransientBucket
// func (xc *XMCache) putUtxos(inputs []*pb.TxInput, outputs []*pb.TxOutput) error {
// 	var in, out []byte
// 	var err error
// 	if len(inputs) != 0 {
// 		in, err = MarshalMessages(inputs)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	if len(outputs) != 0 {
// 		out, err = MarshalMessages(outputs)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	if in != nil {
// 		err = xc.Put(TransientBucket, contractUtxoInputKey, in)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	if out != nil {
// 		err = xc.Put(TransientBucket, contractUtxoOutputKey, out)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (xc *XMCache) writeUtxoRWSet() error {
// 	return xc.putUtxos(xc.GetUtxoRWSets())
// }

// // ParseContractUtxoInputs parse contract utxo inputs from tx write sets
// func ParseContractUtxoInputs(tx *pb.Transaction) ([]*pb.TxInput, error) {
// 	var (
// 		utxoInputs []*pb.TxInput
// 		extInput   []byte
// 	)
// 	for _, out := range tx.GetTxOutputsExt() {
// 		if out.GetBucket() != TransientBucket {
// 			continue
// 		}
// 		if bytes.Equal(out.GetKey(), contractUtxoInputKey) {
// 			extInput = out.GetValue()
// 		}
// 	}
// 	if extInput != nil {
// 		err := UnmsarshalMessages(extInput, &utxoInputs)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
// 	return utxoInputs, nil
// }

// // ParseContractUtxo parse contract utxos from tx write sets
// func ParseContractUtxo(tx *pb.Transaction) ([]*pb.TxInput, []*pb.TxOutput, error) {
// 	var (
// 		utxoInputs  []*pb.TxInput
// 		utxoOutputs []*pb.TxOutput
// 		extInput    []byte
// 		extOutput   []byte
// 	)
// 	for _, out := range tx.GetTxOutputsExt() {
// 		if out.GetBucket() != TransientBucket {
// 			continue
// 		}
// 		if bytes.Equal(out.GetKey(), contractUtxoInputKey) {
// 			extInput = out.GetValue()
// 		}
// 		if bytes.Equal(out.GetKey(), contractUtxoOutputKey) {
// 			extOutput = out.GetValue()
// 		}
// 	}
// 	if extInput != nil {
// 		err := UnmsarshalMessages(extInput, &utxoInputs)
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 	}
// 	if extOutput != nil {
// 		err := UnmsarshalMessages(extOutput, &utxoOutputs)
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 	}
// 	return utxoInputs, utxoOutputs, nil
// }

// func makeInputsMap(txInputs []*pb.TxInput) map[string]bool {
// 	res := map[string]bool{}
// 	if len(txInputs) == 0 {
// 		return nil
// 	}
// 	for _, v := range txInputs {
// 		utxoKey := string(v.GetRefTxid()) + strconv.FormatInt(int64(v.GetRefOffset()), 10)
// 		res[utxoKey] = true
// 	}
// 	return res
// }

// func isSubOutputs(contractOutputs, txOutputs []*pb.TxOutput) bool {
// 	markedOutput := map[string]int{}
// 	for _, v := range txOutputs {
// 		key := string(v.GetAmount()) + string(v.GetToAddr())
// 		markedOutput[key]++
// 	}

// 	for _, v := range contractOutputs {
// 		key := string(v.GetAmount()) + string(v.GetToAddr())
// 		if val, ok := markedOutput[key]; !ok {
// 			return false
// 		} else if val < 1 {
// 			return false
// 		} else {
// 			markedOutput[key] = val - 1
// 		}
// 	}
// 	return true
// }

// // IsContractUtxoEffective check if contract utxo in tx utxo
// func IsContractUtxoEffective(contractTxInputs []*pb.TxInput, contractTxOutputs []*pb.TxOutput, tx *pb.Transaction) bool {
// 	if len(contractTxInputs) > len(tx.GetTxInputs()) || len(contractTxOutputs) > len(tx.GetTxOutputs()) {
// 		return false
// 	}

// 	contractTxInputsMap := makeInputsMap(contractTxInputs)
// 	txInputsMap := makeInputsMap(tx.GetTxInputs())
// 	for k := range contractTxInputsMap {
// 		if !(txInputsMap[k]) {
// 			return false
// 		}
// 	}

// 	if !isSubOutputs(contractTxOutputs, tx.GetTxOutputs()) {
// 		return false
// 	}
// 	return true
// }

// // CrossQuery will query contract from other chain
// func (xc *XMCache) CrossQuery(crossQueryRequest *pb.CrossQueryRequest, queryMeta *pb.CrossQueryMeta) (*pb.ContractResponse, error) {
// 	return xc.crossQueryCache.CrossQuery(crossQueryRequest, queryMeta)
// }

// // ParseCrossQuery parse cross query from tx
// func ParseCrossQuery(tx *pb.Transaction) ([]*pb.CrossQueryInfo, error) {
// 	var (
// 		crossQueryInfos []*pb.CrossQueryInfo
// 		queryInfos      []byte
// 	)
// 	for _, out := range tx.GetTxOutputsExt() {
// 		if out.GetBucket() != TransientBucket {
// 			continue
// 		}
// 		if bytes.Equal(out.GetKey(), crossQueryInfosKey) {
// 			queryInfos = out.GetValue()
// 		}
// 	}
// 	if queryInfos != nil {
// 		err := UnmsarshalMessages(queryInfos, &crossQueryInfos)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
// 	return crossQueryInfos, nil
// }

// // IsCrossQueryEffective check if crossQueryInfos effective
// // TODO: zq
// func IsCrossQueryEffective(cqi []*pb.CrossQueryInfo, tx *pb.Transaction) bool {
// 	return true
// }

// // PutCrossQueries put queryInfos to db
// func (xc *XMCache) putCrossQueries(queryInfos []*pb.CrossQueryInfo) error {
// 	var qi []byte
// 	var err error
// 	if len(queryInfos) != 0 {
// 		qi, err = MarshalMessages(queryInfos)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	if qi != nil {
// 		err = xc.Put(TransientBucket, crossQueryInfosKey, qi)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (xc *XMCache) writeCrossQueriesRWSet() error {
// 	return xc.putCrossQueries(xc.crossQueryCache.GetCrossQueryRWSets())
// }

// ParseContractEvents parse contract events from tx
func ParseContractEvents(tx *lpb.Transaction) ([]*protos.ContractEvent, error) {
	var events []*protos.ContractEvent
	for _, out := range tx.GetTxOutputsExt() {
		if out.GetBucket() != TransientBucket {
			continue
		}
		if !bytes.Equal(out.GetKey(), contractEventKey) {
			continue
		}
		err := xmodel.UnmsarshalMessages(out.GetValue(), &events)
		if err != nil {
			return nil, err
		}
		break
	}
	return events, nil
}

// AddEvent add contract event to xmodel cache
func (xc *XMCache) AddEvent(events ...*protos.ContractEvent) {
	xc.events = append(xc.events, events...)
}

func (xc *XMCache) writeEventRWSet() error {
	if len(xc.events) == 0 {
		return nil
	}
	buf, err := xmodel.MarshalMessages(xc.events)
	if err != nil {
		return err
	}
	return xc.Put(TransientBucket, contractEventKey, buf)
}

// WriteTransientBucket write transient bucket data.
// transient bucket is a special bucket used to store some data
// generated during the execution of the contract, but will not be referenced by other txs.
func (xc *XMCache) Flush() error {
	var err error
	// err = xc.writeUtxoRWSet()
	// if err != nil {
	// 	return err
	// }

	// err = xc.writeCrossQueriesRWSet()
	// if err != nil {
	// 	return err
	// }

	err = xc.writeEventRWSet()
	if err != nil {
		return err
	}
	return nil
}
