package utxo

import (
	"bytes"
	"encoding/json"
	"math/big"
)

// UtxoItem the data structure of an UTXO item
type UtxoItem struct {
	Amount       *big.Int //utxo的面值
	FrozenHeight int64    //锁定until账本高度超过
}

func NewUtxoItem(amount []byte, frozenHeight int64) *UtxoItem {
	amountValue := big.NewInt(0)
	amountValue.SetBytes(amount)
	return &UtxoItem{
		Amount:       amountValue,
		FrozenHeight: frozenHeight,
	}
}

// Loads load UTXO item from JSON encoded data
func (i *UtxoItem) Loads(data []byte) error {
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	return decoder.Decode(i)
}

// Dumps dump UTXO item into JSON encoded data
func (i *UtxoItem) Dumps() ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	err := encoder.Encode(i)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// IsFrozen return true for frozen UTXO
func (i *UtxoItem) IsFrozen(curHeight int64) bool {
	return i.FrozenHeight > curHeight || i.FrozenHeight == -1
}

// IsEmpty return true for unset UTXO
func (i *UtxoItem) IsEmpty() bool {
	return i.Amount == nil
}
