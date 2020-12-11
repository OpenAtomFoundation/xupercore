package models

import (
	"fmt"

	ledgpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/pb"
	edef "github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
)

type TxProceor struct {
	engine edef.Engine
	log    logs.Logger
}

func NewTxProceor(engine edef.Engine, log logs.Logger) *TxProceor {
	return &TxProceor{
		engine: engine,
		log:    log,
	}
}

func (t *TxProceor) SubmitTx(bcName string, tx *ledgpb.Transaction) error {

}

func (t *TxProceor) QueryTx(txId string) (*ledgpb.Transaction, error) {

}
