package xuperos

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/kernel/engines"
	engconf "github.com/xuperchain/xupercore/kernel/engines/xuperos/config"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	xnet "github.com/xuperchain/xupercore/kernel/engines/xuperos/net"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// xuperos执行引擎，为公链场景订制区块链引擎
// 采用多链架构，支持多链，考虑到面向公链场景，暂时不支持群组
type XuperOSEngine struct {
	// 引擎运行环境上下文
	engCtx *def.EngineCtx
	// 日志
	log logs.Logger
	// 链实例
	chains *sync.Map
	// p2p网络事件处理
	netEvent *xnet.NetEvent
	// 依赖代理组件
	relyAgent def.EngineRelyAgent
	// 管理异步任务退出状态
	exitWG sync.WaitGroup
}

func NewXuperOSEngine() engines.BCEngine {
	return &XuperOSEngine{}
}

// 向工厂注册自己的创建方法
func init() {
	engines.Register(def.BCEngineName, NewXuperOSEngine)
}

// 转换引擎句柄类型
// 对外提供类型转义方法，以接口形式对外暴露
func EngineConvert(engine engines.BCEngine) (def.Engine, error) {
	if engine == nil {
		return nil, fmt.Errorf("transfer engine type failed because param is nil")
	}

	if v, ok := engine.(def.Engine); ok {
		return v, nil
	}

	return nil, fmt.Errorf("transfer engine type failed by type assert")
}

// 初始化执行引擎环境上下文
func (t *XuperOSEngine) Init(ecfg *xconf.EnvConf) error {
	// 初始化引擎运行上下文
	engCtx, err := t.createEngCtx(ecfg)
	if err != nil {
		return fmt.Errorf("init engine failed because create engine ctx failed.err:%v", err)
	}
	t.engCtx = engCtx
	t.log = t.engCtx.XLog
	// 默认设置为正式实现，单元测试提供Set方法替换为mock实现
	t.relyAgent = NewEngineRelyAgent(t)
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
func (t *XuperOSEngine) SetRelyAgent(agent def.EngineRelyAgent) error {
	if agent == nil {
		return fmt.Errorf("param error")
	}

	t.relyAgent = agent
	return nil
}

// 启动执行引擎，阻塞等待
func (t *XuperOSEngine) Start() {
	// 遍历启动每条链
	t.chains.Range(func(k, v interface{}) bool {
		chainHD := v.(def.Chain)
		t.log.Trace("start chain " + k.(string))

		t.exitWG.Add(1)
		go func() {
			defer t.exitWG.Done()

			// 启动链
			chainHD.Start()
			t.log.Trace("chain " + k.(string) + "start")
		}()

		return true
	})

	// 启动P2P网络事件消费
	t.exitWG.Add(1)
	go func() {
		defer t.exitWG.Done()
		t.netEvent.Start()
	}()

	// 阻塞等待，直到所有异步任务成功退出
	t.exitWG.Wait()
}

// 关闭执行引擎，需要幂等
func (t *XuperOSEngine) Stop() {
	// 关闭P2P网络
	t.netEvent.Stop()

	// 关闭网络事件处理循环

	// 关闭矿工
	t.chains.Range(func(k, v interface{}) bool {
		chainHD := v.(def.Chain)
		t.log.Trace("stop chain " + k.(string))

		t.exitWG.Add(1)
		go func() {
			defer t.exitWG.Done()

			// 关闭链
			chainHD.Stop()
			t.log.Trace("chain " + k.(string) + "exit")
		}()

		return true
	})

	t.exitWG.Wait()
}

func (t *XuperOSEngine) Get(name string) def.Chain {
	if chain, ok := t.chains.Load(name); ok {
		return chain.(def.Chain)
	}

	return nil
}

func (t *XuperOSEngine) Set(name string, chain def.Chain) {
	t.chains.Store(name, chain)
	return
}

func (t *XuperOSEngine) GetChains() []string {
	chains := make([]string, 0)
	t.chains.Range(func(k, v interface{}) bool {
		chains = append(chains, k.(string))
		return true
	})
	return chains
}

// 获取执行引擎环境
func (t *XuperOSEngine) GetEngineCtx() *def.EngineCtx {
	return t.engCtx
}

func (t *XuperOSEngine) CreateChain(name string, data []byte) (def.Chain, error) {
	if _, ok := t.chains.Load(name); ok {
		t.log.Warn("chains[" + name + "] is exist")
		return nil, fmt.Errorf("blockchain is exist already")
	}

	chain := new(XuperOSChain)
	// TODO: create a new block chain from xuper.json

	t.chains.Store(name, chain)
	return chain, nil
}

// 注册并启动链
func (t *XuperOSEngine) RegisterChain(name string) error {
	chain, ok := t.chains.Load(name)
	if !ok {
		return fmt.Errorf("blockchain is not exist")
	}

	// 启动链
	go chain.(def.Chain).Start()
	t.log.Trace("chain " + name + "start")

	return nil
}

// 关闭并卸载链
func (t *XuperOSEngine) UnloadChain(name string) error {
	v, ok := t.chains.Load(name)
	if !ok {
		return fmt.Errorf("blockchain is not exist")
	}
	//从engine的map里面删了，就不会收到新的请求了
	t.chains.Delete(name)

	//然后停止这个链
	v.(def.Chain).Stop()

	return nil
}

// 从本地存储加载链
func (t *XuperOSEngine) loadChains() error {
	ecfg := t.engCtx.EnvCfg
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

		// 记录链实例
		t.chains.Store(fInfo.Name(), chain)
		t.log.Trace("load chain succ", "chain", fInfo.Name(), "dir", chainDir)
		chainCnt++
	}

	t.log.Trace("load chain form data dir succ", "chain_cnt", chainCnt)
	return nil
}

func (t *XuperOSEngine) createEngCtx(envCfg *xconf.EnvConf) (*def.EngineCtx, error) {
	if envCfg == nil {
		return nil, fmt.Errorf("create engine ctx failed because env config is nil")
	}

	// 加载引擎配置
	engCfg, err := engconf.LoadEngineConf(envCfg.GenConfFilePath(envCfg.EngineConf))
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because engine config load err.err:%v", err)
	}

	// 实例化&启动p2p网络
	netHD, err := t.relyAgent.CreateNetwork()
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because create network failed.err:%v", err)
	}

	engCtx := &def.EngineCtx{}
	engCtx.XLog, _ = logs.NewLogger("", def.BCEngineName)
	engCtx.Timer = timer.NewXTimer()
	engCtx.EnvCfg = envCfg
	engCtx.EngCfg = engCfg
	engCtx.Net = netHD

	return engCtx, nil
}
