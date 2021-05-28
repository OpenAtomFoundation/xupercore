package parachain

import (
	"fmt"
	"github.com/spf13/viper"
	"strconv"
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
	if ctx.BcName != ctx.ChainCtx.EngCtx.EngCfg.RootChain {
		return nil, fmt.Errorf("Permission denied to register this contract")
	}
	conf, err := loadConfig(ctx.ChainCtx.EngCtx.EnvCfg.GenConfFilePath(ConfigName))
	if err != nil {
		return nil, err
	}

	minNewChainAmount, err := strconv.ParseInt(conf.MinNewChainAmount, 10, 64)
	if err != nil {
		return nil, err
	}
	t := NewKernContractMethod(ctx.BcName, minNewChainAmount, ctx.ChainCtx)
	register := ctx.Contract.GetKernRegistry()
	register.RegisterKernMethod(ParaChainKernelContract, "CreateBlockChain", t.CreateBlockChain)
	//todo
	//workerObj := ctx.GetAsyncWorker()
	//workerObj.RegisterHandler(ParaChainKernelContract, "CreateBlockChain", handleCreateChain)
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
