package logs

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/common/utils"

	"github.com/spf13/viper"
)

// LogConfig is the log config of node
type LogConfig struct {
	Module   string `yaml:"module,omitempty"`
	Filepath string `yaml:"filepath,omitempty"`
	Filename string `yaml:"filename,omitempty"`
	Fmt      string `yaml:"fmt,omitempty"`
	Console  bool   `yaml:"console,omitempty"`
	Level    string `yaml:"level,omitempty"`
	Async    bool   `yaml:"async,omitempty"`
	// 日志分割周期（单位：分钟）
	RotateInterval int `yaml:"rotateinterval,omitempty"`
	// 日志保留天数（单位：小时）
	RotateBackups int `yaml:"rotatebackups,omitempty"`
}

func LoadLogConf(cfgFile string) (*LogConfig, error) {
	cfg := GetDefLogConf()
	err := cfg.loadConf(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load p2p config failed.err:%s", err)
	}

	return cfg, nil
}

func GetDefLogConf() *LogConfig {
	return &LogConfig{
		Module:   "xchain",
		Filepath: "logs",
		Filename: "xchain",
		Fmt:      "logfmt",
		Console:  true,
		Level:    "debug",
		Async:    false,
		// rotate every 60 minutes
		RotateInterval: 60,
		// keep old log files for 7 days
		RotateBackups: 168,
	}
}

func (t *LogConfig) loadConf(cfgFile string) error {
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
