package xaddress

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"path/filepath"

	cryptoClinet "github.com/xuperchain/xupercore/lib/crypto/client/base"
)

type Address struct {
	Address       string
	PrivateKey    *ecdsa.PrivateKey
	PrivateKeyStr string
	PublicKey     *ecdsa.PublicKey
	PublicKeyStr  string
}

func LoadAddress(keyDir string) (string, error) {
	addr, err := ioutil.ReadFile(filepath.Join(keyDir, "address"))
	if err != nil {
		return "", fmt.Errorf("read address error: %v", err)
	}

	return string(addr), nil
}

func LoadAddrInfo(keyDir string, crypto cryptoClinet.CryptoClient) (*Address, error) {
	addr, err := ioutil.ReadFile(filepath.Join(keyDir, "address"))
	if err != nil {
		return nil, fmt.Errorf("read address error: %v", err)
	}

	priKey, err := ioutil.ReadFile(filepath.Join(keyDir, "private.key"))
	if err != nil {
		return nil, fmt.Errorf("read private.key error: %v", err)
	}
	privateKey, err := crypto.GetEcdsaPrivateKeyFromJsonStr(string(priKey))
	if err != nil {
		return nil, fmt.Errorf("decode private.key error: %v", err)
	}

	pubKey, err := ioutil.ReadFile(filepath.Join(keyDir, "public.key"))
	if err != nil {
		return nil, fmt.Errorf("read public.key error: %v", err)
	}
	publicKey, err := crypto.GetEcdsaPublicKeyFromJsonStr(string(pubKey))
	if err != nil {
		return nil, fmt.Errorf("decode public.key error: %v", err)
	}

	addInfo := &Address{
		Address:       string(addr),
		PrivateKey:    privateKey,
		PrivateKeyStr: string(priKey),
		PublicKey:     publicKey,
		PublicKeyStr:  string(pubKey),
	}
	return addInfo, nil
}
