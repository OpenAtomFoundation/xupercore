package manager

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/txs"
	"github.com/xuperchain/xupercore/bcs/contract/evm"
	"math/big"
	"path/filepath"
	"sync/atomic"

	"github.com/xuperchain/xupercore/lib/logs"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
)

type managerImpl struct {
	core      contract.ChainCore
	xbridge   *bridge.XBridge
	kregistry registryImpl
}

func newManagerImpl(cfg *contract.ManagerConfig) (contract.Manager, error) {
	if cfg.Basedir == "" || !filepath.IsAbs(cfg.Basedir) {
		return nil, errors.New("base dir of contract manager must be absolute")
	}
	if cfg.BCName == "" {
		return nil, errors.New("empty chain name when init contract manager")
	}
	if cfg.Core == nil {
		return nil, errors.New("nil chain core when init contract manager")
	}
	if cfg.XMReader == nil {
		return nil, errors.New("nil xmodel reader when init contract manager")
	}
	if cfg.EnvConf == nil && cfg.Config == nil {
		return nil, errors.New("nil contract config when init contract manager")
	}
	var xcfg *contract.ContractConfig
	if cfg.EnvConf == nil {
		xcfg = cfg.Config
	} else {
		var err error
		xcfg, err = loadConfig(cfg.EnvConf.GenConfFilePath(contractConfigName))
		if err != nil {
			return nil, fmt.Errorf("error while load contract config:%s", err)
		}
	}

	m := &managerImpl{
		core: cfg.Core,
	}
	var logDriver logs.Logger
	if cfg.Config != nil {
		logDriver = cfg.Config.LogDriver
	}
	xbridge, err := bridge.New(&bridge.XBridgeConfig{
		Basedir: cfg.Basedir,
		VMConfigs: map[bridge.ContractType]bridge.VMConfig{
			bridge.TypeWasm:   &xcfg.Wasm,
			bridge.TypeNative: &xcfg.Native,
			bridge.TypeEvm:    &xcfg.EVM,
			bridge.TypeKernel: &contract.XkernelConfig{
				Driver:   xcfg.Xkernel.Driver,
				Enable:   xcfg.Xkernel.Enable,
				Registry: &m.kregistry,
			},
		},
		Config:    *xcfg,
		XModel:    cfg.XMReader,
		Core:      cfg.Core,
		LogDriver: logDriver,
	})
	if err != nil {
		return nil, err
	}
	m.xbridge = xbridge
	registry := &m.kregistry
	registry.RegisterKernMethod("$contract", "deployContract", m.deployContract)
	registry.RegisterKernMethod("$contract", "upgradeContract", m.upgradeContract)
	registry.RegisterKernMethod("$contract", "proxy", m.evmproxy)

	registry.RegisterShortcut("Deploy", "$contract", "deployContract")
	registry.RegisterShortcut("Upgrade", "$contract", "upgradeContract")
	return m, nil
}

func (m *managerImpl) NewContext(cfg *contract.ContextConfig) (contract.Context, error) {
	return m.xbridge.NewContext(cfg)
}

func (m *managerImpl) NewStateSandbox(cfg *contract.SandboxConfig) (contract.StateSandbox, error) {
	return sandbox.NewXModelCache(cfg), nil
}

func (m *managerImpl) GetKernRegistry() contract.KernRegistry {
	return &m.kregistry
}

func (m *managerImpl) deployContract(ctx contract.KContext) (*contract.Response, error) {
	// check if account exist
	accountName := ctx.Args()["account_name"]
	contractName := ctx.Args()["contract_name"]
	if accountName == nil || contractName == nil {
		return nil, errors.New("invoke DeployMethod error, account name or contract name is nil")
	}
	// check if contractName is ok
	if err := contract.ValidContractName(string(contractName)); err != nil {
		return nil, fmt.Errorf("deploy failed, contract `%s` contains illegal character, error: %s", contractName, err)
	}
	_, err := ctx.Get(utils.GetAccountBucket(), accountName)
	if err != nil {
		return nil, fmt.Errorf("get account `%s` error: %s", accountName, err)
	}

	resp, limit, err := m.xbridge.DeployContract(ctx)
	if err != nil {
		return nil, err
	}
	ctx.AddResourceUsed(limit)

	// key: contract, value: account
	err = ctx.Put(utils.GetContract2AccountBucket(), contractName, accountName)
	if err != nil {
		return nil, err
	}
	key := utils.MakeAccountContractKey(string(accountName), string(contractName))
	err = ctx.Put(utils.GetAccount2ContractBucket(), []byte(key), []byte(utils.GetAccountContractValue()))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *managerImpl) upgradeContract(ctx contract.KContext) (*contract.Response, error) {
	contractName := ctx.Args()["contract_name"]
	if contractName == nil {
		return nil, errors.New("invoke Upgrade error, contract name is nil")
	}

	err := m.core.VerifyContractOwnerPermission(string(contractName), ctx.AuthRequire())
	if err != nil {
		return nil, err
	}

	resp, limit, err := m.xbridge.UpgradeContract(ctx)
	if err != nil {
		return nil, err
	}
	ctx.AddResourceUsed(limit)
	return resp, nil
}

type Transaction struct {
	data txdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	AccountNonce uint64
	Price        *big.Int
	GasLimit     uint64
	//Recipient    *common.Address
	Amount  *big.Int
	Payload []byte

	V *big.Int
	R *big.Int
	S *big.Int

	// This is only used when marshaling to JSON.
	// Hash *common.Hash `json:"hash" rlp:"-"`
}

//2. 普通合约调用方案方案
func (m *managerImpl) evmproxy(ctx contract.KContext) (*contract.Response, error) {
	return m.evmproxy2(ctx)
}

func (m *managerImpl) evmproxy2(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()

	//to := args["to"]
	to := "313131312D2D2D2D2D2D2D2D2D636F756E746572"
	// TODO length check
	//from := args["from"]
	from := "b60e8dd61c5d32be8058bb8eb970870f07233155"
	data := args["param"]
	r1 := "9910f8e6fc72f08b0caddf1b1135ed4e4dbee034849fab65eed88003e76ac087"
	s1 := "04bf09c57e9af2829a39859ba93525fd21da489abf78d0e4fe613d5411090a82"
	hash1 := "549e6094d23179b5d0e092ee32621cf79d3bb35855043d713ca86fbd096a4639"
	//gas := string(args["gas"])
	//nonce := string(args["nonce"])
	//gasLimit := string(args["gas_limit"])
	//gasPrice := ""
	//amount := ""

	r, err := hex.DecodeString(string(r1))
	if err != nil {
		return nil, err
	}

	// TODO 避免来回转换
	s, err := hex.DecodeString(string(s1))
	if err != nil {
		return nil, err
	}
	//hash, err := hex.DecodeString(string(args["hash"]))
	//if err != nil {
	//	return nil, err
	//}
	//all := args["all"]
	// TODO 0x prefix

	//req := &web3.EthSendTransactionParams{}
	// TODO  两种的区别
	//if err := json.Unmarshal(data, req); err != nil {
	//	return nil, err
	//}
	//hash:=req.Hash
	// TODO  variable naming
	//unc := crypto.UncompressedSignatureFromParams([]byte(req.R), []byte(req.S))
	//unc := crypto.UncompressedSignatureFromParams(r, s)

	//sig, err := crypto.SignatureFromBytes(unc, crypto.CurveTypeSecp256k1)
	//if err != nil {
	//	return nil, err
	//}
	//hash, err := hex.DecodeString(hash1)
	//if err != nil {
	//	return nil, err
	//}
	//sig.RawBytes()
	//fmt.Println(sig.String())
	//if !bytes.Equal(sig.RawBytes(), []byte(hash)) {
	//	return nil, errors.New("signature verification failed")
	//}
	// TODO
	chainID := 1
	net := uint64(chainID)
	enc, err := txs.RLPEncode(rawTx.Nonce, rawTx.GasPrice, rawTx.GasLimit, rawTx.To, rawTx.Value, rawTx.Data)
	if err != nil {
		return nil, err
	}

	sig := crypto.CompressedSignatureFromParams(rawTx.V-net-8-1, rawTx.R, rawTx.S)
	pub, err := crypto.PublicKeyFromSignature(sig, crypto.Keccak256(enc))
	if err != nil {
		return nil, err
	}
	from := pub.GetAddress()
	unc := crypto.UncompressedSignatureFromParams(rawTx.R, rawTx.S)
	signature, err := crypto.SignatureFromBytes(unc, crypto.CurveTypeSecp256k1)
	if err != nil {
		return nil, err
	}

	input, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	args1 := map[string][]byte{
		"input":       input,
		"jsonEncoded": []byte("false"),
	}
	// for fields
	//tx := web3.EthSendTransactionParams{web3.Transaction{
	//	TransactionIndex: "",
	//	BlockHash:        "",
	//	From:             string(from),
	//	Hash:             "",
	//	Data:             "",
	//	Nonce:            "",
	//	Gas:              gas,
	//	Value:            "",
	//	// TODO 好像没用到
	//	V:           "",
	//	S:           s1,
	//	GasPrice:    "",
	//	To:          to,
	//	BlockNumber: "",
	//	R:           r1,
	//}}
	pk, err := crypto.PublicKeyFromSignature(sig.RawBytes(), hash)
	if err != nil {
		return nil, err
	}
	//msg, err := txs.RLPEncode(nonce, gasPrice, gasLimit, from, amount, data)
	if err != nil {
		return nil, err
	}
	if err := pk.Verify(nil, sig); err != nil {
		return nil, err
	}
	address, err := crypto.AddressFromHexString(string(to))
	if err != nil {
		return nil, err
	}
	contractName, err := evm.DetermineContractNameFromEVM(address)
	if err != nil {
		return nil, err
	}
	fromAddress, err := crypto.AddressFromHexString(string(from))
	if err != nil {
		return nil, err
	}
	Initiator, err := evm.EVMAddressToXchain(fromAddress)
	// TODO
	// 1.地址转换相关问题
	// 2. 跨合约调用
	// 3.合约部署与合约升级
	nctx, err := m.xbridge.NewContext(&contract.ContextConfig{
		State:     ctx,
		Initiator: Initiator,

		AuthRequire: []string{Initiator},
		//
		Caller:                "",
		Module:                "evm",
		ContractName:          contractName,
		ResourceLimits:        contract.MaxLimits,
		CanInitialize:         false,
		TransferAmount:        "",
		ContractSet:           nil,
		ContractCodeFromCache: false,
	})
	if err != nil {
		return nil, err
	}
	resp, err := nctx.Invoke("", args1)
	if err != nil {
		return nil, err
	}
	return resp, err

}

//func UnmarshalTransaction(data []byte,)
func init() {
	contract.Register("default", newManagerImpl)
}
