package utxo

import (
	"bytes"
	"errors"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/protos"
	"math/big"
)

type UTXOSandbox struct {
	inputCache  []*protos.TxInput
	outputCache []*protos.TxOutput
	inputIdx    int
	Penetrate   bool
	utxovm      contract.UtxoVM
}

func NewUTXOSandbox(vm contract.UtxoVM, inputs []*protos.TxInput, Penetrate bool) *UTXOSandbox {
	return &UTXOSandbox{
		utxovm:      vm,
		inputCache:  inputs,
		outputCache: []*protos.TxOutput{},
		Penetrate:   Penetrate,
		inputIdx:    0,
	}
}

func (u *UTXOSandbox) selectUtxos(from string, amount *big.Int) (*big.Int, error) {
	if u.Penetrate {
		inputs, _, total, err := u.utxovm.SelectUtxos(from, amount, true, false)
		if err != nil {
			return nil, err
		}
		u.inputCache = append(u.inputCache, inputs...)
		return total, nil
	}

	fromBytes := []byte(from)
	inputCache := u.inputCache[u.inputIdx:]
	sum := new(big.Int)
	n := 0
	for _, input := range inputCache {
		n++
		// Since contract calls bridge serially, a mismatched from address is an error
		if !bytes.Equal(input.GetFromAddr(), fromBytes) {
			return nil, errors.New("from address mismatch in utxo cache")
		}
		sum.Add(sum, new(big.Int).SetBytes(input.GetAmount()))
		if sum.Cmp(amount) >= 0 {
			break
		}
	}
	if sum.Cmp(amount) < 0 {
		return nil, errors.New("utxo not enough in utxo cache")
	}
	u.inputIdx += n
	return sum, nil
}

func (u *UTXOSandbox) Transfer(from, to string, amount *big.Int) error {
	if amount.Cmp(new(big.Int)) == 0 {
		return nil
	}
	total, err := u.selectUtxos(from, amount)
	if err != nil {
		return err
	}
	u.outputCache = append(u.outputCache, &protos.TxOutput{
		Amount: amount.Bytes(),
		ToAddr: []byte(to),
	})
	// make change
	if total.Cmp(amount) > 0 {
		u.outputCache = append(u.outputCache, &protos.TxOutput{
			Amount: new(big.Int).Sub(total, amount).Bytes(),
			ToAddr: []byte(from),
		})
	}
	return nil
}

func (uc *UTXOSandbox) GetUTXORWSets() ([]*protos.TxInput, []*protos.TxOutput) {

	if uc.Penetrate {
		return uc.inputCache, uc.outputCache
	}
	return uc.inputCache[:uc.inputIdx], uc.outputCache
}
