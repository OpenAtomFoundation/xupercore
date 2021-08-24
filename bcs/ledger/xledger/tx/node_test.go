package tx

import (
	"testing"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/protos"
)

func TestNewNode(t *testing.T) {
	testTx := &pb.Transaction{
		TxInputs: []*protos.TxInput{
			{RefTxid: []byte("test")},
		},
		TxInputsExt: []*protos.TxInputExt{
			{RefTxid: []byte("test")},
		},
	}

	n := NewNode("test", testTx)
	if n.txid != "test" {
		t.Error("new node test failed")
	}

	if len(n.txInputs) != 1 {
		t.Error("new node test failed")
	}
}

func TestGetAllChildren(t *testing.T) {

}
