# XuperCore Global

XuperCore Global 除了作为 global 动态内核，还支持 BLS 算法作为信标链网络共识的安全保证。

## 编译部署

### 环境准备

* **golang 环境 go1.20.10 及以上**
* **gcc   g++  4.8.5 及以上**
* **rust 1.63.0**

### 编译

#### 执行层

https://github.com/xuperchain/yogurt-chain?tab=readme-ov-file#%E6%89%A7%E8%A1%8C%E5%B1%82

#### 信标链层

信标链层作为网络的共识基础层，需要编译 BLS：

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

编译 ychain：https://github.com/xuperchain/yogurt-chain


