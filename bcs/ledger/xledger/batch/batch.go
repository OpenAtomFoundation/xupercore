package batch

import (
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

type RichBatch struct {
	kvdb.Batch
}

func NewRichBatch(batch kvdb.Batch) *RichBatch {
	return &RichBatch{
		Batch: batch,
	}
}

// PutConfirmedTx updates data in confirmed table
func (b *RichBatch) PutConfirmedTx(txID []byte, pbTxBuf []byte) {
	key := append([]byte(xldgpb.ConfirmedTablePrefix), txID...)
	// TODO: deal with error
	_ = b.Put(key, pbTxBuf)
}

// PutUnconfirmedTx updates data in confirmed table
func (b *RichBatch) PutUnconfirmedTx(txID []byte, pbTxBuf []byte) {
	key := append([]byte(xldgpb.UnconfirmedTablePrefix), txID...)
	// TODO: deal with error
	_ = b.Put(key, pbTxBuf)
}

// DeleteUnconfirmedTx deletes data in unconfirmed table
func (b *RichBatch) DeleteUnconfirmedTx(txID []byte) {
	key := append([]byte(xldgpb.UnconfirmedTablePrefix), txID...)
	// TODO: deal with error
	_ = b.Delete(key)
}

// PutMeta updates data in meta table
func (b *RichBatch) PutMeta(keySuffix string, value []byte) error {
	key := []byte(xldgpb.MetaTablePrefix + keySuffix)
	return b.Put(key, value)
}

// PutExtUtxo updates data for Ext-UTXO
func (b *RichBatch) PutExtUtxo(bucketAndKey []byte, version string, previousDeleted bool) (err error) {
	// remove mark in Ext-UTXO-Delete table
	if previousDeleted {
		markKey := append([]byte(xldgpb.ExtUtxoDelTablePrefix), bucketAndKey...)
		if delErr := b.Delete(markKey); delErr != nil {
			err = delErr
		}
	}

	// update data in Ext-UTXO table
	dataKey := append([]byte(xldgpb.ExtUtxoTablePrefix), bucketAndKey...)
	if dataErr := b.Put(dataKey, []byte(version)); dataErr != nil {
		err = dataErr
	}
	return err
}

// SoftDeleteExtUtxo deletes(as mark) data for Ext-UTXO
// delete with data version
func (b *RichBatch) SoftDeleteExtUtxo(bucketAndKey []byte, version string) (err error) {
	// add mark in Ext-UTXO-Delete table
	markKey := append([]byte(xldgpb.ExtUtxoDelTablePrefix), bucketAndKey...)
	if delErr := b.Put(markKey, []byte(version)); delErr != nil {
		err = delErr
	}

	// remove data in Ext-UTXO table
	if dataErr := b.deleteExtUtxoData(bucketAndKey); dataErr != nil {
		err = dataErr
	}
	return err
}

// HardDeleteExtUtxo deletes(as remove) data for Ext-UTXO
func (b *RichBatch) HardDeleteExtUtxo(bucketAndKey []byte) {
	// TODO: deal with error
	_ = b.deleteExtUtxoData(bucketAndKey)
}

// deleteExtUtxoData deletes data from Ext-UTXO table
func (b *RichBatch) deleteExtUtxoData(bucketAndKey []byte) error {
	dataKey := append([]byte(xldgpb.ExtUtxoTablePrefix), bucketAndKey...)
	return b.Delete(dataKey)
}

// PutUtxoWithPrefix updates data in UTXO table
func (b *RichBatch) PutUtxoWithPrefix(utxo string, data []byte) {
	// TODO: deal with error
	keyWithPrefix := []byte(utxo)
	_ = b.Put(keyWithPrefix, data)
}

// DeleteUtxoWithPrefix deletes data in UTXO table
func (b *RichBatch) DeleteUtxoWithPrefix(utxo string) {
	// TODO: deal with error
	keyWithPrefix := []byte(utxo)
	_ = b.Delete(keyWithPrefix)
}
