package parachain

import (
	"fmt"
	"strconv"

	"github.com/spf13/viper"
	"github.com/xuperchain/xupercore/kernel/contract"
)

const (
	ConfigName = "engine.yaml"
)

type ParaChainConfig struct {
	MinNewChainAmount string
}

// Manager
type Manager struct {
	Ctx *ParaChainCtx
}

// NewParaChainManager create instance of ParaChain
func NewParaChainManager(ctx *ParaChainCtx) (*Manager, error) {
	if ctx == nil || ctx.Contract == nil || ctx.BcName == "" {
		return nil, fmt.Errorf("parachain ctx set error")
	}
	conf, err := loadConfig(ctx.ChainCtx.EngCtx.EnvCfg.GenConfFilePath(ConfigName))
	if err != nil {
		return nil, err
	}

	minNewChainAmount, err := strconv.ParseInt(conf.MinNewChainAmount, 10, 64)
	if err != nil {
		return nil, err
	}
	t := NewParaChainContract(ctx.BcName, minNewChainAmount, ctx.ChainCtx)
	register := ctx.Contract.GetKernRegistry()
	// 注册合约方法
	kMethods := map[string]contract.KernMethod{
		"createChain": t.createChain,
		"editGroup":   t.editGroup,
		"getGroup":    t.getGroup,
	}
	for method, f := range kMethods {
		if _, err := register.GetKernMethod(ParaChainKernelContract, method); err != nil {
			register.RegisterKernMethod(ParaChainKernelContract, method, f)
		}
	}
	// 仅主链绑定handleCreateChain 从链上下文中获取链绑定的异步任务worker
	ctx.ChainCtx.Asyncworker.RegisterHandler(ParaChainKernelContract, "CreateBlockChain", handleCreateChain)
	mg := &Manager{
		Ctx: ctx,
	}

	return mg, nil
}

func loadConfig(fname string) (*ParaChainConfig, error) {
	viperObj := viper.New()
	viperObj.SetConfigFile(fname)
	err := viperObj.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("read config failed.path:%s,err:%v", fname, err)
	}

	cfg := &ParaChainConfig{
		MinNewChainAmount: "100",
	}
	if err = viperObj.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmatshal config failed.path:%s,err:%v", fname, err)
	}
	return cfg, nil
}
