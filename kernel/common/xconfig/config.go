package xconfig

import (
	"fmt"
	"path/filepath"

	"github.com/xuperchain/xupercore/lib/utils"

	"github.com/spf13/viper"
)

type EnvConf struct {
	// Program running root directory
	RootPath string `yaml:"rootPath,omitempty"`
	// config file directory
	ConfDir string `yaml:"confDir,omitempty"`
	// data file directory
	DataDir string `yaml:"dataDir,omitempty"`
	// log file directory
	LogDir string `yaml:"logDir,omitempty"`
	// tls file directory
	TlsDir string `yaml:"tlsDir,omitempty"`
	// engine config file name
	EngineConf string `yaml:"engineConf,omitempty"`
	// log config file name
	LogConf string `yaml:"logConf,omitempty"`
	// server config file name
	ServConf string `yaml:"servConf,omitempty"`
	// network config file name
	NetConf string `yaml:"netConf,omitempty"`
}

func LoadEnvConf(cfgFile string) (*EnvConf, error) {
	cfg := GetDefEnvConf()
	err := cfg.loadConf(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load env config failed.err:%s", err)
	}

	return cfg, nil
}

func GetDefEnvConf() *EnvConf {
	return &EnvConf{
		// 默认设置为当前执行目录
		RootPath:   utils.GetCurExecDir(),
		ConfDir:    "conf",
		DataDir:    "data",
		LogDir:     "logs",
		TlsDir:     "tls",
		EngineConf: "engine.yaml",
		LogConf:    "log.yaml",
		ServConf:   "server.yaml",
		NetConf:    "network.yaml",
	}
}

func (t *EnvConf) GenDirAbsPath(dir string) string {
	return filepath.Join(t.RootPath, dir)
}

func (t *EnvConf) GenConfFilePath(fName string) string {
	return filepath.Join(t.GenDirAbsPath(t.ConfDir), fName)
}

func (t *EnvConf) loadConf(cfgFile string) error {
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
