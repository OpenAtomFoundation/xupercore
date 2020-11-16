package xuperos

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
)

// chainCore is the implement of contract.ChainCore
type chainCore struct {
}

// BCName get blockchain name
func (c *chainCore) BCName() string {
	return ""
}

// GetAccountAddress get addresses associated with account name
func (c *chainCore) GetAccountAddresses(accountName string) ([]string, error) {
	return []string{}, nil
}

// VerifyContractPermission verify permission of calling contract
func (c *chainCore) VerifyContractPermission(initiator string, authRequire []string, contractName, methodName string) (bool, error) {
	return true, nil
}

// VerifyContractOwnerPermission verify contract ownership permisson
func (c *chainCore) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	return nil
}

func RegisterKernMethod() {
	kernel.RegisterKernMethod("kernel", "CreateChain", CreateChain)
	kernel.RegisterKernMethod("kernel", "UnloadChain", UnloadChain)
}

func CreateChain(ctx kernel.KContext) (*contract.Response, error) {

}

func UnloadChain(ctx kernel.KContext) (*contract.Response, error) {

}
