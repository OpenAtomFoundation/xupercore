# XuperCore

XuperCore定位为广域场景适用、高可扩展、超高性能、自由开放的区块链基础内核框架。

基于动态内核技术，实现无内核代码侵入的自由扩展内核核心组件和轻量级的扩展订制内核引擎，满足面向各类场景的区块链实现的需要；提供了全面的、高性能的标准内核组件实现。全面降低区块链研发成本，开启一键发链新时代。

XuperCore是XuperChain技术体系的基础内核，基于XuperCore构建的区块链标准发行版XuperChain和XuperOS，在多行业、多场景得到了落地验证。

# 快速使用

## 十分钟构建自己的区块链，并搭建测试网络

XuperCore提供了自动生成区块链发行版代码框架的工具，可以通过该工具一键生成发行版代码。

```
// clone项目
git clone https://github.com/xuperchain/xupercore.git

// 生成发行版代码框架，第一个参数是新链名，第二个参数是输出代码库保存目录
sh ./tools/autogen_chain.sh -n turbo -r bob -o /home/rd/gopath/src

// 去新生成的代码仓库做订制升级，完了编译
cd /home/rd/gopath/src/github.com/xuperchain/turbo
make all

// 部署测试网络，对网络做测试验证
sh ./auto/deploy_testnet.sh

// 进入testnet目录逐节点启动
sh ./control.sh start

```

可参考基于XuperCore实现的面向公开网络的发行版XuperOS项目。
https://github.com/xuperchain/xuperos

## 部署测试网络

XuperCore也提供了示例链（example/xchain）实现，初次尝试可以通过该链便捷部署测试网络体验。

```
// clone项目
git clone https://github.com/xuperchain/xupercore.git

// 进入工程目录
cd xupercore

// 编译工程
make all

// 部署测试网络
sh ./tools/deploy_testnet.sh

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

