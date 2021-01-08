package engines

import (
	"fmt"
	"sort"
	"sync"

	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/lib/logs"
)

// 区块链引擎
// 考虑到引擎的可扩展性，只约束最基本接口，具体暴露接口可由各引擎实现个性化扩展
// 各引擎提供类型转换函数，由上层引擎使用者调用引擎提供的类型转换函数转换后使用
type BCEngine interface {
	// 初始化引擎
	Init(*xconf.EnvConf) error
	// 启动引擎(阻塞)
	Run()
	// 退出引擎，需要幂等
	Exit()
}

// 创建engine实例方法
type NewBCEngineFunc func() BCEngine

var (
	engineMu sync.RWMutex
	engines  = make(map[string]NewBCEngineFunc)
)

func Register(name string, f NewBCEngineFunc) {
	engineMu.Lock()
	defer engineMu.Unlock()

	if f == nil {
		panic("network: Register new func is nil")
	}
	if _, dup := engines[name]; dup {
		panic("network: Register called twice for func " + name)
	}
	engines[name] = f
}

func Engines() []string {
	engineMu.RLock()
	defer engineMu.RUnlock()
	list := make([]string, 0, len(engines))
	for name := range engines {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func newBCEngine(name string) BCEngine {
	engineMu.RLock()
	defer engineMu.RUnlock()

	if f, ok := engines[name]; ok {
		return f()
	}

	return nil
}

// 采用工厂模式，对上层统一区块链执行引擎创建操作，方便框架开发
// 区块链执行引擎注册通过init实现，由应用方选择具体要使用的引擎
func CreateBCEngine(egName string, envCfg *xconf.EnvConf) (BCEngine, error) {
	// 检查参数
	if egName == "" || envCfg == nil {
		return nil, fmt.Errorf("create bc engine failed because some param unset")
	}

	// 初始化日志实例，失败会panic，日志初始化操作是幂等的
	logs.InitLog(envCfg.GenConfFilePath(envCfg.LogConf), envCfg.GenDirAbsPath(envCfg.LogDir))

	// 创建区块链执行引擎
	engine := newBCEngine(egName)
	if engine == nil {
		return nil, fmt.Errorf("create bc engine failed because engine not exist. name:%s", egName)
	}

	// 初始化区块链执行引擎
	err := engine.Init(envCfg)
	if err != nil {
		return nil, fmt.Errorf("init engine error: %v", err)
	}

	return engine, nil
}
