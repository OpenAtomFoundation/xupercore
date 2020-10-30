package config

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/common/xutils"
	"github.com/xuperchain/xupercore/lib/utils"

	"github.com/spf13/viper"
)

type P2PConfig struct {
	// port the p2p network listened
	Port int32 `yaml:"port,omitempty"`
	// keyPath is the node private key path, xuper will gen a random one if is nil
	KeyPath string `yaml:"keyPath,omitempty"`
	// isNat config whether the node use NAT manager
	IsNat bool `yaml:"isNat,omitempty"`
	// isSecure config whether the node use secure transparent
	IsSecure bool `yaml:"isSecure,omitempty"`
	// isHidden config whether the node can be found
	IsHidden bool `yaml:"isHidden,omitempty"`
	// bootNodes config the bootNodes the node to connect
	BootNodes []string `yaml:"bootNodes,omitempty"`
	// staticNodes config the nodes which you trust
	StaticNodes map[string][]string `yaml:"staticNodes,omitempty"`
	// isBroadCast config whether broadcast to all StaticNodes
	IsBroadCast bool `yaml:"isBroadCast,omitempty"`
	// maxStreamLimits config the max stream num
	MaxStreamLimits int32 `yaml:"maxStreamLimits,omitempty"`
	// maxMessageSize config the max message size
	MaxMessageSize int64 `yaml:"maxMessageSize,omitempty"`
	// timeout config the timeout of Request with response
	Timeout int64 `yaml:"timeout,omitempty"`
	// IsAuthentication determine whether peerID and Xchain addr correspond
	IsAuthentication bool `yaml:"isauthentication,omitempty"`
	// StreamIPLimitSize set the limitation size for same ip
	StreamIPLimitSize int64 `yaml:"streamIPLimitSize,omitempty"`
	// MaxBroadcastPeers limit the number of common peers in a broadcast,
	// this number do not include MaxBroadcastCorePeers.
	MaxBroadcastPeers int `yaml:"maxBroadcastPeers,omitempty"`
	// MaxBroadcastCorePeers limit the number of core peers in a broadcast,
	// this only works when NodeConfig.CoreConnection is true. Note that the number
	// of core peers is included in MaxBroadcastPeers.
	MaxBroadcastCorePeers int `yaml:"maxBroadcastCorePeers,omitempty"`
	// P2PDataPath stores the peer info connected last time
	P2PDataPath string `yaml:"p2PDataPath,omitempty"`
	// IsStorePeers determine wherther storing the peers infos
	IsStorePeers bool `yaml:"isStorePeers,omitempty"`
	// CertPath define the path of certificate
	CertPath string `yaml:"certPath,omitempty"`
	// IsUseCert define whether to use certificate, default true
	IsUseCert bool `yaml:"isUseCert,omitempty"`
	// ServiceName
	ServiceName string `yaml:"serviceName,omitempty"`
}

func LoadP2PConf(cfgFile string) (*P2PConfig, error) {
	cfg := GetDefP2PConf()
	err := cfg.loadConf(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load p2p config failed.err:%s", err)
	}

	return cfg, nil
}

func GetDefP2PConf() *P2PConfig {
	return &P2PConfig{
		Port:             47101,
		KeyPath:          "./data/netkeys/",
		IsNat:            true,
		IsSecure:         true,
		IsHidden:         false,
		MaxStreamLimits:  1024,
		MaxMessageSize:   128,
		Timeout:          3,
		IsAuthentication: false,
		// default stream ip limit size
		StreamIPLimitSize:     10,
		MaxBroadcastPeers:     20,
		MaxBroadcastCorePeers: 17,
		IsStorePeers:          false,
		P2PDataPath:           "./data/p2p",
		StaticNodes:           make(map[string][]string),
		CertPath:              "./data/cert",
		IsUseCert:             true,
		ServiceName:           "",
		IsBroadCast:           true,
	}
}

func (t *P2PConfig) loadConf(cfgFile string) error {
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

// For testing.
const (
	DefConfFile = "conf/network.yaml"
)

// For testing.
// 为了方便单测运行，设置一个环境变量X_ROOT_PATH，统一从这个目标加载配置和数据
func GetNetConfFile() string {
	var utPath = xutils.GetXRootPath()
	if utPath == "" {
		panic("X_ROOT_PATH environment variable not set")
	}
	return utPath + DefConfFile
}
