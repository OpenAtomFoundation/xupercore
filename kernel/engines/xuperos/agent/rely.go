package agent

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	ldef "github.com/chunhui01/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/network"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/permission/acl"
	actx "github.com/xuperchain/xupercore/kernel/permission/acl/context"
	cryptoClient "github.com/xuperchain/xupercore/lib/crypto/client"

	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

// 代理依赖组件实例化操作，方便mock单测和并行开发
type EngineRelyAgentImpl struct {
	engine def.Engine
}

func NewEngineRelyAgent(engine def.Engine) *EngineRelyAgentImpl {
	return &EngineRelyAgentImpl{engine}
}

// 创建并启动p2p网络
func (t *EngineRelyAgentImpl) CreateNetwork() (common.XNetwork, error) {
	ctx, err := nctx.NewNetCtx(t.engine.Context().EnvCfg)
	if err != nil {
		return nil, fmt.Errorf("create network context failed.err:%v", err)
	}

	netObj, err := network.NewNetwork(ctx)
	if err != nil {
		return nil, fmt.Errorf("create network object failed.err:%v", err)
	}

	return netObj, nil
}

// 代理依赖组件实例化操作，方便mock单测和并行开发
type ChainRelyAgentImpl struct {
	chain common.Chain
}

func NewChainRelyAgent(chain common.Chain) *ChainRelyAgentImpl {
	return &ChainRelyAgentImpl{chain}
}

// 创建账本
func (t *ChainRelyAgentImpl) CreateLedger(isCreate bool) (common.XLedger, error) {
	ctx := t.chain.Context()
	legCtx, err := ldef.NewLedgerCtx(ctx.EnvCfg, ctx.BCName)
	if err != nil {
		return nil, fmt.Errorf("new ledger ctx failed.err:%v", err)
	}

	leg, err := ledger.NewLedger(legCtx, isCreate)
	if err != nil {
		return nil, fmt.Errorf("new ledger failed.err:%v", err)
	}

	return leg, nil
}

// 创建状态机实例
func (t *ChainRelyAgentImpl) CreateState(leg common.XLedger) (common.XState, error) {
	ctx := t.chain.Context()
	legCtx, err := ldef.NewLedgerCtx(ctx.EnvCfg, ctx.BCName)
	if err != nil {
		return nil, fmt.Errorf("new ledger ctx failed.err:%v", err)
	}

	stat, err := state.NewState(legCtx, leg)
	if err != nil {
		return nil, fmt.Errorf("new state failed.err:%v", err)
	}

	return stat, nil
}

// 加密
func (t *ChainRelyAgentImpl) CreateCrypto(cryptoType string) (common.XCrypto, error) {
	crypto, err := cryptoClient.CreateCryptoClient(cryptoType)
	if err != nil {
		return nil, fmt.Errorf("create crypto client failed.err:%v,type:%s", err, cryptoType)
	}

	return crypto, nil
}

// Acl权限
func (t *ChainRelyAgentImpl) CreateAcl() (common.XAcl, error) {
	ctx := t.chain.Context()
	legAgent := agent.NewLedgerAgent(ctx)
	aclCtx, err := actx.NewAclCtx(ctx.BCName, legAgent, ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create acl ctx failed.err:%v", err)
	}

	aclObj, err := acl.NewACLManager(aclCtx)
	if err != nil {
		return nil, fmt.Errorf("create acl failed.err:%v", err)
	}

	return aclObj, nil
}

// 创建合约实例
func (t *ChainRelyAgentImpl) CreateContract() (common.XContract, error) {
	return contract.CreateManager("default", new(chainCore))
}

// 创建共识实例
func (t *ChainRelyAgentImpl) CreateConsensus() (def.XConsensus, error) {
	ctx := t.chain.Context()
	consensusCtx := context.CreateConsensusCtx(ctx.BCName, ctx.Ledger, ctx.Net, ctx.Crypto, ctx.BaseCtx)
	return consensus.NewPluggableConsensus(consensusCtx, nil, ctx.Contract)
}
