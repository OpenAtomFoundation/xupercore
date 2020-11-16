package client

import (
	"encoding/json"
	"errors"
	"sort"
	"sync"

	"github.com/xuperchain/crypto/core/account"
	"github.com/xuperchain/xupercore/lib/crypto/client/base"
	"github.com/xuperchain/xupercore/lib/crypto/client/gm"
	"github.com/xuperchain/xupercore/lib/crypto/client/xchain"
)

const (
	// CryptoTypeDefault : default Nist ECC
	CryptoTypeDefault = "default"
	// CryptoTypeGM : support for GM
	CryptoTypeGM = "gm"
	// CryptoTypeSchnorr : support for Nist + Schnorr
	CryptoTypeSchnorr = "schnorr"
)

type NewCryptoFunc func() base.CryptoClient

var (
	servsMu  sync.RWMutex
	services = make(map[string]NewCryptoFunc)
)

func Register(name string, f NewCryptoFunc) {
	servsMu.Lock()
	defer servsMu.Unlock()

	if f == nil {
		panic("crypto: Register new func is nil")
	}
	if _, dup := services[name]; dup {
		panic("crypto: Register called twice for func " + name)
	}
	services[name] = f
}

func Drivers() []string {
	servsMu.RLock()
	defer servsMu.RUnlock()
	list := make([]string, 0, len(services))
	for name := range services {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func CreateCryptoClient(cryptoType string) (base.CryptoClient, error) {
	servsMu.RLock()
	defer servsMu.RUnlock()

	if f, ok := services[cryptoType]; ok {
		return f(), nil
	}

	return nil, errors.New("get cryptoClient fail")
}

// init
func init() {
	Register(CryptoTypeDefault, NewCryptoFunc(eccdefault.GetInstance))
	Register(CryptoTypeGM, NewCryptoFunc(gm.GetInstance))
}

// CreateCryptoClientFromJSONPublicKey create CryptoClient by json encoded public key
func CreateCryptoClientFromJSONPublicKey(jsonKey []byte) (base.CryptoClient, error) {
	cryptoType, err := getCryptoTypeByJSONPublicKey(jsonKey)
	if err != nil {
		return nil, err
	}
	// create crypto client
	return CreateCryptoClient(cryptoType)
}

// CreateCryptoClientFromJSONPrivateKey create CryptoClient by json encoded private key
func CreateCryptoClientFromJSONPrivateKey(jsonKey []byte) (base.CryptoClient, error) {
	cryptoType, err := getCryptoTypeByJSONPrivateKey(jsonKey)
	if err != nil {
		return nil, err
	}
	// create crypto client
	return CreateCryptoClient(cryptoType)
}

func getCryptoTypeByJSONPublicKey(jsonKey []byte) (string, error) {
	publicKey := new(account.ECDSAPublicKey)
	err := json.Unmarshal(jsonKey, publicKey)
	if err != nil {
		return "", err //json有问题
	}
	curveName := publicKey.Curvname
	return getTypeByCurveName(curveName)
}

func getCryptoTypeByJSONPrivateKey(jsonKey []byte) (string, error) {
	privateKey := new(account.ECDSAPrivateKey)
	err := json.Unmarshal(jsonKey, privateKey)
	if err != nil {
		return "", err
	}
	curveName := privateKey.Curvname
	return getTypeByCurveName(curveName)
}

func getTypeByCurveName(name string) (string, error) {
	switch name {
	case "P-256":
		return CryptoTypeDefault, nil
	case "SM2-P-256":
		return CryptoTypeGM, nil
	case "P-256-SN":
		return CryptoTypeSchnorr, nil
	default:
		return "", errors.New("Unknown curve name")
	}
}
