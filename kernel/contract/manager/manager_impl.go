package manager

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/hyperledger/burrow/crypto"
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
	r := args["r"]
	s := args["s"]
	hash := args["hash"]
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
	unc := crypto.UncompressedSignatureFromParams(r, s)

	sig, err := crypto.SignatureFromBytes(unc, crypto.CurveTypeSecp256k1)
	if err != nil {
		return nil, err
	}

	if bytes.Equal(sig.RawBytes(), []byte(hash)) {
		return nil, errors.New("signature verification failed")
	}
	// TODO
	input, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	args1 := map[string][]byte{
		"input":       input,
		"jsonEncoded": []byte("false"),
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
