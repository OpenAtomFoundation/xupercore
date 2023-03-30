package xuperos

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/kernel/engines"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/agent"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/asyncworker"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	engconf "github.com/xuperchain/xupercore/kernel/engines/xuperos/config"
	xnet "github.com/xuperchain/xupercore/kernel/engines/xuperos/net"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/parachain"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/lib/timer"
)

// xuperos执行引擎，为公链场景订制区块链引擎
type Engine struct {
	// 引擎运行环境上下文
	engCtx *common.EngineCtx
	// 日志
	log logs.Logger
	// 链管理成员
	chainM ChainManagerImpl
	// p2p网络事件处理
	netEvent *xnet.Event
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
	t.chainM = ChainManagerImpl{
		engCtx: engCtx,
		log:    t.log,
	}
	t.engCtx.ChainM = &t.chainM
	t.log = t.engCtx.XLog
	t.log.Trace("init engine context succeeded")

	// 加载区块链，初始化链上下文
	err = t.loadChains()
	if err != nil {
		t.log.Error("load chain failed", "err", err)
		return err
	}
	t.log.Trace("load all chain succeeded")

	// 初始化P2P网络事件
	netEvent, err := xnet.NewEvent(t)
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
	t.chainM.StartChains()

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
	if chain, err := t.chainM.Get(name); err == nil {
		return chain, nil
	}

	return nil, common.ErrChainNotExist
}

func (t *Engine) Stop(name string) error {
	return t.chainM.Stop(name)
}

// 获取执行引擎环境
func (t *Engine) Context() *common.EngineCtx {
	return t.engCtx
}

func (t *Engine) GetChains() []string {
	return t.chainM.GetChains()
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

	// 优先加载主链
	for _, fInfo := range dir {
		if !fInfo.IsDir() || fInfo.Name() != rootChain {
			continue
		}
		chainDir := filepath.Join(dataDir, fInfo.Name())
		t.log.Trace("start load chain", "chain", fInfo.Name(), "dir", chainDir)
		chain, err := LoadChain(t.engCtx, fInfo.Name())
		if err != nil {
			t.log.Error("load chain from data dir failed", "error", err, "dir", chainDir)
			return err
		}
		t.log.Trace("load chain from data dir succ", "chain", fInfo.Name())

		// 记录链实例
		t.chainM.Put(fInfo.Name(), chain)

		// 启动异步任务worker
		if fInfo.Name() == rootChain {
			aw, err := asyncworker.NewAsyncWorkerImpl(fInfo.Name(), t, chain.ctx.State.GetLDB())
			if err != nil {
				t.log.Error("create asyncworker error", "bcName", rootChain, "err", err)
				return err
			}
			chain.ctx.Asyncworker = aw
			err = chain.CreateParaChain()
			if err != nil {
				t.log.Error("create parachain mgmt error", "bcName", rootChain, "err", err)
				return fmt.Errorf("create parachain error")
			}
			if err = aw.Start(); err != nil {
				return err
			}
		}

		t.log.Trace("load chain succeeded", "chain", fInfo.Name(), "dir", chainDir)
		chainCnt++
	}

	// root链必须存在
	rootChainHandle, err := t.chainM.Get(rootChain)
	if err != nil {
		t.log.Error("root chain not exist, please create it first", "rootChain", rootChain)
		return fmt.Errorf("root chain not exist")
	}
	rootChainReader, err := rootChainHandle.Context().State.GetTipXMSnapshotReader()
	if err != nil {
		t.log.Error("root chain get tip reader failed", "err", err.Error())
		return err
	}
	// 加载平行链
	for _, fInfo := range dir {
		if !fInfo.IsDir() || fInfo.Name() == rootChain {
			continue
		}

		// 通过主链的平行链账本状态，确认是否可以加载该平行链
		group, err := parachain.GetParaChainGroup(rootChainReader, fInfo.Name())
		if err != nil {
			t.log.Error("get para chain group failed", "chain", fInfo.Name(), "err", err.Error())
			if !kvdb.ErrNotFound(err) {
				continue
			}
			return err
		}

		if !group.IsParaChainEnable() {
			t.log.Debug("para chain stopped", "chain", fInfo.Name())
			continue
		}

		chainDir := filepath.Join(dataDir, fInfo.Name())
		t.log.Trace("start load chain", "chain", fInfo.Name(), "dir", chainDir)
		chain, err := LoadChain(t.engCtx, fInfo.Name())
		if err != nil {
			t.log.Error("load chain from data dir failed", "error", err, "dir", chainDir)
			// 平行链加载失败时可以忽略直接跳过运行
			continue
		}
		t.log.Trace("load chain from data dir succ", "chain", fInfo.Name())

		// 记录链实例
		t.chainM.Put(fInfo.Name(), chain)

		t.log.Trace("load chain succeeded", "chain", fInfo.Name(), "dir", chainDir)
		chainCnt++
	}

	t.log.Trace("load chain from data dir succeeded", "chainCnt", chainCnt)
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
	t.chainM.StopChains()

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

// LoadChain load an instance of blockchain and start it dynamically
func (t *Engine) LoadChain(name string) error {
	return t.chainM.LoadChain(name)
}
