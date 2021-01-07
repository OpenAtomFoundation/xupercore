package xuperos

import (
	"encoding/json"
	"github.com/xuperchain/xuperchain/core/common/log"
	"github.com/xuperchain/xuperchain/core/global"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/pb"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	"io/ioutil"
	"os"
	"path/filepath"
)

// GetKVEngineType get kv engine type from xuper.json
func GetKVEngineType(config []byte) (string, error) {
	rootJSON := map[string]interface{}{}
	err := json.Unmarshal(config, &rootJSON)
	if err != nil {
		return "", err
	}
	kvEngineType := rootJSON["kvengine"]
	if kvEngineType == nil {
		return "default", nil
	}
	return kvEngineType.(string), nil
}

// GetCryptoType get crypto type from xuper.json
func GetCryptoType(config []byte) (string, error) {
	rootJSON := map[string]interface{}{}
	err := json.Unmarshal(config, &rootJSON)
	if err != nil {
		return "", err
	}
	cryptoType := rootJSON["crypto"]
	if cryptoType == nil {
		return client.CryptoTypeDefault, nil
	}
	return cryptoType.(string), nil
}

func CreateBlockChain(path string, name string, config []byte) error {
	if global.PathExists(path) {
		log.Warn("path exist", "path", path)
		return ErrBlockChainExist
	}

	err := os.MkdirAll(path, 0755)
	if err != nil {
		log.Warn("can't create path[" + path + "] %v")
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(path)
		}
	}()

	rootFile := filepath.Join(path, def.BlockChainConfig)
	err = ioutil.WriteFile(rootFile, config, 0666)
	if err != nil {
		log.Warn("write file error ", "file", rootFile)
		return err
	}

	// 创建账本
	kvEngineType, err := GetKVEngineType(config)
	if err != nil {
		log.Warn("failed to get `kvengine`", "err", err)
		return err
	}
	cryptoType, err := GetCryptoType(config)
	if err != nil {
		log.Warn("failed to get `crypto`", "err", err)
		return err
	}
	newLedger, err := ledger.NewLedger(path, log, nil, kvEngineType, cryptoType)
	if err != nil {
		log.Warn("NewLedger error", "path", path, "err", err)
		return err
	}
	defer newLedger.Close()

	tx, err := utxo.GenerateRootTx(config)
	if err != nil {
		log.Warn("GenerateRootTx error", "path", path, "err", err)
		return err
	}
	txList := []*pb.Transaction{tx}
	log.Trace("Start to ConfirmBlock")
	b, err := newLedger.FormatRootBlock(txList)
	if err != nil {
		log.Warn("format block error", "err", err)
		return ErrCreateBlockChain
	}
	newLedger.ConfirmBlock(b, true)
	log.Info("ConfirmBlock Success", "Height", 1)

	// 更新状态机
	//TODO 因为是创建创世块，所以这里不填写publicKey和address, 后果是如果存在合约的话，肯定是执行失败
	utxovm, err := utxo.NewUtxoVM(name, newLedger, path, "", "", nil, log, false, kvEngineType, cryptoType)
	if err != nil {
		log.Warn("NewUtxoVM error", "path", path, "err", err)
		return err
	}
	defer utxovm.Close()

	utxovm.DebugTx(tx)

	err = utxovm.Play(b.Blockid)
	if err != nil {
		log.Warn("utxo play error ", "error", err, "blockid", b.Blockid)
		return err
	}

	return nil
}
