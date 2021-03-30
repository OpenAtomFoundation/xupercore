package config

import (
	"fmt"
	"time"

	"github.com/xuperchain/xupercore/lib/utils"

	"github.com/spf13/viper"
)

const (
	// root链默认链名
	RootBlockChain = "xuper"
)

type EngineConf struct {
	// root chain name
	RootChain string `yaml:"rootChain,omitempty"`
	// BlockBroadcaseMode is the mode for broadcast new block
	BlockBroadcastMode uint8 `yaml:"blockBroadcastMode,omitempty"`
	// TxCacheExpiredTime expired time for tx cache
	TxIdCacheExpiredTime time.Duration `yaml:"txidCacheExpiredTime,omitempty"`
	// TxIdCacheGCInterval clean up interval for tx cache
	TxIdCacheGCInterval time.Duration `yaml:"txIdCacheGCInterval,omitempty"`
	// MaxBlockQueueSize the queue size of the processing block
	MaxBlockQueueSize int64 `yaml:"maxBlockQueueSize,omitempty"`
}

func LoadEngineConf(cfgFile string) (*EngineConf, error) {
	cfg := GetDefEngineConf()
	err := cfg.loadConf(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load engine config failed.err:%s", err)
	}

	return cfg, nil
}

func GetDefEngineConf() *EngineConf {
	return &EngineConf{
		RootChain:            RootBlockChain,
		BlockBroadcastMode:   0,
		TxIdCacheExpiredTime: 180 * time.Second,
		TxIdCacheGCInterval:  300 * time.Second,
		MaxBlockQueueSize:    100,
	}
}

func (t *EngineConf) loadConf(cfgFile string) error {
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
