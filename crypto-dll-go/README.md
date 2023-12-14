# Go 动态调用 Rust 密码学工具库

## 支持算法

- BLS 门限签名算法

## 使用方式

### 代码调用

```go
package pkg

import (
	crypto "github.com/OpenAtomFoundation/xupercore/crypto-dll-go"
)

func f()  {

	client := crypto.NewBlsClient()

	// create BLS account
	client.CreateAccount()
	account := client.Account

	// get peer's account
	// set group
	client.UpdateGroup(peers)

	// generate MK part for peers
	// at same time, exchange and get peer's MK part for current account
	mkPartsTo, err := client.GenerateMkParts()
	mkPartsFrom = append(mkPartsFrom, peerMkPart)
	// ...

	// update MK
	client.UpdateMk(mkParts)
	mk := client.Mk

	// a client signs message
	signPart, err := client.Sign(message)

	// exchange signature as part
	signParts[index] = signPart
	// ...

	// combine signature
	sign, err := client.CombineSignatureParts(signParts)

	// generate proof
	proof, err := client.Proof(message, sign)

	// verify signature
	ok := client.VerifySignatureByProof(sign.Signature, proof)
}
```

### 动态库引入

- Linux
	- 系统默认搜索路径
		- `/lib`
		- `/usr/lib`
		- `/usr/local/lib`
	- 动态库文件
		- `libxcrypto.so`
- Mac
	- 系统默认搜索路径
		- `/usr/lib`
		- `/usr/local/lib`
		- `/opt/local/lib`
	- 动态库文件
		- `libxcrypto.dylib`

> 如何获取动态库文件：[获取方式](#动态库获取)

## 动态库获取

> 依赖工具：`git` `cargo`

编译获取最新的动态库：

1. 克隆代码库
2. 编译动态库
    ```shell
    cd crypto-rust/x-crypto-ffi
    cargo build --release
    cd target/release
    ls | grep lib
    ```