package mock

import (
	"github.com/OpenAtomFoundation/xupercore/global/bcs/ledger/xledger/state"
	"github.com/OpenAtomFoundation/xupercore/global/bcs/ledger/xledger/xldgpb"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract/bridge/pb"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/ledger"
)

type fakeChainCore struct {
}

// GetAccountAddress get addresses associated with account name
func (f *fakeChainCore) GetAccountAddresses(accountName string) ([]string, error) {
	return []string{accountName}, nil
}

// VerifyContractPermission verify permission of calling contract
func (f *fakeChainCore) VerifyContractPermission(initiator string, authRequire []string, contractName string, methodName string) (bool, error) {
	return true, nil
}

// VerifyContractOwnerPermission verify contract ownership permisson
func (f *fakeChainCore) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	return nil
}

func (t *fakeChainCore) QueryBlock(blockid []byte) (ledger.BlockHandle, error) {
	return state.NewBlockAgent(&xldgpb.InternalBlock{
		Blockid: []byte("testblockid"),
	}), nil
}

func (t *fakeChainCore) QueryTransaction(txid []byte) (*pb.Transaction, error) {
	return &pb.Transaction{
		Txid:    "testtxid",
		Blockid: "testblockd",
	}, nil
}
