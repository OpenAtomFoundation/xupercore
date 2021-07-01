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
	return t.chainCtx.State.QueryTransaction(txid)

}

// QueryBlock query block
func (t *ChainCoreAgent) QueryBlock(blockid []byte) (ledger.BlockHandle, error) {
	block, err := t.chainCtx.State.QueryBlock(blockid)
	return block, err
	//if err != nil {
	//	return nil, err
	//}
	//return NewBlockAgent(block), nil
}
