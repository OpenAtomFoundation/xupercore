# XuperCore Auto Generate

基于XuperCore动态内核自动生成的区块链发行版。

# 快速使用

## 编译环境

- Linux or Mac OS
- Go 1.13
- git、make、gcc、curl、unzip

## 部署测试网络

提供了测试网络部署工具，初次尝试可以通过该工具便捷部署测试网络。

```
// clone项目
git clone https://github.com/xxx/xxx.git

// 进入工程目录
cd xxx

// 编译工程
make all

// 部署测试网络
sh ./auto/deploy_testnet.sh

// 分别启动三个节点（确保端口未被占用）
cd ./testnet/node1
sh ./control.sh start
cd ../node2
sh ./control.sh start
cd ../node3
sh ./control.sh start
// 观察每个节点状态
./bin/xxx-cli status -H :36301
./bin/xxx-cli status -H :36302
./bin/xxx-cli status -H :36303

```

测试网络搭建完成，开启您的区块链之旅！

# 参与贡献

https://github.com/xuperchain/xupercore

如果你遇到问题或需要新功能，欢迎到xupercore创建issue。

如果你可以解决某个issue, 欢迎发送PR。

