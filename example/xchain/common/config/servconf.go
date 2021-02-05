package config

import (
	"fmt"

	"github.com/xuperchain/xupercore/lib/utils"

	"github.com/spf13/viper"
)

type ServConf struct {
	// rpc server listen port
	RpcPort            int   `yaml:"rpcPort,omitempty"`
	MaxRecvMsgSize     int   `yaml:"maxRecvMsgSize,omitempty"`
	ReadBufSize        int   `yaml:"readBufSize,omitempty"`
	WriteBufSize       int   `yaml:"writeBufSize,omitempty"`
	InitWindowSize     int32 `yaml:"initWindowSize,omitempty"`
	InitConnWindowSize int32 `yaml:"initConnWindowSize,omitempty"`
}

func LoadServConf(cfgFile string) (*ServConf, error) {
	cfg := GetDefServConf()
	err := cfg.loadConf(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load server config failed.err:%s", err)
	}

	return cfg, nil
}

func GetDefServConf() *ServConf {
	return &ServConf{
		RpcPort:            38101,
		MaxRecvMsgSize:     128 << 20,
		ReadBufSize:        32 << 10,
		WriteBufSize:       32 << 10,
		InitWindowSize:     128 << 10,
		InitConnWindowSize: 64 << 10,
	}
}

func (t *ServConf) loadConf(cfgFile string) error {
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
