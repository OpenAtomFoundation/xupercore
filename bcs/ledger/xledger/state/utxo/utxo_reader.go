package utxo

import (
	"bytes"
	"errors"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/protos"
	"math/big"
)

type UTXOReader struct {
	inputCache []*protos.TxInput
	inputIdx   int
}

func NewUTXOReaderFromInput(input []*protos.TxInput) contract.UtxoReader {
	return &UTXOReader{
		inputCache: input,
		inputIdx:   0,
	}
}

func (r *UTXOReader) SelectUtxo(from string, amount *big.Int, lock bool, excludeUnconfirmed bool) ([]*protos.TxInput, [][]byte, *big.Int, error) {
	fromBytes := []byte(from)
	inputCache := r.inputCache[r.inputIdx:]
	sum := new(big.Int)
	n := 0
	for _, input := range inputCache {
		n++
		// Since contract calls bridge serially, a mismatched from address is an error
		if !bytes.Equal(input.GetFromAddr(), fromBytes) {
			return nil, nil, nil, errors.New("from address mismatch in utxo cache")
		}
		sum.Add(sum, new(big.Int).SetBytes(input.GetAmount()))
		if sum.Cmp(amount) >= 0 {
			break
		}
	}
	if sum.Cmp(amount) < 0 {
		return nil, nil, nil, errors.New("utxo not enough in utxo cache")
	}
	r.inputIdx += n
	return inputCache[:n], nil, sum, nil
}
