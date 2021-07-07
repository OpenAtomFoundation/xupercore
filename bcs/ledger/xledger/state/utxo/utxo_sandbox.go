package utxo

import (
	"errors"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/protos"
	"math/big"
)

type UTXOSandbox struct {
	inputCache  []*protos.TxInput
	outputCache []*protos.TxOutput
	utxoReader  contract.UtxoReader
}

func NewUTXOSandbox(utxoReader contract.UtxoReader) *UTXOSandbox {
	return &UTXOSandbox{
		outputCache: []*protos.TxOutput{},
		utxoReader:  utxoReader,
	}
}

func (u *UTXOSandbox) Transfer(from, to string, amount *big.Int) error {
	if amount.Cmp(new(big.Int)) == 0 {
		return errors.New("should  be large than zero")
	}
	inputs, _, total, err := u.utxoReader.SelectUtxo(from, amount, true, false)
	if err != nil {
		return err
	}
	u.inputCache = append(u.inputCache, inputs...)
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

func (uc *UTXOSandbox) GetUTXORWSets() contract.UTXORWSet {
	return contract.UTXORWSet{
		Rset: uc.inputCache,
		WSet: uc.outputCache,
	}
}
