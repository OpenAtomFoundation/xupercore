package config

import (
	"fmt"

	"github.com/xuperchain/xupercore/lib/utils"

	"github.com/spf13/viper"
)

type XLedgerConf struct {
	// kv storage type
	KVEngineType string     `yaml:"kvEngineType,omitempty"`
	OtherPaths   []string   `yaml:"otherPaths,omitempty"`
	StorageType  string     `yaml:"storageType,omitempty"`
	Utxo         UtxoConfig `yaml:"utxo,omitempty"`
}

type UtxoConfig struct {
	CacheSize      int `yaml:"cachesize,omitempty"`
	TmpLockSeconds int `yaml:"tmplockSeconds,omitempty"`
}

func LoadLedgerConf(cfgFile string) (*XLedgerConf, error) {
	cfg := GetDefLedgerConf()
	err := cfg.loadConf(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load ledger config failed.err:%s", err)
	}

	return cfg, nil
}

func GetDefLedgerConf() *XLedgerConf {
	return &XLedgerConf{
		KVEngineType: "leveldb",
		OtherPaths:   nil,
		StorageType:  "",
		Utxo: UtxoConfig{
			CacheSize:      1000,
			TmpLockSeconds: 60,
		},
	}
}

func (t *XLedgerConf) loadConf(cfgFile string) error {
	if cfgFile == "" || !utils.FileIsExist(cfgFile) {
		return fmt.Errorf("config file set error.path:%s", cfgFile)
	}

	viperObj := viper.New()
	viperObj.SetConfigFile(cfgFile)
	err := viperObj.ReadInConfig()
	if err != nil {
		return fmt.Errorf("read config failed.path:%s,err:%v", cfgFile, err)
	}

	if err = viperObj.Unmarshal(t); err != nil {
		return fmt.Errorf("unmatshal config failed.path:%s,err:%v", cfgFile, err)
	}

	return nil
}
