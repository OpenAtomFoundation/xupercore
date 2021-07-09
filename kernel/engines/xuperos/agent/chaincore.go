package agent

import (
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/logs"
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
	ltx, err := t.chainCtx.State.QueryTransaction(txid)
	if err != nil {
		return nil, err
	}
	txInputs := []*pb.TxInput{}
	txOutputs := []*pb.TxOutput{}
	for _, input := range ltx.TxInputs {
		txInputs = append(txInputs, &pb.TxInput{
			RefTxid:              input.GetRefTxid(),
			RefOffset:            input.RefOffset,
			FromAddr:             input.FromAddr,
			Amount:               input.GetAmount(),
			FrozenHeight:         input.FrozenHeight,
			XXX_NoUnkeyedLiteral: input.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:     input.XXX_unrecognized,
			XXX_sizecache:        input.XXX_sizecache,
		})
	}
	for _, output := range ltx.TxOutputs {
		txOutputs = append(txOutputs, &pb.TxOutput{
			Amount:               output.GetAmount(),
			ToAddr:               output.ToAddr,
			FrozenHeight:         output.FrozenHeight,
			XXX_NoUnkeyedLiteral: output.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:     output.XXX_unrecognized,
			XXX_sizecache:        output.XXX_sizecache,
		})
	}

	tx := &pb.Transaction{
		Txid:                 ltx.Txid,
		Blockid:              ltx.Blockid,
		TxInputs:             txInputs,
		TxOutputs:            txOutputs,
		Desc:                 ltx.Desc,
		Initiator:            ltx.Initiator,
		AuthRequire:          ltx.AuthRequire,
		XXX_NoUnkeyedLiteral: ltx.XXX_NoUnkeyedLiteral,
		XXX_unrecognized:     ltx.XXX_unrecognized,
		XXX_sizecache:        ltx.XXX_sizecache,
	}

	return tx, err
}

// QueryBlock query block
func (t *ChainCoreAgent) QueryBlock(blockid []byte) (ledger.BlockHandle, error) {
	block, err := t.chainCtx.State.QueryBlock(blockid)
	if err != nil {
		return nil, err
	}
	return block, err
}
