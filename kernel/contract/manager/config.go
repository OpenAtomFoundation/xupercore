package manager

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/xuperchain/xupercore/kernel/contract"
)

const (
	contractConfigName = "contract.yaml"
)

func loadConfig(fname string) (*contract.ContractConfig, error) {
	viperObj := viper.New()
	viperObj.SetConfigFile(fname)
	err := viperObj.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("read config failed.path:%s,err:%v", fname, err)
	}

	cfg := contract.DefaultContractConfig()
	if err = viperObj.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmatshal config failed.path:%s,err:%v", fname, err)
	}
	return cfg, nil
}
