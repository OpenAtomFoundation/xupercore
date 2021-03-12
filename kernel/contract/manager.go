package contract

import (
	"fmt"
	"sync"

	"github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/kernel/ledger"
)

var (
	managerMutex sync.Mutex
	managers     = make(map[string]NewManagerFunc)
)

type NewManagerFunc func(cfg *ManagerConfig) (Manager, error)

type Manager interface {
	NewContext(cfg *ContextConfig) (Context, error)
	NewStateSandbox(cfg *SandboxConfig) (StateSandbox, error)
	GetKernRegistry() KernRegistry
}

type ManagerConfig struct {
	Basedir  string
	BCName   string
	EnvConf  *xconfig.EnvConf
	Core     ChainCore
	XMReader ledger.XMReader

	Config *ContractConfig // used by testing
}

// ChainCore is the interface of chain service
type ChainCore interface {
	// GetAccountAddress get addresses associated with account name
	GetAccountAddresses(accountName string) ([]string, error)
	// VerifyContractPermission verify permission of calling contract
	VerifyContractPermission(initiator string, authRequire []string, contractName, methodName string) (bool, error)
	// VerifyContractOwnerPermission verify contract ownership permisson
	VerifyContractOwnerPermission(contractName string, authRequire []string) error

	// QueryTransaction query confirmed tx
	// QueryTransaction(txid []byte) (*pb.Transaction, error)
	// QueryBlock query block
	// QueryBlock(blockid []byte) (*pb.InternalBlock, error)
	// ResolveChain resolve chain endorsorinfos
	// ResolveChain(chainName string) (*pb.CrossQueryMeta, error)
}

func Register(name string, f NewManagerFunc) {
	managerMutex.Lock()
	defer managerMutex.Unlock()

	if _, exists := managers[name]; exists {
		panic(fmt.Sprintf("contract manager of type %s exists", name))
	}
	managers[name] = f
}

func CreateManager(name string, cfg *ManagerConfig) (Manager, error) {
	mgfunc, ok := managers[name]
	if !ok {
		return nil, fmt.Errorf("contract manager of type %s not exists", name)
	}
	return mgfunc(cfg)
}
