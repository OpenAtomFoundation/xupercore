package context

import (
	"fmt"

	lconf "github.com/xuperchain/xupercore/bcs/ledger/xledger/config"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract"
	governToken "github.com/xuperchain/xupercore/kernel/contract/proposal/govern_token"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/propose"
	timerTask "github.com/xuperchain/xupercore/kernel/contract/proposal/timer"
	aclBase "github.com/xuperchain/xupercore/kernel/permission/acl/base"
	cryptoBase "github.com/xuperchain/xupercore/lib/crypto/client/base"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// 状态机运行上下文环境
type StateCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *xconf.EnvConf
	// 账本配置
	LedgerCfg *lconf.XLedgerConf
	// 链名
	BCName string
	// ledger handle
	Ledger *ledger.Ledger
	// crypto client
	Crypt cryptoBase.CryptoClient
	// acl manager
	// 注意：注入后才可以使用
	AclMgr aclBase.AclManager
	// contract Manager
	// 注意：依赖注入后才可以使用
	ContractMgr contract.Manager
	// 注意：注入后才可以使用
	GovernTokenMgr governToken.GovManager
	// 注意：注入后才可以使用
	ProposalMgr propose.ProposeManager
	// 注意：注入后才可以使用
	TimerTaskMgr timerTask.TimerManager
}

func NewStateCtx(envCfg *xconf.EnvConf, bcName string,
	leg *ledger.Ledger, crypt cryptoBase.CryptoClient) (*StateCtx, error) {
	// 参数检查
	if envCfg == nil || leg == nil || crypt == nil || bcName == "" {
		return nil, fmt.Errorf("create state context failed because env conf is nil")
	}

	// 加载配置
	lcfg, err := lconf.LoadLedgerConf(envCfg.GenConfFilePath(envCfg.LedgerConf))
	if err != nil {
		return nil, fmt.Errorf("create state context failed because load config error.err:%v", err)
	}
	log, err := logs.NewLogger("", def.StateSubModName)
	if err != nil {
		return nil, fmt.Errorf("create state context failed because new logger error. err:%v", err)
	}

	ctx := new(StateCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.EnvCfg = envCfg
	ctx.LedgerCfg = lcfg
	ctx.BCName = bcName
	ctx.Ledger = leg
	ctx.Crypt = crypt

	return ctx, nil
}

func (t *StateCtx) SetAclMG(aclMgr aclBase.AclManager) {
	t.AclMgr = aclMgr
}

func (t *StateCtx) SetContractMG(contractMgr contract.Manager) {
	t.ContractMgr = contractMgr
}

func (t *StateCtx) SetGovernTokenMG(governTokenMgr governToken.GovManager) {
	t.GovernTokenMgr = governTokenMgr
}

func (t *StateCtx) SetProposalMG(proposalMgr propose.ProposeManager) {
	t.ProposalMgr = proposalMgr
}

func (t *StateCtx) SetTimerTaskMG(timerTaskMgr timerTask.TimerManager) {
	t.TimerTaskMgr = timerTaskMgr
}

//state各个func里尽量调一下判断
func (t *StateCtx) IsInit() bool {
	if t.AclMgr == nil || t.ContractMgr == nil || t.GovernTokenMgr == nil || t.ProposalMgr == nil ||
		t.TimerTaskMgr == nil || t.Crypt == nil || t.Ledger == nil {
		return false
	}

	return true
}
