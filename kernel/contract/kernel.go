package contract

type KernRegistry interface {
	RegisterKernMethod(contract, method string, handler KernMethod)
	GetKernMethod(contract, method string) (KernMethod, error)
}

type KernMethod func(ctx KContext) (*Response, error)

type KContext interface {
	// 交易相关数据
	Args() map[string][]byte
	Initiator() string
	AuthRequire() []string

	// 状态修改接口
	StateSandbox
	ChainCore

	AddResourceUsed(delta Limits)
	ResourceLimit() Limits

	Call(module, contract, method string, args map[string][]byte) (*Response, error)
}
