package xuperos

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/chunhui01/xupercore/kernel/engines/xuperos/commom"
	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/kernel/engines"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/agent"
	engconf "github.com/xuperchain/xupercore/kernel/engines/xuperos/config"
	xnet "github.com/xuperchain/xupercore/kernel/engines/xuperos/net"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// 向工厂注册自己的创建方法
func init() {
	engines.Register(def.BCEngineName, NewEngine)
}

// xuperos执行引擎，为公链场景订制区块链引擎
type Engine struct {
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

var _ def.Engine = &Engine{}

func NewEngine() engines.BCEngine {
	return &Engine{}
}

// 转换引擎句柄类型
// 对外提供类型转义方法，以接口形式对外暴露
func EngineConvert(engine engines.BCEngine) (def.Engine, error) {
	if engine == nil {
		return nil, fmt.Errorf("transfer engine type failed due to param is nil")
	}

	if v, ok := engine.(def.Engine); ok {
		return v, nil
	}

	return nil, fmt.Errorf("transfer engine type failed by type assert")
}

// 初始化执行引擎环境上下文
func (t *Engine) Init(envCfg *xconf.EnvConf) error {
	// 单元测试提供Set方法替换为mock实现
	t.relyAgent = agent.NewEngineRelyAgent(t)

	// 初始化引擎运行上下文
	engCtx, err := t.createEngCtx(envCfg)
	if err != nil {
		return fmt.Errorf("create engine ctx failed: %v", err)
	}
	t.engCtx = engCtx
	t.log = t.engCtx.XLog

	t.log.Trace("init engine context succeeded")

	// 加载区块链，初始化链上下文
	t.chains = new(sync.Map)
	err = t.loadChains()
	if err != nil {
		return fmt.Errorf("load chain failed: %v", err)
	}
	t.log.Trace("load all chain succeeded")

	// 初始化P2P网络事件
	netEvent, err := xnet.NewNetEvent(t)
	if err != nil {
		return fmt.Errorf("new net event failed: %v", err)
	}
	t.netEvent = netEvent
	t.log.Trace("init register subscriber network event succeeded")

	t.log.Trace("init engine succeeded")
	return nil
}

// 供单测时设置rely agent为mock agent，非并发安全
func (t *Engine) SetRelyAgent(agent def.EngineRelyAgent) error {
	if agent == nil {
		return ErrParamError
	}

	t.relyAgent = agent
	return nil
}

// 启动执行引擎，阻塞等待
func (t *Engine) Start() {
	// 启动P2P网络
	t.engCtx.Net.Start()

	// 启动P2P网络事件消费
	t.exitWG.Add(1)
	go func() {
		defer t.exitWG.Done()
		t.netEvent.Start()
	}()

	// 遍历启动每条链
	t.chains.Range(func(k, v interface{}) bool {
		chainHD := v.(def.Chain)
		t.log.Trace("start chain " + k.(string))

		t.exitWG.Add(1)
		go func() {
			defer t.exitWG.Done()

			// 启动链
			chainHD.Start()
			t.log.Trace("chain " + k.(string) + "started")
		}()

		return true
	})

	// 阻塞等待，直到所有异步任务成功退出
	t.exitWG.Wait()
}

// 关闭执行引擎，需要幂等
func (t *Engine) Stop() {
	// 关闭矿工
	t.chains.Range(func(k, v interface{}) bool {
		chainHD := v.(def.Chain)
		t.log.Trace("stop chain " + k.(string))

		t.exitWG.Add(1)
		go func() {
			defer t.exitWG.Done()

			// 关闭链
			chainHD.Stop()
			t.log.Trace("chain " + k.(string) + "closed")
		}()

		return true
	})

	// 关闭网络事件处理循环
	t.netEvent.Stop()

	// 关闭P2P网络
	t.engCtx.Net.Stop()

	t.exitWG.Wait()
}

func (t *Engine) Get(name string) def.Chain {
	if chain, ok := t.chains.Load(name); ok {
		return chain.(def.Chain)
	}

	return nil
}

func (t *Engine) Set(name string, chain def.Chain) {
	t.chains.Store(name, chain)
	return
}

func (t *Engine) GetChains() []string {
	chains := make([]string, 0)
	t.chains.Range(func(k, v interface{}) bool {
		chains = append(chains, k.(string))
		return true
	})
	return chains
}

// 获取执行引擎环境
func (t *Engine) Context() *def.EngineCtx {
	return t.engCtx
}

func (t *Engine) CreateChain(name string, data []byte) error {
	if _, ok := t.chains.Load(name); ok {
		t.log.Warn("chains[" + name + "] is exist")
		return ErrBlockChainExist
	}

	t.log.Debug("create block chain by contract", "name", name)
	// TODO: 1.仅xuper可以创建平行链
	//if k.bcName != "xuper" {
	//	t.log.Warn("only xuper chain can create side-chain", "bcName", k.bcName)
	//	return ErrPermissionDenied
	//}

	dataPath := t.engCtx.EnvCfg.GenDirAbsPath(t.engCtx.EnvCfg.DataDir)
	fullPath := filepath.Join(dataPath, name)
	return CreateBlockChain(fullPath, name, data)
}

// 注册并启动链
func (t *Engine) RegisterChain(name string) error {
	chain, ok := t.chains.Load(name)
	if !ok {
		return ErrBlockChainNotExist
	}

	// 启动链
	go chain.(def.Chain).Start()
	t.log.Trace("chain " + name + "start")

	return nil
}

// 关闭并卸载链
func (t *Engine) UnloadChain(name string) error {
	v, ok := t.chains.Load(name)
	if !ok {
		return ErrBlockChainNotExist
	}
	//从engine的map里面删了，就不会收到新的请求了
	t.chains.Delete(name)

	//然后停止这个链
	v.(def.Chain).Stop()

	return nil
}

// 从本地存储加载链
func (t *Engine) loadChains() error {
	envCfg := t.engCtx.EnvCfg
	dataDir := envCfg.GenDataAbsPath(envCfg.ChainDir)

	t.log.Trace("start load chain from blockchain data dir.", "dir", dataDir)

	dir, err := ioutil.ReadDir(dataDir)
	if err != nil {
		t.log.Error("read blockchain data dir failed.", "error", err, "dir", dataDir)
		return fmt.Errorf("load chain failed because read blockchain data dir error")
	}

	chainCnt := 0
	for _, fInfo := range dir {
		if !fInfo.IsDir() {
			// 忽略非目录
			continue
		}

		chainDir := filepath.Join(dataDir, fInfo.Name())
		t.log.Trace("start load chain.", "chain", fInfo.Name(), "dir", chainDir)

		// 实例化每条链
		chainCtx := &commom.ChainCtx{
			EngCtx: t.engCtx,
			BCName: fInfo.Name(),
		}
		chain, err := LoadChain(chainCtx)
		if err != nil {
			t.log.Error("load chain from data dir failed", "error", err, "dir", chainDir)
			return ErrLoadChainError
		}

		// 记录链实例
		t.chains.Store(fInfo.Name(), chain)
		t.log.Trace("load chain succeeded", "chain", fInfo.Name(), "dir", chainDir)
		chainCnt++
	}

	rootChain := t.Context().EngCfg.RootChain
	if _, ok := t.chains.Load(rootChain); !ok {
		t.log.Error("root chain not exist, please create it first", "rootChain", rootChain, "error", err)
		return ErrRootChainNotExist
	}

	t.log.Trace("load chain form data dir succeeded", "chain_cnt", chainCnt)
	return nil
}

func (t *Engine) createEngCtx(envCfg *xconf.EnvConf) (*commom.EngineCtx, error) {
	if envCfg == nil {
		return nil, ErrParamError
	}

	// 引擎日志
	log, err := logs.NewLogger("", commom.BCEngineName)
	if err != nil {
		return nil, fmt.Errorf("new logger failed.err: %v", err)
	}

	// 加载引擎配置
	engCfg, err := engconf.LoadEngineConf(envCfg.GenConfFilePath(envCfg.EngineConf))
	if err != nil {
		return nil, fmt.Errorf("load engine config error: %v", err)
	}

	engCtx := &commom.EngineCtx{}
	engCtx.XLog = log
	engCtx.Timer = timer.NewXTimer()
	engCtx.EnvCfg = envCfg
	engCtx.EngCfg = engCfg

	// 实例化&启动p2p网络
	engCtx.Net, err = t.relyAgent.CreateNetwork()
	if err != nil {
		return nil, fmt.Errorf("create network failed: %v", err)
	}

	return engCtx, nil
}
