package xuperos

import (
    "fmt"
    "testing"

    // import要使用的内核核心组件驱动
    _ "github.com/xuperchain/xupercore/bcs/consensus/pow"
    _ "github.com/xuperchain/xupercore/bcs/consensus/single"
    _ "github.com/xuperchain/xupercore/bcs/consensus/tdpos"
    _ "github.com/xuperchain/xupercore/bcs/consensus/xpoa"
    _ "github.com/xuperchain/xupercore/bcs/contract/native"
    _ "github.com/xuperchain/xupercore/bcs/contract/xvm"
    _ "github.com/xuperchain/xupercore/bcs/network/p2pv1"
    _ "github.com/xuperchain/xupercore/bcs/network/p2pv2"
    _ "github.com/xuperchain/xupercore/kernel/contract/kernel"
    _ "github.com/xuperchain/xupercore/kernel/contract/manager"
    _ "github.com/xuperchain/xupercore/lib/crypto/client"
    _ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"

    "github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
    "github.com/xuperchain/xupercore/kernel/mock"
)

func mockEngine(path string) (common.Engine, error) {
    conf, err := mock.NewEnvConfForTest(path)
    if err != nil {
        return nil, fmt.Errorf("new env conf error: %v", err)
    }

    engine := NewEngine()
    if err := engine.Init(conf); err != nil {
        return nil, fmt.Errorf("init engine error: %v", err)
    }

    eng, err := EngineConvert(engine)
    if err != nil {
        return nil, fmt.Errorf("engine convert error: %v", err)
    }

    go engine.Run()
    return eng, nil
}

func TestEngine(t *testing.T) {
    engine, err := mockEngine("p2pv2/node1/conf/env.yaml")
    if err != nil {
        t.Logf("%v", err)
        return
    }

    engine.Exit()
}