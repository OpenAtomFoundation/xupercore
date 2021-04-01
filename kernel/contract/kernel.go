package contract

type KernRegistry interface {
	RegisterKernMethod(contract, method string, handler KernMethod)
	// RegisterShortcut 用于contractName缺失的时候选择哪个合约名字和合约方法来执行对应的kernel合约
	RegisterShortcut(oldmethod, contract, method string)
	GetKernMethod(contract, method string) (KernMethod, error)
}

type KernMethod func(ctx KContext) (*Response, error)

type KContext interface {
	// 交易相关数据
	Args() map[string][]byte
	Initiator() string
	Caller() string
	AuthRequire() []string

	// 状态修改接口
	StateSandbox

	AddResourceUsed(delta Limits)
	ResourceLimit() Limits

	Call(module, contract, method string, args map[string][]byte) (*Response, error)
}
