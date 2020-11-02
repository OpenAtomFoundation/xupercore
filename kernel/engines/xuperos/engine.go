package xuperos

import (
	"fmt"
	"sync"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines"
	envconf "github.com/xuperchain/xupercore/kernel/engines/config"
	engconf "github.com/xuperchain/xupercore/kernel/engines/xuperos/config"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// xuperos执行引擎，为公链场景订制区块链引擎
type XuperOSEngine struct {
	// 引擎运行环境上下文
	engCtx *def.EngineCtx
	// 日志
	log logs.Logger
	// 链实例
	chains sync.Map
	// p2p网络事件处理
	netEvent *xnet.NetEvent
	// 依赖代理组件
	relyAgent def.RelyAgent
}

func NewXuperOSEngine() *XuperOSEngine {
	return &XuperOSEngine{}
}

// 向工厂注册自己的创建方法
func init() {
	blockchain.Register(def.BCEngineName, NewXuperOSEngine)
}

// 转换引擎句柄类型
// 对外提供类型转义方法，以接口形式对外暴露
func EngineConvert(engine interface{}) (def.Engine, error) {
	if engine == nil {
		return nil, fmt.Errorf("transfer engine type failed because param is nil")
	}

	if v, ok := resp.(def.Engine); ok {
		return v, nil
	}

	return nil, fmt.Errorf("transfer engine type failed by type assert")
}

// 初始化执行引擎环境上下文
func (t *XuperOSEngine) Init(ecfg *envconf.EnvConf) error {
	// 初始化引擎运行上下文
	engCtx, err := t.createEngCtx(ecfg)
	if err != nil {
		return fmt.Errorf("init engine failed because create engine ctx failed.err:%v", err)
	}
	t.engCtx = engCtx
	t.log = t.engCtx.GetLog()
	// 默认设置为正式实现，单元测试提供Set方法替换为mock实现
	t.relyAgent = NewRelyAgent(t)
	t.log.Trace("init engine context succ")

	// 加载区块链，初始化链上下文
	t.chains = new(sync.Map)
	err = t.loadChains()
	if err != nil {
		return fmt.Errorf("init engine failed because load chain failed.err:%v", err)
	}
	t.log.Trace("init load chain succ")

	// 初始化P2P网络事件
	netEvent, err := xnet.NewNetEvent(t)
	if err != nil {
		return fmt.Errorf("init engine failed because new net event failed.err:%v", err)
	}
	t.netEvent = netEvent
	t.log.Trace("init register subscriber network event succ")

	t.log.Trace("init engine succ")
	return nil
}

// 供单测时设置rely agent为mock agent，非并发安全
func (t *XuperOSEngine) SetRelyAgent(agent def.RelyAgent) error {
	if agent == nil {
		return fmt.Errorf("param error")
	}

	t.relyAgent = agent
	return nil
}

// 启动执行引擎
func (t *XuperOSEngine) Start() error {
	// 启动每条链的矿工

	// 启动定时任务

	// 启动P2P网络事件消费
}

// 关闭执行引擎，需要幂等
func (t *XuperOSEngine) Stop() {
	// 关闭P2Pw网络

	// 关闭定时任务

	// 关闭矿工
}

func (t *XuperOSEngine) Get(string) def.Chain {

	return nil
}

func (t *XuperOSEngine) Set(string, Chain) {

}

func (t *XuperOSEngine) GetChains() []string {

}

// 获取执行引擎环境
func (t *XuperOSEngine) GetEngineCtx() *def.EngineCtx {
	return t.engCtx
}

func (t *XuperOSEngine) CreateChain(string, data []byte) (Chain, error) {

}

// 注册并启动链
func (t *XuperOSEngine) RegisterChain(name string, js []byte) (Chain, error) {

}

// 关闭并卸载链
func (t *XuperOSEngine) UnloadChain(name string) error {

}

// 从本地存储加载链
func (t *XuperOSEngine) loadChains() error {
	ecfg := t.engCtx.GetEnvConf()
	dataDir := ecfg.GenDirAbsPath(ecfg.DataDir)

	t.log.Trace("start load chain from data dir", "dir", dataDir)

	dir, err := ioutil.ReadDir(dataDir)
	if err != nil {
		t.log.Error("load chains failed because read data dir failed", "err", err, "data_dir", dataDir)
		return fmt.Errorf("load chains failed")
	}
	chainCnt := 0
	for _, fInfo := range dir {
		if !fInfo.IsDir() {
			// 忽略非目录
			continue
		}
		chainDir := filepath.Join(dataDir, fInfo.Name())
		t.log.Trace("start load chain", "chain", fInfo.Name(), "dir", chainDir)

		// 实例化每条链
		chain, err := LoadChain(filepath.Join(dataDir, fInfo.Name()))
		if err != nil {
			t.log.Error("load chain from data dir failed.", "err", err, "dir", chainDir)
			return fmt.Errorf("load chain failed")
		}

		// 设置root链
		if fInfo.Name() == ecfg.RootChain {
			t.rootChain = chain
		}
		// 记录链实例
		t.chains.Store(fInfo.Name(), chain)
		t.log.Trace("load chain succ", "chain", fInfo.Name(), "dir", chainDir)
		chainCnt++
	}

	t.log.Trace("load chain form data dir succ", "chain_cnt", chainCnt)
	return nil
}

func (t *XuperOSEngine) createEngCtx(envCfg *envconf.EnvConf) (*def.EngineCtx, error) {
	if envCfg == nil {
		return nil, fmt.Errorf("create engine ctx failed because env config is nil")
	}

	// 加载引擎配置
	engCfg, err := engconf.LoadEngineConf(envCfg.GenConfFilePat(envCfg.EngineConf))
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because engine config load err.err:%v", err)
	}

	// 实例化网络
	netHD, err := t.relyAgent.CreateNetwork()
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because create network failed.err:%v", err)
	}

	engCtx := &def.EngineCtx{
		xctx.BaseCtx: xctx.BaseCtx{
			XLog:  logs.NewLogger("", def.BCEngineName),
			Timer: timer.NewXTimer(),
		},
		EnvCfg: envCfg,
		EngCfg: engCfg,
		Net:    netHD,
	}

	return engCtx, nil
}
