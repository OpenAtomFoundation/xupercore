package xrandom

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/OpenAtomFoundation/xupercore/crypto-dll-go/bls"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	BlsAccountFileName = "bls.account"
	EthAccountFileName = "eth.account"
)

func loadBlsAccount(ctx *Context) (*bls.Account, error) {

	// load data
	data, err := os.ReadFile(filePath(ctx, BlsAccountFileName))
	if err != nil {
		return nil, err
	}
	data = bytes.TrimSpace(data)

	// convert to account
	account := bls.Account{}
	if err = json.Unmarshal(data, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

func saveBlsAccount(ctx *Context, account *bls.Account) error {
	data, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf(`Failed to marshal account for save to file: %v`, err)
	}
	return os.WriteFile(filePath(ctx, BlsAccountFileName), data, 0644)
}

func filePath(ctx *Context, fileName string) string {
	env := ctx.ChainCtx.EngCtx.EnvCfg
	return env.GenDataAbsPath(path.Join(env.KeyDir, fileName))
}

// load ETH private key
func loadEthAccount(ctx *Context) (*ecdsa.PrivateKey, error) {
	fileContent, err := os.ReadFile(filePath(ctx, EthAccountFileName))
	if err != nil {
		return nil, err
	}
	hexPrivateKey := string(fileContent)
	privateKeyBytes, err := hex.DecodeString(hexPrivateKey)
	if err != nil {
		return nil, err
	}
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// save ETH private key
func saveEthAccount(ctx *Context, privateKey *ecdsa.PrivateKey) error {
	privateKeyBytes := crypto.FromECDSA(privateKey)
	hexPrivateKey := hex.EncodeToString(privateKeyBytes)
	return os.WriteFile(filePath(ctx, EthAccountFileName), []byte(hexPrivateKey), 0600)
}
