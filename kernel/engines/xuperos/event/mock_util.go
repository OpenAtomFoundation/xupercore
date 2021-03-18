package event

import (
	"crypto/rand"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/xmodel"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/protos"
)

type blockBuilder struct {
	block *lpb.InternalBlock
}

func newBlockBuilder() *blockBuilder {
	return &blockBuilder{
		block: &lpb.InternalBlock{
			Blockid: makeRandID(),
		},
	}
}

func (b *blockBuilder) AddTx(tx ...*lpb.Transaction) *blockBuilder {
	b.block.Transactions = append(b.block.Transactions, tx...)
	return b
}

func (b *blockBuilder) Block() *lpb.InternalBlock {
	return b.block
}

type txBuilder struct {
	tx     *lpb.Transaction
	events []*protos.ContractEvent
}

func newTxBuilder() *txBuilder {
	return &txBuilder{
		tx: &lpb.Transaction{
			Txid: makeRandID(),
		},
	}
}

func (t *txBuilder) Initiator(addr string) *txBuilder {
	t.tx.Initiator = addr
	return t
}

func (t *txBuilder) AuthRequire(addr ...string) *txBuilder {
	t.tx.AuthRequire = addr
	return t
}

func (t *txBuilder) Transfer(from, to, amount string) *txBuilder {
	input := &protos.TxInput{
		RefTxid:  makeRandID(),
		FromAddr: []byte(from),
		Amount:   []byte(amount),
	}
	output := &protos.TxOutput{
		ToAddr: []byte(to),
		Amount: []byte(amount),
	}
	t.tx.TxInputs = append(t.tx.TxInputs, input)
	t.tx.TxOutputs = append(t.tx.TxOutputs, output)
	return t
}

func (t *txBuilder) Invoke(contract, method string, events ...*protos.ContractEvent) *txBuilder {
	req := &protos.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: contract,
		MethodName:   method,
	}
	t.tx.ContractRequests = append(t.tx.ContractRequests, req)
	t.events = append(t.events, events...)
	return t
}

func (t *txBuilder) eventRWSet() []*protos.TxOutputExt {
	buf, _ := xmodel.MarshalMessages(t.events)
	return []*protos.TxOutputExt{
		{
			Bucket: xmodel.TransientBucket,
			Key:    []byte("contractEvent"),
			Value:  buf,
		},
	}
}

func (t *txBuilder) Tx() *lpb.Transaction {
	t.tx.TxOutputsExt = t.eventRWSet()
	return t.tx
}

func makeRandID() []byte {
	buf := make([]byte, 32)
	rand.Read(buf)
	return buf
}
