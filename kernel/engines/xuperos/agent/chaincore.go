package agent

import (
	"encoding/hex"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/logs"
	"math/big"
)

type ChainCoreAgent struct {
	log      logs.Logger
	chainCtx *common.ChainCtx
}

func NewChainCoreAgent(chainCtx *common.ChainCtx) *ChainCoreAgent {
	return &ChainCoreAgent{
		log:      chainCtx.GetLog(),
		chainCtx: chainCtx,
	}
}

// 查询合约acl
func (t *ChainCoreAgent) GetAccountAddresses(accountName string) ([]string, error) {
	return t.chainCtx.Acl.GetAccountAddresses(accountName)
}

// 结合合约acl设置鉴权
func (t *ChainCoreAgent) VerifyContractPermission(initiator string, authRequire []string, contractName, methodName string) (bool, error) {
	return t.chainCtx.State.VerifyContractPermission(initiator, authRequire, contractName, methodName)
}

// 结合合约acl设置鉴权
func (t *ChainCoreAgent) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	return t.chainCtx.State.VerifyContractOwnerPermission(contractName, authRequire)
}

// QueryTransaction query confirmed tx
func (t *ChainCoreAgent) QueryTransaction(txid []byte) (*pb.Transaction, error) {
	lpb, err := t.chainCtx.State.QueryTransaction(txid)
	if err != nil {
		return nil, err
	}
	txInputs := []*pb.TxInput{}
	for _, input := range lpb.TxInputs {
		txInputs = append(txInputs, &pb.TxInput{
			RefTxid:              hex.EncodeToString(input.GetRefTxid()),
			RefOffset:            input.GetRefOffset(),
			FromAddr:             input.GetFromAddr(),
			Amount:               new(big.Int).SetBytes(input.GetAmount()).String(),
			FrozenHeight:         0,
			XXX_NoUnkeyedLiteral: struct{}{},
			XXX_unrecognized:     nil,
			XXX_sizecache:        0,
		})
	}
	txOutputs := []*pb.TxOutput{}
	for _, output := range lpb.TxOutputs {
		txOutputs = append(txOutputs, &pb.TxOutput{
			Amount:               new(big.Int).SetBytes(output.Amount).String(),
			ToAddr:               output.GetToAddr(),
			FrozenHeight:         output.GetFrozenHeight(),
			XXX_NoUnkeyedLiteral: struct{}{},
			XXX_unrecognized:     nil,
			XXX_sizecache:        0,
		})
	}
	return &pb.Transaction{
		Txid:                 hex.EncodeToString(lpb.Txid),
		Blockid:              hex.EncodeToString(lpb.Blockid),
		TxInputs:             txInputs,
		TxOutputs:            txOutputs,
		Desc:                 lpb.GetDesc(),
		Initiator:            lpb.GetInitiator(),
		AuthRequire:          lpb.GetAuthRequire(),
		XXX_NoUnkeyedLiteral: lpb.XXX_NoUnkeyedLiteral,
		XXX_unrecognized:     lpb.XXX_unrecognized,
		XXX_sizecache:        lpb.XXX_sizecache,
	}, nil

}

// QueryBlock query block
func (t *ChainCoreAgent) QueryBlock(blockid []byte) (ledger.BlockHandle, error) {
	block, err := t.chainCtx.State.QueryBlock(blockid)
	if err != nil {
		return nil, err
	}
	return block, err
	//return state.NewBlockAgent(block), nil
}
