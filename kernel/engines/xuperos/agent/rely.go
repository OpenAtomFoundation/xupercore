package agent

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	ldef "github.com/chunhui01/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/kernel/consensus"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	cdef "github.com/xuperchain/xupercore/kernel/consensus/def"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/network"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/permission/acl"
	aclBase "github.com/xuperchain/xupercore/kernel/permission/acl/base"
	actx "github.com/xuperchain/xupercore/kernel/permission/acl/context"
	cryptoClient "github.com/xuperchain/xupercore/lib/crypto/client"
	cryptoBase "github.com/xuperchain/xupercore/lib/crypto/client/base"
)

// 代理依赖组件实例化操作，方便mock单测和并行开发
type EngineRelyAgentImpl struct {
	engine def.Engine
}

func NewEngineRelyAgent(engine def.Engine) *EngineRelyAgentImpl {
	return &EngineRelyAgentImpl{engine}
}

// 创建并启动p2p网络
func (t *EngineRelyAgentImpl) CreateNetwork() (network.Network, error) {
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
func (t *ChainRelyAgentImpl) CreateLedger(isCreate bool) (*ledger.Ledger, error) {
	ctx := t.chain.Context()
	legCtx, err := ldef.NewLedgerCtx(ctx.EngCtx.EnvCfg, ctx.BCName)
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
func (t *ChainRelyAgentImpl) CreateState(leg *ledger.Ledger) (*state.State, error) {
	ctx := t.chain.Context()
	legCtx, err := ldef.NewLedgerCtx(ctx.EngCtx.EnvCfg, ctx.BCName)
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
func (t *ChainRelyAgentImpl) CreateCrypto(cryptoType string) (cryptoBase.CryptoClient, error) {
	crypto, err := cryptoClient.CreateCryptoClient(cryptoType)
	if err != nil {
		return nil, fmt.Errorf("create crypto client failed.err:%v,type:%s", err, cryptoType)
	}

	return crypto, nil
}

// Acl权限
func (t *ChainRelyAgentImpl) CreateAcl() (aclBase.AclManager, error) {
	ctx := t.chain.Context()
	legAgent := NewLedgerAgent(ctx)
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
func (t *ChainRelyAgentImpl) CreateContract() (contract.Manager, error) {
	ctx := t.chain.Context()
	mgCfg := &contract.ManagerConfig{
		Core: NewChainCoreAgent(ctx),
	}
	contractObj, err := contract.CreateManager("default", mgCfg)
	if err != nil {
		return nil, fmt.Errorf("create contract manager failed.err:%v", err)
	}

	return contractObj, nil
}

// 创建共识实例
func (t *ChainRelyAgentImpl) CreateConsensus() (consensus.ConsensusInterface, error) {
	ctx := t.chain.Context()
	legAgent := NewLedgerAgent(ctx)
	consCtx := &cctx.ConsensusCtx{
		BcName:   ctx.BCName,
		Address:  ctx.Address,
		Crypto:   ctx.Crypto,
		Contract: ctx.Contract,
		Ledger:   legAgent,
		Network:  ctx.Network,
	}

	log, err := logs.NewLogger("", cdef.SubModName)
	if err != nil {
		return nil, fmt.Errorf("create consensus failed because new logger error.err:%v", err)
	}
	consCtx.XLog = log
	consCtx.Timer = timer.NewXTimer()

	cons, err := consensus.NewPluggableConsensus(consCtx)
	if err != nil {
		return nil, fmt.Errorf("new pluggable consensus failed.err:%v", err)
	}

	return cons, nil
}
