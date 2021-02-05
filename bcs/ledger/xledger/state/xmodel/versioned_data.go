package xmodel

import (
	"fmt"

	kledger "github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/protos"
)

func parseVersion(version string) ([]byte, int, error) {
	txid := []byte{}
	offset := 0
	okNum, err := fmt.Sscanf(version, "%x_%d", &txid, &offset)
	if okNum != 2 && err != nil {
		return nil, 0, fmt.Errorf("parseVersion failed, invalid version: %s", version)
	}
	return txid, offset, nil
}

//GetTxidFromVersion parse version and fetch txid from version string
func GetTxidFromVersion(version string) []byte {
	txid, _, err := parseVersion(version)
	if err != nil {
		return []byte("")
	}
	return txid
}

// MakeVersion generate a version by txid and offset, version = txid_offset
func MakeVersion(txid []byte, offset int32) string {
	return fmt.Sprintf("%x_%d", txid, offset)
}

// GetVersion get VersionedData's version, if refTxid is nil, return ""
func GetVersion(vd *kledger.VersionedData) string {
	if vd.RefTxid == nil {
		return ""
	}
	return MakeVersion(vd.RefTxid, vd.RefOffset)
}

// GetVersionOfTxInput get version of TxInput
func GetVersionOfTxInput(txIn *protos.TxInputExt) string {
	if txIn.RefTxid == nil {
		return ""
	}
	return MakeVersion(txIn.RefTxid, txIn.RefOffset)
}

// GetTxOutputs get transaction outputs
func GetTxOutputs(pds []*kledger.PureData) []*protos.TxOutputExt {
	outputs := make([]*protos.TxOutputExt, 0, len(pds))
	for _, pd := range pds {
		outputs = append(outputs, &protos.TxOutputExt{
			Bucket: pd.Bucket,
			Key:    pd.Key,
			Value:  pd.Value,
		})
	}
	return outputs
}

// GetTxInputs get transaction inputs
func GetTxInputs(vds []*kledger.VersionedData) []*protos.TxInputExt {
	inputs := make([]*protos.TxInputExt, 0, len(vds))
	for _, vd := range vds {
		inputs = append(inputs, &protos.TxInputExt{
			Bucket:    vd.GetPureData().GetBucket(),
			Key:       vd.GetPureData().GetKey(),
			RefTxid:   vd.RefTxid,
			RefOffset: vd.RefOffset,
		})
	}
	return inputs
}

// IsEmptyVersionedData check if VersionedData is empty
func IsEmptyVersionedData(vd *kledger.VersionedData) bool {
	return vd.RefTxid == nil && vd.RefOffset == 0
}

func makeEmptyVersionedData(bucket string, key []byte) *kledger.VersionedData {
	verData := &kledger.VersionedData{PureData: &kledger.PureData{}}
	verData.PureData.Bucket = bucket
	verData.PureData.Key = key
	return verData
}
