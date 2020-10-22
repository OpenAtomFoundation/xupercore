package engines

import (
	"fmt"
	"path/filepath"

	"github.com/xuperchain/xupercore/lib/utils"

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
	// log config file name
	LogConf string `yaml:"logConf,omitempty"`
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
		// 默认设置为当前执行目录
		RootPath: utils.GetCurExecDir(),
		ConfDir:  "conf",
		DataDir:  "data",
		LogDir:   "logs",
		LogConf:  "log.yaml",
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

func (t *EnvConfig) GenDirAbsPath(dir string) string {
	return filepath.Join(t.RootPath, dir)
}

func (t *EnvConfig) GenConfFilePath(fName string) string {
	return filepath.Join(t.GenDirAbsPath(t.ConfDir), fName)
}
