package xuperos

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/kernel/engines"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/agent"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	engconf "github.com/xuperchain/xupercore/kernel/engines/xuperos/config"
	xnet "github.com/xuperchain/xupercore/kernel/engines/xuperos/net"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// xuperos执行引擎，为公链场景订制区块链引擎
type Engine struct {
	// 引擎运行环境上下文
	engCtx *common.EngineCtx
	// 日志
	log logs.Logger
	// 链实例
	chains sync.Map
	// p2p网络事件处理
	netEvent *xnet.NetEvent
	// 依赖代理组件
	relyAgent common.EngineRelyAgent
	// 确保Exit调用幂等
	exitOnce sync.Once
}

// 向工厂注册自己的创建方法
func init() {
	engines.Register(common.BCEngineName, NewEngine)
}

func NewEngine() engines.BCEngine {
	return &Engine{}
}

// 转换引擎句柄类型
// 对外提供类型转义方法，以接口形式对外暴露
func EngineConvert(engine engines.BCEngine) (common.Engine, error) {
	if engine == nil {
		return nil, common.ErrParameter
	}

	if v, ok := engine.(common.Engine); ok {
		return v, nil
	}

	return nil, common.ErrNotEngineType
}

// 初始化执行引擎环境上下文
func (t *Engine) Init(envCfg *xconf.EnvConf) error {
	if envCfg == nil {
		return common.ErrParameter
	}

	// 单元测试提供Set方法替换为mock实现
	t.relyAgent = agent.NewEngineRelyAgent(t)

	// 初始化引擎运行上下文
	engCtx, err := t.createEngCtx(envCfg)
	if err != nil {
		return common.ErrNewEngineCtxFailed.More("%v", err)
	}
	t.engCtx = engCtx
	t.log = t.engCtx.XLog
	t.log.Trace("init engine context succeeded")

	// 加载区块链，初始化链上下文
	err = t.loadChains()
	if err != nil {
		t.log.Error("load chain failed", "err", err)
		return common.ErrLoadChainFailed
	}
	t.log.Trace("load all chain succeeded")

	// 初始化P2P网络事件
	netEvent, err := xnet.NewNetEvent(t)
	if err != nil {
		t.log.Error("new net event failed", "err", err)
		return common.ErrNewNetEventFailed
	}
	t.netEvent = netEvent
	t.log.Trace("init register subscriber network event succeeded")

	t.log.Trace("init engine succeeded")
	return nil
}

// 供单测时设置rely agent为mock agent，非并发安全
func (t *Engine) SetRelyAgent(agent common.EngineRelyAgent) error {
	if agent == nil {
		return common.ErrParameter
	}

	t.relyAgent = agent
	return nil
}

// 启动执行引擎，阻塞等待
func (t *Engine) Run() {
	wg := &sync.WaitGroup{}

	// 启动P2P网络
	t.engCtx.Net.Start()

	// 启动P2P网络事件消费
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.netEvent.Start()
	}()

	// 遍历启动每条链
	t.chains.Range(func(k, v interface{}) bool {
		chainHD := v.(common.Chain)
		t.log.Trace("start chain " + k.(string))

		wg.Add(1)
		go func() {
			defer wg.Done()

			t.log.Trace("run chain " + k.(string))
			// 启动链
			chainHD.Start()
			t.log.Trace("chain " + k.(string) + " exit")
		}()

		return true
	})

	// 阻塞等待，直到所有异步任务成功退出
	wg.Wait()
}

// 关闭执行引擎，需要幂等
func (t *Engine) Exit() {
	t.exitOnce.Do(func() {
		t.exit()
	})
}

func (t *Engine) Get(name string) (common.Chain, error) {
	if chain, ok := t.chains.Load(name); ok {
		return chain.(common.Chain), nil
	}

	return nil, common.ErrChainNotExist
}

// 获取执行引擎环境
func (t *Engine) Context() *common.EngineCtx {
	return t.engCtx
}

func (t *Engine) GetChains() []string {
	chains := make([]string, 0)
	t.chains.Range(func(k, v interface{}) bool {
		chains = append(chains, k.(string))
		return true
	})
	return chains
}

// 从本地存储加载链
func (t *Engine) loadChains() error {
	envCfg := t.engCtx.EnvCfg
	dataDir := envCfg.GenDataAbsPath(envCfg.ChainDir)

	t.log.Trace("start load chain from blockchain data dir", "dir", dataDir)
	dir, err := ioutil.ReadDir(dataDir)
	if err != nil {
		t.log.Error("read blockchain data dir failed", "error", err, "dir", dataDir)
		return fmt.Errorf("read blockchain data dir failed")
	}

	chainCnt := 0
	rootChain := t.engCtx.EngCfg.RootChain
	for _, fInfo := range dir {
		if !fInfo.IsDir() {
			continue
		}

		chainDir := filepath.Join(dataDir, fInfo.Name())
		t.log.Trace("start load chain", "chain", fInfo.Name(), "dir", chainDir)

		// 实例化每条链
		chain, err := LoadChain(t.engCtx, fInfo.Name())
		if err != nil {
			t.log.Error("load chain from data dir failed", "error", err, "dir", chainDir)
			return fmt.Errorf("load chain failed")
		}
		t.log.Trace("load chain from data dir succ", "chain", fInfo.Name())

		// 记录链实例
		t.chains.Store(fInfo.Name(), chain)
		t.log.Trace("load chain succeeded", "chain", fInfo.Name(), "dir", chainDir)
		chainCnt++
	}

	// root链必须存在
	if _, ok := t.chains.Load(rootChain); !ok {
		t.log.Error("root chain not exist, please create it first", "rootChain", rootChain)
		return fmt.Errorf("root chain not exist")
	}

	t.log.Trace("load chain form data dir succeeded", "chainCnt", chainCnt)
	return nil
}

func (t *Engine) createEngCtx(envCfg *xconf.EnvConf) (*common.EngineCtx, error) {
	// 引擎日志
	log, err := logs.NewLogger("", common.BCEngineName)
	if err != nil {
		return nil, fmt.Errorf("new logger failed.err:%v", err)
	}

	// 加载引擎配置
	engCfg, err := engconf.LoadEngineConf(envCfg.GenConfFilePath(envCfg.EngineConf))
	if err != nil {
		return nil, fmt.Errorf("load engine config failed.err:%v", err)
	}

	engCtx := &common.EngineCtx{}
	engCtx.XLog = log
	engCtx.Timer = timer.NewXTimer()
	engCtx.EnvCfg = envCfg
	engCtx.EngCfg = engCfg

	// 实例化p2p网络
	engCtx.Net, err = t.relyAgent.CreateNetwork(envCfg)
	if err != nil {
		return nil, fmt.Errorf("create network failed.err:%v", err)
	}

	return engCtx, nil
}

func (t *Engine) exit() {
	// 关闭矿工
	wg := &sync.WaitGroup{}
	t.chains.Range(func(k, v interface{}) bool {
		chainHD := v.(common.Chain)

		t.log.Trace("stop chain " + k.(string))
		wg.Add(1)
		// 关闭链
		chainHD.Stop()
		wg.Done()
		t.log.Trace("chain " + k.(string) + " closed")

		return true
	})

	// 关闭P2P网络
	wg.Add(1)
	t.engCtx.Net.Stop()
	wg.Done()

	// 关闭网络事件处理循环
	wg.Add(1)
	t.netEvent.Stop()
	wg.Done()

	// 等待全部退出完成
	wg.Wait()
}

// RegisterBlockChain load an instance of blockchain and start it dynamically
func (t *Engine) RegisterBlockChain(name string) error {
	chain, err := LoadChain(t.engCtx, name)
	if err != nil {
		t.log.Error("load chain failed", "error", err, "chain_name", name)
		return fmt.Errorf("load chain failed")
	}
	t.chains.Store(name, chain)
	go chain.Start()
	return nil
}
