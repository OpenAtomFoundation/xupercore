package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	// ErrBlockChainExist is returned when create an existed block chain
	ErrBlockChainExist = errors.New("BlockChain Exist")
	// ErrCreateBlockChain is returned when create block chain error
	ErrCreateBlockChain = errors.New("Create BlockChain error")
)

// 通过创世块配置创建全新账本
func CreateLedger(bcName, genesisConf string, envCfg *xconf.EnvConf) error {
	if bcName == "" || genesisConf == "" || envCfg == nil {
		return fmt.Errorf("param set error")
	}

	dataDir := envCfg.GenDataAbsPath(envCfg.ChainDir)
	fullpath := filepath.Join(dataDir, bcName)
	if utils.PathExists(fullpath) {
		return ErrBlockChainExist
	}
	err := os.MkdirAll(fullpath, 0755)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadFile(genesisConf)
	if err != nil {
		return err
	}
	rootfile := filepath.Join(fullpath, fmt.Sprintf("%s.json", bcName))
	err = ioutil.WriteFile(rootfile, data, 0666)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	lctx, err := ledger.NewLedgerCtx(envCfg, bcName)
	if err != nil {
		return err
	}
	xledger, err := ledger.CreateLedger(lctx, data)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	tx, err := tx.GenerateRootTx(data)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	txlist := []*pb.Transaction{tx}
	b, err := xledger.FormatRootBlock(txlist)
	if err != nil {
		os.RemoveAll(fullpath)
		return ErrCreateBlockChain
	}
	xledger.ConfirmBlock(b, true)
	cryptoType, err := GetCryptoType(data)
	if err != nil {
		os.RemoveAll(fullpath)
		return ErrCreateBlockChain
	}
	crypt, err := client.CreateCryptoClient(cryptoType)
	if err != nil {
		os.RemoveAll(fullpath)
		return ErrCreateBlockChain
	}
	sctx, err := context.NewStateCtx(envCfg, bcName, xledger, crypt)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	handleState, err := state.NewState(sctx)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}

	defer xledger.Close()
	defer handleState.Close()
	err = handleState.Play(b.Blockid)
	if err != nil {
		return err
	}
	return nil
}

// GetCryptoType get crypto type from xuper.json
func GetCryptoType(data []byte) (string, error) {
	rootJSON := map[string]interface{}{}
	err := json.Unmarshal(data, &rootJSON)
	if err != nil {
		return "", err
	}
	cryptoType := rootJSON["crypto"]
	if cryptoType == nil {
		return client.CryptoTypeDefault, nil
	}
	return cryptoType.(string), nil
}
