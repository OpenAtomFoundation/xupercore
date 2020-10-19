package engines

import (
	"fmt"
	"path/filepath"

	"github.com/xuperchain/xupercore/kernel/common/utils"

	"github.com/spf13/viper"
)

type EnvConfig struct {
	// Program running root directory
	RootPath string `yaml:"rootPath,omitempty"`
	// config file directory
	ConfDir string `yaml:"confDir,omitempty"`
	// data file directory
	DataDir string `yaml:"dataDir,omitempty"`
	// log file directory
	LogDir string `yaml:"logDir,omitempty"`
}

func LoadEnvConf(cfgFile string) (*EnvConfig, error) {
	cfg := GetDefEnvConf()
	err := cfg.loadConf(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load env config failed.err:%s", err)
	}

	return cfg, nil
}

func GetDefEnvConf() *EnvConfig {
	return &EnvConfig{
		RootPath: "./",
		ConfDir:  "conf",
		DataDir:  "data",
		LogDir:   "logs",
	}
}

func (t *EnvConfig) loadConf(cfgFile string) error {
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

func (t *EnvConfig) GenAbsPath(dir string) string {
	return filepath.Join(t.RootPath, dir)
}

// For testing.
const (
	DefConfFile = "conf/env.yaml"
)

// For testing.
func GetEnvConfFile() string {
	var utPath = utils.GetXchainRootPath()
	if utPath == "" {
		panic("XCHAIN_ROOT_PATH environment variable not set")
	}

	return filepath.Join(utPath, DefConfFile)
}
