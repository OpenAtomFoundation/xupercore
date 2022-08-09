# XuperCore

开放原子超级链内核XuperCore，定位为广域适用、高可扩展、超高性能、高度易用，并且完全自由开放的区块链通用内核框架。基于自主研发的“动态内核技术”，实现无内核代码侵入的自由扩展内核核心组件和轻量级的扩展订制内核引擎，满足面向各类场景的区块链实现的需要；并且提供了全面的、高性能的标准内核组件和引擎实现。全面降低区块链研发成本，实现非常轻量级的订制满足特定场景的区块链实现。XuperCore是超级链XuperChain的基础内核，基于XuperCore构建的区块链标准发行版XuperChain，在多行业、多场景得到了落地应用验证。

## 特点

#### 1. 广域适用
 
XuperCore通过极致的可扩展性，做到“广域场景适用”的区块链技术。基于XuperCore可以多纬度的自由扩展订制，非常便捷、快速的构建起适用于各类场景的区块链。开发者可以根据自己实际场景的需要，非常自由、多纬度的扩展。可选择对部分内核组件做订制；也可以基于标准组件订制自己的区块链引擎；也可以基于标准引擎，轻量级的订制自己的区块链实现。从而满足开发者的各纬度的需要，做到广域场景适用。
 
#### 2. 高可扩展

XuperCore通过“动态内核技术”，做到了区块链内核核心流程和核心组件，都可以没有内核框架代码侵入的自由扩展替换，支持多纬度的自由扩展，让整个内核具备极好的可扩展性。通过对共识、账本、合约等这些内核组件做抽象，制定了区块链内核组件编程规范，这些规范就像主板上的插槽，起到承上启下的作用，让内核各核心组件可以自由的扩展替换，同时让引擎订制变得非常的轻量级。再在内核核心组件编程规范的基础上，设计了多引擎架构，让内核核心处理流程和技术选型也可以无内核代码侵入的自由扩展替换。

#### 3. 超高性能

XuperCore在架构上做到高可扩展的同时，还提供了超高性能的内核组件和标准区块链引擎的实现。比如：通过XuperCore提供的智能合约和账本标准组件，就可以实现智能合约的并行预执行、和并行的验证，可以很好的提升智能合约的性能。

#### 4. 高度易用

XuperCore提供了非常完备的内核标准组件实现和易用性工具，可以做到一键生成完整区块链实现，对开发者屏蔽区块链技术的核心复杂度，实现自由组装区块链系统。XuperCore还提供了区块链代码自动生成工具、编译工具、测试网络部署工具，非常轻量级就可以开发一套全新区块链系统。

## 架构

<img width="641" alt="image" src="https://user-images.githubusercontent.com/61530942/183580026-ddc19777-a731-4b66-8353-f9e8287c2317.png">

# 快速试用

## 环境配置

- 操作系统：支持Linux以及Mac OS
- 开发语言：Go 1.14.x及以上
- 编译器：GCC 4.8.x及以上
- 版本控制工具：Git

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

可参考基于XuperCore实现的标准发行版[XuperChain](https://github.com/xuperchain/xuperchain)项目。

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
./bin/xchain-cli chain status -H 127.0.0.1:36101
./bin/xchain-cli chain status -H 127.0.0.1:36102
./bin/xchain-cli chain status -H 127.0.0.1:36103

```

# 参与贡献

XuperCore在持续建设阶段，欢迎感兴趣的同学一起参与贡献，可参考[设计揭秘](https://mp.weixin.qq.com/s/pLQq_Qw8XyXJihEOXWv8Gg)。

如果你遇到问题或需要新功能，欢迎创建[issue](https://github.com/xuperchain/xupercore/issues)。

如果你可以解决某个issue, 欢迎发送PR。

如项目对您有帮助，欢迎star支持，项目源码[xupercore](https://github.com/xuperchain/xupercore)。

# 开源许可

XuperCore使用的Apache 2.0开源协议，目前已捐赠到了开放原子开源基金会。

# 联系我们

Email：xchain-help@baidu.com，如果你对XuperCore开源技术及应用感兴趣，欢迎关注"百度超级链"公众号。
