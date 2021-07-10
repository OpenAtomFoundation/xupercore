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
	return state.NewBlockAgent(&xldgpb.InternalBlock{
		Version:              0,
		Nonce:                0,
		Blockid:              []byte("testblockid"),
		PreHash:              nil,
		Proposer:             nil,
		Sign:                 nil,
		Pubkey:               nil,
		MerkleRoot:           nil,
		Height:               0,
		Timestamp:            0,
		Transactions:         nil,
		TxCount:              0,
		MerkleTree:           nil,
		CurTerm:              0,
		CurBlockNum:          0,
		FailedTxs:            nil,
		TargetBits:           0,
		Justify:              nil,
		InTrunk:              false,
		NextHash:             nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}), nil
}

func (t *fakeChainCore) QueryTransaction(txid []byte) (*pb.Transaction, error) {
	return &pb.Transaction{
		Txid:                 "testtxid",
		Blockid:              "testblockd",
		TxInputs:             nil,
		TxOutputs:            nil,
		Desc:                 nil,
		Initiator:            "",
		AuthRequire:          nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}, nil
}
