package agent

import (
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
)

type ChainCoreAgent struct {
	log      logs.Logger
	chainCtx common.ChainCtx
}

func NewChainCoreAgent(chainCtx common.ChainCtx) *ChainCoreAgent {
	return &ChainCoreAgent{
		log:      chainCtx.GetXLog(),
		chainCtx: chainCtx,
	}
}

func (t *ChainCoreAgent) BCName() string {
	return t.chainCtx.BCName
}

func (t *ChainCoreAgent) GetAccountAddresses(accountName string) ([]string, error) {
	// 查询合约acl
	return nil, nil
}

func (t *ChainCoreAgent) VerifyContractPermission(initiator string, authRequire []string,
	contractName, methodName string) (bool, error) {
	// 结合合约acl设置鉴权
	return false, nil
}

func (t *ChainCoreAgent) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	// 结合合约acl设置鉴权
	return nil
}
