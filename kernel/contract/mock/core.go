package mock

import (
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
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

func (t *fakeChainCore) QueryBlock(blockid []byte) (*xldgpb.InternalBlock, error) {
	return &xldgpb.InternalBlock{}, nil
}

func (t *fakeChainCore) QueryTransaction(txid []byte) (*pb.Transaction, error) {

	return &pb.Transaction{}, nil
}
