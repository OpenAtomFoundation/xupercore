package agent

import (
	"fmt"
	"path/filepath"

	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	statctx "github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/kernel/consensus"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	cdef "github.com/xuperchain/xupercore/kernel/consensus/def"
	"github.com/xuperchain/xupercore/kernel/contract"
	governToken "github.com/xuperchain/xupercore/kernel/contract/proposal/govern_token"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/propose"
	timerTask "github.com/xuperchain/xupercore/kernel/contract/proposal/timer"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/parachain"
	kledger "github.com/xuperchain/xupercore/kernel/ledger"
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
	engine common.Engine
}

func NewEngineRelyAgent(engine common.Engine) *EngineRelyAgentImpl {
	return &EngineRelyAgentImpl{engine}
}

// 创建并启动p2p网络
func (t *EngineRelyAgentImpl) CreateNetwork(envCfg *xconf.EnvConf) (network.Network, error) {
	ctx, err := nctx.NewNetCtx(envCfg)
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
func (t *ChainRelyAgentImpl) CreateLedger() (*ledger.Ledger, error) {
	ctx := t.chain.Context()
	legCtx, err := ledger.NewLedgerCtx(ctx.EngCtx.EnvCfg, ctx.BCName)
	if err != nil {
		return nil, fmt.Errorf("new ledger ctx failed.err:%v", err)
	}

	leg, err := ledger.OpenLedger(legCtx)
	if err != nil {
		return nil, fmt.Errorf("open ledger failed.err:%v", err)
	}

	return leg, nil
}

// 创建状态机实例
func (t *ChainRelyAgentImpl) CreateState(leg *ledger.Ledger,
	crypt cryptoBase.CryptoClient) (*state.State, error) {

	// 创建状态机上下文
	ctx := t.chain.Context()
	stateCtx, err := statctx.NewStateCtx(ctx.EngCtx.EnvCfg, ctx.BCName, leg, crypt)
	if err != nil {
		return nil, fmt.Errorf("new state ctx failed.err:%v", err)
	}

	stat, err := state.NewState(stateCtx)
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
func (t *ChainRelyAgentImpl) CreateContract(xmreader kledger.XMReader) (contract.Manager, error) {
	ctx := t.chain.Context()
	envcfg := ctx.EngCtx.EnvCfg
	basedir := filepath.Join(envcfg.GenDataAbsPath(envcfg.ChainDir), ctx.BCName)

	mgCfg := &contract.ManagerConfig{
		BCName:   ctx.BCName,
		Basedir:  basedir,
		EnvConf:  envcfg,
		Core:     NewChainCoreAgent(ctx),
		XMReader: xmreader,
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
	consCtx := cctx.ConsensusCtx{
		BcName:   ctx.BCName,
		Address:  (*cctx.Address)(ctx.Address),
		Crypto:   ctx.Crypto,
		Contract: ctx.Contract,
		Ledger:   legAgent,
		Network:  ctx.EngCtx.Net,
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

// 创建治理代币实例
func (t *ChainRelyAgentImpl) CreateGovernToken() (governToken.GovManager, error) {
	ctx := t.chain.Context()
	legAgent := NewLedgerAgent(ctx)
	governTokenCtx, err := governToken.NewGovCtx(ctx.BCName, legAgent, ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create govern_token ctx failed.err:%v", err)
	}

	governTokenObj, err := governToken.NewGovManager(governTokenCtx)
	if err != nil {
		return nil, fmt.Errorf("create govern_token instance failed.err:%v", err)
	}

	return governTokenObj, nil
}

// 创建提案实例
func (t *ChainRelyAgentImpl) CreateProposal() (propose.ProposeManager, error) {
	ctx := t.chain.Context()
	legAgent := NewLedgerAgent(ctx)
	proposalCtx, err := propose.NewProposeCtx(ctx.BCName, legAgent, ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create proposal ctx failed.err:%v", err)
	}

	proposalObj, err := propose.NewProposeManager(proposalCtx)
	if err != nil {
		return nil, fmt.Errorf("create proposal instance failed.err:%v", err)
	}

	return proposalObj, nil
}

// 创建定时器任务实例
func (t *ChainRelyAgentImpl) CreateTimerTask() (timerTask.TimerManager, error) {
	ctx := t.chain.Context()
	legAgent := NewLedgerAgent(ctx)
	timerCtx, err := timerTask.NewTimerTaskCtx(ctx.BCName, legAgent, ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create timer_task ctx failed.err:%v", err)
	}

	timerObj, err := timerTask.NewTimerTaskManager(timerCtx)
	if err != nil {
		return nil, fmt.Errorf("create timer_task instance failed.err:%v", err)
	}

	return timerObj, nil
}

// 创建平行链实例
func (t *ChainRelyAgentImpl) CreateParaChain() error {
	ctx := t.chain.Context()
	paraChainCtx, err := parachain.NewParaChainCtx(ctx.BCName, ctx)
	if err != nil {
		return fmt.Errorf("create parachain ctx failed.err:%v", err)
	}

	_, err = parachain.NewParaChainManager(paraChainCtx)
	if err != nil {
		return fmt.Errorf("create parachain instance failed.err:%v", err)
	}

	return nil
}
