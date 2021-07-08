package mock

import (
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
	"github.com/xuperchain/xupercore/kernel/ledger"
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
	return state.NewBlockAgent(&xldgpb.InternalBlock{}), nil
}

func (t *fakeChainCore) QueryTransaction(txid []byte) (*pb.Transaction, error) {
	return &pb.Transaction{}, nil
}
