package config

import (
	"fmt"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"time"

	"github.com/xuperchain/xupercore/lib/utils"

	"github.com/spf13/viper"
)

const (
	// NodeModeNormal NODE_MODE_NORMAL node mode for normal
	NodeModeNormal = "Normal"
	// NodeModeFastSync NODE_MODE_FAST_SYNC node mode for fast
	NodeModeFastSync = "FastSync"

	DefaultNetModule = "p2pv2"
	DefaultKeyPath   = "data/keys" // node key path
	DefaultFailSkip  = false
)

type EngineConf struct {
	// root chain name
	RootChain string `yaml:"rootChain,omitempty"`
	// netDisk plugin name
	NetModule string      `yaml:"netModule,omitempty"`
	Miner     MinerConfig `yaml:"miner,omitempty"`
	// 扩展盘的路径
	DataPathOthers []string `yaml:"datapathOthers,omitempty"`

	// 节点模式: NORMAL | FAST_SYNC 两种模式
	// NORMAL: 为普通的全节点模式
	// FAST_SYNC 模式下:节点需要连接一个可信的全节点; 拒绝事务提交; 同步区块时跳过块验证和tx验证; 去掉load未确认事务;
	NodeMode string `yaml:"nodeMode,omitempty"`

	FailSkip bool `yaml:"failSkip,omitempty"`

	// BlockBroadcaseMode is the mode for broadcast new block
	//  * Full_BroadCast_Mode = 0, means send full block data
	//  * Interactive_BroadCast_Mode = 1, means send block id and the receiver get block data by itself
	//  * Mixed_BroadCast_Mode = 2, means miner use Full_BroadCast_Mode, other nodes use Interactive_BroadCast_Mode
	//  1. 一种是完全块广播模式(Full_BroadCast_Mode)，即直接广播原始块给所有相邻节点;
	//  2. 一种是问询式块广播模式(Interactive_BroadCast_Mode)，即先广播新块的头部给相邻节点，
	//     相邻节点在没有相同块的情况下通过GetBlock主动获取块数据.
	//  3. Mixed_BroadCast_Mode是指出块节点将新块用Full_BroadCast_Mode模式广播，其他节点使用Interactive_BroadCast_Mode
	BlockBroadcastMode uint8 `yaml:"blockBroadcastMode,omitempty"`

	// TxCacheExpiredTime expired time for tx cache
	TxIdCacheExpiredTime time.Duration `yaml:"txidCacheExpiredTime,omitempty"`
}

// MinerConfig is the config of miner
type MinerConfig struct {
	KeyPath string `yaml:"keypath,omitempty"`
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
		RootChain: def.RootChain,
		NetModule: DefaultNetModule,
		Miner: MinerConfig{
			KeyPath: DefaultKeyPath,
		},
		NodeMode:           NodeModeNormal,
		FailSkip:           DefaultFailSkip,
		BlockBroadcastMode: def.FullBroadCastMode,
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
