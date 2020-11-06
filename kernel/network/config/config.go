package config

import (
    "fmt"
    "github.com/spf13/viper"
    utils "github.com/xuperchain/xupercore/kernel/common/xutils"
)

// default settings
const (
    DefaultNetPort          = 47101             // p2p port
    DefaultNetKeyPath       = "./data/netkeys" 	// node private key path
    DefaultNetIsNat         = true  			// use NAT
    DefaultNetIsTls         = false             // use tls secure transport
    DefaultNetIsHidden      = false
    DefaultNetIsIpv6        = false
    DefaultMaxStreamLimits  = 1024
    DefaultMaxMessageSize   = 128
    DefaultTimeout          = 3
    DefaultStreamIPLimitSize     = 10
    DefaultMaxBroadcastPeers     = 20
    DefaultIsStorePeers          = false
    DefaultP2PDataPath           = "./data/p2p"
    DefaultP2PModuleName         = "p2pv2"
    DefaultServiceName           = "localhost"
    DefaultIsBroadCast           = true
)

// Config is the config of p2p server. Attention, config of dht are not expose
type Config struct {
    // Module is the name of p2p module plugin
    Module string `yaml:"module,omitempty"`
    // port the p2p network listened
    Port int32 `yaml:"port,omitempty"`
    // keyPath is the node private key path, xuper will gen a random one if is nil
    KeyPath string `yaml:"keyPath,omitempty"`
    // isNat config whether the node use NAT manager
    IsNat bool `yaml:"isNat,omitempty"`
    // isHidden config whether the node can be found
    IsHidden bool `yaml:"isHidden,omitempty"`
    // IsIpv6 config whether the node use ipv6
    IsIpv6 bool `yaml:"isIpv6,omitempty"`
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
    // StreamIPLimitSize set the limitation size for same ip
    StreamIPLimitSize int64 `yaml:"streamIPLimitSize,omitempty"`
    // MaxBroadcastPeers limit the number of common peers in a broadcast,
    // this number do not include MaxBroadcastCorePeers.
    MaxBroadcastPeers int `yaml:"maxBroadcastPeers,omitempty"`
    // IsStorePeers determine wherther storing the peers infos
    IsStorePeers bool `yaml:"isStorePeers,omitempty"`
    // P2PDataPath stores the peer info connected last time
    P2PDataPath string `yaml:"p2PDataPath,omitempty"`
    // isTls config the node use tls secure transparent
    IsTls bool `yaml:"isTls,omitempty"`
    // ServiceName
    ServiceName string `yaml:"serviceName,omitempty"`
}

func LoadP2PConf(cfgFile string) (*Config, error) {
    cfg := GetDefP2PConf()
    err := cfg.loadConf(cfgFile)
    if err != nil {
        return nil, fmt.Errorf("load p2p config failed.err:%s", err)
    }

    return cfg, nil
}

func GetDefP2PConf() *Config {
    return &Config{
        Module:           DefaultP2PModuleName,
        Port:             DefaultNetPort,
        KeyPath:          DefaultNetKeyPath,
        IsNat:            DefaultNetIsNat,
        IsTls:            DefaultNetIsTls,
        IsHidden:         DefaultNetIsHidden,
        IsIpv6:           DefaultNetIsIpv6,
        MaxStreamLimits:  DefaultMaxStreamLimits,
        MaxMessageSize:   DefaultMaxMessageSize,
        Timeout:          DefaultTimeout,
        // default stream ip limit size
        StreamIPLimitSize:     DefaultStreamIPLimitSize,
        MaxBroadcastPeers:     DefaultMaxBroadcastPeers,
        IsStorePeers:          DefaultIsStorePeers,
        P2PDataPath:           DefaultP2PDataPath,
        StaticNodes:           make(map[string][]string),
        ServiceName:           DefaultServiceName,
        IsBroadCast:           DefaultIsBroadCast,
    }
}

func (t *Config) loadConf(cfgFile string) error {
    if cfgFile == "" {
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
// 为了方便单测运行，设置一个环境变量XCHAIN_UT_PATH，统一从这个目标加载配置和数据
func GetNetConfFile() string {
    var utPath = utils.GetXRootPath()
    if utPath == "" {
        panic("XCHAIN_ROOT_PATH environment variable not set")
    }
    return utPath + DefConfFile
}
