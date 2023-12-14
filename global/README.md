# Yogurt 信标链

---

## Yogurt 信标链是什么?

Yogurt 信标链主要任务是为 Yogurt 的执行层提供随机数，支持执行层的共识运行。 

- 共识算法：TDPoS

## 快速试用

### 环境配置

* 操作系统：支持Linux以及Mac OS
* 开发语言：Go 1.9.*及以上
* 编译器：GCC 4.8.x及以上
* 版本控制工具：Git

### 构建

编译
```
make
```

跑单测
```
make test
```

单机版
```
cd ./output
sh ./control.sh start
./bin/ychain-cli status
```

多节点ychain

生成多节点。
在运行下面的命令之前，请确保已经运行`make`去编译代码。
```
make testnet
```

进入testnet目录，分别启动三个节点(确保端口未被占用)。
```
cd ./testnet/node1
sh ./control.sh start
cd ../node2
sh ./control.sh start
cd ../node3
sh ./control.sh start
```

观察每个节点状态
```
./bin/ychain-cli status -H :37101
./bin/ychain-cli status -H :37102
./bin/ychain-cli status -H :37103
```