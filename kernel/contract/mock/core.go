package mock

import (
	"math/big"

	"github.com/xuperchain/xupercore/protos"
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

func (f *fakeChainCore) Transfer(from string, to string, amount *big.Int) error {
	return nil
}

func (f *fakeChainCore) SelectUtxos(string, *big.Int, bool, bool) ([]*protos.TxInput, [][]byte, *big.Int, error) {
	return nil, nil, nil, nil
}
