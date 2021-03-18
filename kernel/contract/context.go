package contract

const (
	// StatusOK is used when contract successfully ends.
	StatusOK = 200
	// StatusErrorThreshold is the status dividing line for the normal operation of the contract
	StatusErrorThreshold = 400
	// StatusError is used when contract fails.
	StatusError = 500
)

// Context define context interface
type Context interface {
	Invoke(method string, args map[string][]byte) (*Response, error)
	ResourceUsed() Limits
	Release() error
}

// Response is the result of the contract run
type Response struct {
	// Status 用于反映合约的运行结果的错误码
	Status int `json:"status"`
	// Message 用于携带一些有用的debug信息
	Message string `json:"message"`
	// Data 字段用于存储合约执行的结果
	Body []byte `json:"body"`
}

// ContextConfig define the config of context
type ContextConfig struct {
	State StateSandbox

	Initiator   string
	AuthRequire []string

	Caller string

	Module       string
	ContractName string

	ResourceLimits Limits

	// Whether contract can be initialized
	CanInitialize bool

	// The amount transfer to contract
	TransferAmount string

	// Contract being called
	// set by bridge to check recursive contract call
	ContractSet map[string]bool

	// ContractCodeFromCache control whether fetch contract code from XMCache
	ContractCodeFromCache bool
}
