# XuperKernel

XuperKernel定位为广域场景适用、高可扩展、超高性能、自由开放的区块链基础内核框架。基于动态内核技术，实现无内核代码侵入的自由扩展内核核心组件和轻量级的订制内核引擎。支持内核核心处理流程、核心组件技术选型、内核各组件的具体实现等多纬度的自由扩展，满足面向各类场景的区块链实现的需要，做到广域场景适用的基础区块链内核框架。并且提供了全面的、高性能的标准内核基础组件实现，全面降低区块链技术研发难度，开启一键发链新时代。

XuperKernel是XuperChain技术体系的基础内核，基于XuperKernel构建的区块链标准发行版XuperChain和XuperOS，在多行业、多场景得到了落地验证。

## 系统结构

![jiagou](https://raw.githubusercontent.com/xuperchain/xupercore/master/docs/images/jiagou.png)

## 内核框架

![kernel](https://raw.githubusercontent.com/xuperchain/xupercore/master/docs/images/kernel.png)

# 快速使用

开发链可参考基于XuperKernel实现的标准发行版XuperOS项目。详细文档还在建设中，敬请期待。

XuperKernel也提供了示例链（example/xchain）实现，初次尝试可以通过该链便捷部署测试网络体验。

```
// clone项目
git clone https://github.com/xuperchain/xupercore.git
// 进入工程目录
cd xupercore
// 编译工程
make all
// 部署测试网络
make testnet
// 分别启动三个节点（请确保使用到的端口未被占用）
cd ./testnet/node1
sh ./control.sh start
cd ../node2
sh ./control.sh start
cd ../node3
sh ./control.sh start
// 观察每个节点状态
./bin/xchain-cli status -H :36301
./bin/xchain-cli status -H :36302
./bin/xchain-cli status -H :36303

```

# 参与贡献

XuperKernel在持续建设阶段，欢迎感兴趣的同学一起参与贡献。

如果你遇到问题或需要新功能，欢迎创建issue。

如果你可以解决某个issue, 欢迎发送PR。

如项目对您有帮助，欢迎star支持。

