# 区块链引擎

区块链引擎：定义一种区块链核心流程实现。

内核框架采用多引擎架构，每个引擎订制一套区块链内核实现，所有引擎注册到引擎工厂，外部通过工厂实例化引擎。
每个引擎提供执行引擎和读组件两部分能力。各引擎间交易、区块结构无关，共用内核核心组件。

## 引擎介绍

xuperos: 面向公链场景区块链网络内核实现。

xchain: 面向联盟联盟场景区块链网络内核实现。

## 使用示例

```

// 加载内核运行环境配置
envCfgPath := "/home/rd/xx/conf/env.yaml"
envCfg, _ := engines.LoadEnvConf(envCfgPath)

// 创建内核引擎实例
engine, _ := engines.CreateBCEngine("xchain", envCfg)

engine.Init()
engine.Start()
engine.Stop()

xEngine, _ := xuperos.EngineConvert(engine)
xChain := xEngine.Get("xuper")
xChain.PreExec()
xChain.ProcessTx()
```
