package xuperos

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"io/ioutil"
	"path/filepath"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/kernel/network"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
)

// 代理依赖组件实例化操作，方便mock单测和并行开发
type EngineRelyAgentImpl struct {
	engine def.Engine
}

func NewEngineRelyAgent(engine def.Engine) *EngineRelyAgentImpl {
	return &EngineRelyAgentImpl{engine}
}

// 创建并启动p2p网络
func (t *EngineRelyAgentImpl) CreateNetwork() (def.XNetwork, error) {
	conf := t.engine.Context().EnvCfg
	ctx, err := nctx.CreateDomainCtx(conf.GenConfFilePath(conf.NetConf))
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because create network ctx failed.err:%v", err)
	}

	netHD, err := network.CreateNetwork(ctx)
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because create network failed.err:%v", err)
	}

	return netHD, nil
}

// 代理依赖组件实例化操作，方便mock单测和并行开发
type ChainRelyAgentImpl struct {
	chain def.Chain
}

func NewChainRelyAgent(chain def.Chain) *ChainRelyAgentImpl {
	return &ChainRelyAgentImpl{chain}
}

// 创建账本
func (t *ChainRelyAgentImpl) CreateLedger() (def.XLedger, error) {
	ctx := t.chain.Context()
	engCfg := ctx.EngCfg
	dataPathOthers := make([]string, len(engCfg.DataPathOthers))
	for i, path := range engCfg.DataPathOthers {
		dataPathOthers[i] = filepath.Join(path, ctx.BCName)
	}
	kvParam := &kvdb.KVParameter{
		DBPath:     filepath.Join(ctx.DataDir, "ledger"),
		OtherPaths: dataPathOthers,
	}

	return ledger.Open(kvParam, ctx.Crypto)
}

// 创建状态机实例
func (t *ChainRelyAgentImpl) CreateState() (def.XState, error) {
	ctx := t.chain.Context()
	return state.New(ctx.BCName, kvParam, ctx.AddrInfo, ctx.Ledger, ctx.Crypto)
}

// 创建合约实例
func (t *ChainRelyAgentImpl) CreateContract() (def.XContract, error) {
	return contract.CreateManager("default", new(chainCore))
}

// 创建共识实例
func (t *ChainRelyAgentImpl) CreateConsensus() (def.XConsensus, error) {
	ctx := t.chain.Context()
	consensusCtx := context.CreateConsensusCtx(ctx.BCName, ctx.Ledger, ctx.Net, ctx.Crypto, ctx.BaseCtx)
	return consensus.NewPluggableConsensus(consensusCtx, nil, ctx.Contract)
}

// 加密
func (t *ChainRelyAgentImpl) CreateCrypto() (def.XCrypto, error) {
	// 创世块配置
	ctx := t.chain.Context()
	rootJs, err := ioutil.ReadFile(filepath.Join(ctx.DataDir, def.BlockChainConfig))
	if err != nil {
		return nil, fmt.Errorf("read xuper.json failed: %v", err)
	}

	// 加密
	cryptoType, err := GetCryptoType(rootJs)
	if err != nil {
		return nil, fmt.Errorf("crypto type not found: %v", err)
	}

	return client.CreateCryptoClient(cryptoType)
}

// 权限
func (t *ChainRelyAgentImpl) CreateAcl() (def.XAcl, error) {
	return nil, fmt.Errorf("the interface is not implemented")
}
