package manager

import (
	"fmt"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
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
	if err = viperObj.Unmarshal(&cfg, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
	}); err != nil {
		return nil, fmt.Errorf("unmatshal config failed.path:%s,err:%v", fname, err)
	}
	return cfg, nil
}
