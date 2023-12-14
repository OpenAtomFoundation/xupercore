# Yogurt

Yogurt 分为三个部分，BLS、信标链和执行层链。使用时需要根据需要编译所需部分。

## 编译部署

### 环境准备

* **golang 环境 go1.20.10 及以上**
* **gcc   g++  4.8.5 及以上**
* **rust 1.63.0**

### 编译

#### 执行层

https://github.com/xuperchain/yogurt-chain?tab=readme-ov-file#%E6%89%A7%E8%A1%8C%E5%B1%82

#### 信标链层

信标链层作为 Yogurt 网络的共识基础层，需要编译 BLS 以及 ychain：

BLS 动态链接库编译：

```shell
cd crypto-rust/x-crypto-ffi
go mod tidy
make
```

编译成功后将在 crypto-rust/x-crypto-ffi/lib 下看到 libxcrypto.so 或 libxcrypto.dylib，将此文件拷贝到 /usr/local/lib 目录下。

linux 下可能需要配置动态链接库路径（根据本地环境按需配置）

示例：

```bash
vim /etc/ld.so.conf.d/libc.conf
```

添加如下内容：

```bash
# libc default configuration
/usr/local/lib 
```

推出 vim 后执行：

```bash
ldconfig
```

检查是否生效：

```bash
ldconfig -p  |grep 'libxcrypto.so'

# 成功应看到类似如下输出
# libxcrypto.so (libc6,x86-64) => /usr/local/lib/libxcrypto.so
```

然后编译 ychain

```shell
cd ychain
go mod tidy
make
```

成功后应看到类似如下内容：

```bash
tree ./output/bin/
./output/bin/
├── wasm2c
├── ychain
└── ychain-cli
```



### 部署测试网络

**以下示例为本地测试网络搭建，如果部署正式环境需重新生成节点的公私钥、账户地址等。**

整个网络包括信标链网络以及执行层网络，信标链网络有多个 ychain 节点组成，执行层网络由多个 geth 网络组成，同时两个网络之间需要配置相关公钥、地址等。

#### 部署信标链测试网络

执行如下命令生成三个测试网络节点：

```bash
make testnet
```

应看到如下内容：

```bash
tree ./testnet/ -L 2

./testnet/
├── control_all.sh
├── node1
│   ├── bin
│   ├── conf
│   ├── control.sh
│   └── data
├── node2
│   ├── bin
│   ├── conf
│   ├── control.sh
│   └── data
└── node3
    ├── bin
    ├── conf
    ├── control.sh
    └── data
```

在node1目录生成所需账户：

```
cd testnet/node1
./bin/ychain-cli ethAccount./bin/ychain-cli ethAccount generate
```

输出内容类似如下：

```bash
Saved private key to file:  ./data/keys/eth.account
Address: 0x6Fa4322CfF52265b3049e823C388797Ef1308cb9
Public key: 041d7519bca41832866aa501215e6fc0a6cd4153c0cfe54a91468a446df246763a12941fa75c9dc4e974c2670e45deddcb8b9702c44dcf5812bc934a823fd21b20
Compress public key: 0x021d7519bca41832866aa501215e6fc0a6cd4153c0cfe54a91468a446df246763a
```

记录好输出信息，Compress public key 后面会用到。

将生成的 eth.account 拷贝到 node2 和 node3 目录下：

```
# 在node1目录下执行
cp ./data/keys/eth.account ../node2/data/keys/
cp ./data/keys/eth.account ../node3/data/keys/
```

启动三个节点：

```bash
# 回到 testnet 目录
cd ..

# 启动网络
sh control_all.sh start

# 查看网络状态
./bin/ychain-cli  status -H :37101
```

#### 部署执行层网络

执行层网络基于 geth，其他使用方式请参考：https://geth.ethereum.org/docs

先进入 geth 代码库目录下：

```
# 先从 testnet 目录退出后进入 geth 目录
cd ../../geth/build/
```

geth 网络搭建可以参考 Clique 共识网络搭建：https://geth.ethereum.org/docs/fundamentals/private-network

不同点在于创世文件需改成如下：

```json
{
	"config": {
		"chainId": 88681,
		"homesteadBlock": 0,
		"eip150Block": 0,
		"eip155Block": 0,
		"eip158Block": 0,
		"byzantiumBlock": 0,
		"constantinopleBlock": 0,
		"petersburgBlock": 0,
		"istanbulBlock": 0,
		"berlinBlock": 0,
		"yogurt": {
			"period": 5,
			"epoch": 30000,
			"beaconURL": "127.0.0.1:37101",
			"beaconPubKey": "0x03af2e5f5d303c40c14abe4dd7640a3d72cd2626d4416d16980bf137b8007798e3"
		}
	}, 
  "difficulty": "1",
	"gasLimit": "8000000",
	"extradata": "0x00000000000000000000000000000000000000000000000000000000000000005594CffB7Ce2fbeA1e1A780dcD1171f4b260912Df6dbE76a457a90Dd5fB5e99338B0Bd2fc09141550000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	"alloc": {
		"5594CffB7Ce2fbeA1e1A780dcD1171f4b260912D": {
			"balance": "900000000000000000000000"
		},
		"f6dbE76a457a90Dd5fB5e99338B0Bd2fc0914155": {
			"balance": "800000000000000000000000"
		}
	}
}
```

和 Clique 不同配置说明：

* yogurt.beaconURL 为执行层节点地址，例如：127.0.0.1:37101
* yogurt.beaconPubKey 为执行层公钥，即在node1 目录使用 `./bin/ychain-cli ethAccount./bin/ychain-cli ethAccount generate ` 生成的公钥，每次生成的都不一样，需要根据本地生成为主，将 Compress public key 配置到此处。

其他步骤参考 https://geth.ethereum.org/docs/fundamentals/private-network 

#### 信标链配置执行层节点

执行层启动前，需要将执行层节点矿工的地址配置到信标链网络，因此在搭建执行此网络时，需要记录矿工的地址。

然后执行如下命令：

```
# 在测试网络node1目录下执行，yogurt-chain/ychain/testnet/node1 目录
./bin/ychain-cli xkernel invoke 'XRandom' --method AddNode -a '{"node_address":"0x5594CffB7Ce2fbeA1e1A780dcD1171f4b260912D"}'  -H :37101
# 其中 0x5594CffB7Ce2fbeA1e1A780dcD1171f4b260912D 修改为你的执行层网络矿工地址，这里只是示例。
# 如果有多个节点需多次执行此命令

# 查看结果
./bin/ychain-cli xkernel query 'XRandom' --method QueryAccessList   -H :37101
contract response: ["0x5594CffB7Ce2fbeA1e1A780dcD1171f4b260912D","0xf6dbE76a457a90Dd5fB5e99338B0Bd2fc0914155"]
```

执行成功后，启动执行层 geth 节点即可。

转账、部署合约、调用合约等交易都需要发送到执行层网络，可以使用 metamask、remix 等工具操作。

