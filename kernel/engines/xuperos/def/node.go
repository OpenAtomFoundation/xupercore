package def

import (
	"fmt"
)

type NodeAddrInfo struct {
	Address    string
	PrivateKey string
	PublicKey  string
}

func LoadAddrInfo(keyDir string) (*NodeAddrInfo, error) {
	// 从目录加载账户信息
	addr := ""
	// 生成公钥和私钥
	priKey := ""
	pubKey := ""

	addInfo := &NodeAddrInfo{
		Address:    addr,
		PrivateKey: priKey,
		PublicKey:  pubKey,
	}

	return addInfo, nil
}
