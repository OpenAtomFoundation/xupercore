package global

import (
	"crypto/ecdsa"

	"github.com/xuperchain/xupercore/kernel/common/xaddress"
	cryptoClient "github.com/xuperchain/xupercore/lib/crypto/client"
)

// 从本地文件加载账户地址信息
func LoadAccount(cryptoType string, keyPath string) (*xaddress.Address, error) {
	crypto, err := cryptoClient.CreateCryptoClient(cryptoType)
	if err != nil {
		return nil, err
	}

	return xaddress.LoadAddrInfo(keyPath, crypto)
}

// 用账户私钥对数据做签名
func SignByPrivateKey(cryptoType string, privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	crypto, err := cryptoClient.CreateCryptoClient(cryptoType)
	if err != nil {
		return nil, err
	}

	return crypto.SignECDSA(privateKey, data)
}
