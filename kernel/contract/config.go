package contract

// NativeConfig contains the two above config
type NativeConfig struct {
	Driver string
	// Timeout (in seconds) to stop native code process
	StopTimeout int
	Docker      NativeDockerConfig
	Enable      bool
}

func (n *NativeConfig) DriverName() string {
	if n.Driver != "" {
		return n.Driver
	}

	return "native"
}

func (n *NativeConfig) IsEnable() bool {
	return n.Enable
}

// NativeDockerConfig native contract use docker config
type NativeDockerConfig struct {
	Enable    bool
	ImageName string
	Cpus      float32
	Memory    string
}

// XVMConfig contains the xvm configuration
type XVMConfig struct {
	// From 0 to 3
	// The higher the number, the faster the program runs,
	// but the compilation speed will be slower
	OptLevel int `yaml:"optlevel"`
}

// WasmConfig wasm config
type WasmConfig struct {
	Enable bool
	Driver string
	XVM    XVMConfig
}

func (w *WasmConfig) DriverName() string {
	return w.Driver
}

func (w *WasmConfig) IsEnable() bool {
	return w.Enable
}

type EVMConfig struct {
	Enable bool
	Driver string
}

func (e *EVMConfig) DriverName() string {
	return e.Driver
}

func (e *EVMConfig) IsEnable() bool {
	return e.Enable
}

type XkernelConfig struct {
	Enable bool
	Driver string

	Registry KernRegistry
}

func (x *XkernelConfig) DriverName() string {
	return x.Driver
}

func (x *XkernelConfig) IsEnable() bool {
	return x.Enable
}

// LogConfig is the log config of node
type LogConfig struct {
	Module         string `yaml:"module,omitempty"`
	Filepath       string `yaml:"filepath,omitempty"`
	Filename       string `yaml:"filename,omitempty"`
	Fmt            string `yaml:"fmt,omitempty"`
	Console        bool   `yaml:"console,omitempty"`
	Level          string `yaml:"level,omitempty"`
	Async          bool   `yaml:"async,omitempty"`
	RotateInterval int    `yaml:"rotateinterval,omitempty"`
	RotateBackups  int    `yaml:"rotatebackups,omitempty"`
}

// ContractConfig define the config of XuperBridge
type ContractConfig struct {
	EnableDebugLog bool
	DebugLog       LogConfig
	EnableUpgrade  bool

	Native  NativeConfig
	Wasm    WasmConfig
	Xkernel XkernelConfig
	EVM     EVMConfig
}

func DefaultContractConfig() *ContractConfig {
	return &ContractConfig{
		EnableDebugLog: true,
		EnableUpgrade:  true,
		Native: NativeConfig{
			Enable: true,
			Driver: "native",
		},
		Wasm: WasmConfig{
			Enable: true,
			Driver: "xvm",
		},
		Xkernel: XkernelConfig{
			Enable: true,
			Driver: "default",
		},
		EVM: EVMConfig{
			Enable: true,
			Driver: "evm",
		},
	}
}
