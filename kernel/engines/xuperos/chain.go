package xuperos

import (
	"bytes"
	"fmt"
	"time"

	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/pb"
	"github.com/xuperchain/xupercore/kernel/common/xaddress"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/agent"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"

	"github.com/patrickmn/go-cache"
)

const (
	// 提交交易cache有效期(s)
	TxIdCacheExpired = 120 * time.Second
	// 提交交易cache GC 周期（s）
	TxIdCacheGCInterval = 180 * time.Second
)

// 定义一条链的具体行为，对外暴露接口错误统一使用标准错误
type Chain struct {
	// 链上下文
	ctx *common.ChainCtx
	// log
	log logs.Logger
	// 矿工
	miner *miner
	// 依赖代理组件
	relyAgent common.ChainRelyAgent

	// 提交交易cache
	txIdCache *cache.Cache
}

// 从本地存储加载链
func LoadChain(engCtx *common.EngineCtx, bcName string) (*Chain, error) {
	if engCtx == nil || bcName == "" {
		return nil, commom.ErrParameter
	}

	// 实例化链日志句柄
	log, err := logs.NewLogger("", bcName)
	if err != nil {
		return nil, commom.ErrNewLogFailed
	}

	// 实例化链实例
	ctx := &common.ChainCtx{}
	ctx.EngCtx = engCtx
	ctx.BCName = bcName
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	chainObj := &Chain{}
	chainObj.ctx = ctx
	chainObj.log = ctx.XLog
	chainObj.relyAgent = agent.NewChainRelyAgent(chainObj)

	// 初始化链运行环境上下文
	err = chainObj.initChainCtx()
	if err != nil {
		t.log.Error("init chain ctx failed", "bcName", bcName, "err", err)
		return nil, common.ErrNewChainCtxFailed.More("err:%v", err)
	}

	// 创建矿工
	chainObj.miner = NewMiner(ctx)
	chainObj.txIdCache = cache.New(TxIdCacheExpired, TxIdCacheGCInterval)

	return chain, nil
}

// 供单测时设置rely agent为mock agent，非并发安全
func (t *Chain) SetRelyAgent(agent common.ChainRelyAgent) error {
	if agent == nil {
		return common.ErrParameter
	}

	t.relyAgent = agent
	return nil
}

// 阻塞
func (t *Chain) Start() {
	// 启动矿工
	t.miner.start()
}

func (t *Chain) Stop() {
	// 停止矿工
	t.miner.stop()
}

func (t *Chain) Context() *common.ChainCtx {
	return t.ctx
}

// 交易预执行
func (t *Chain) PreExec(ctx xctx.XContext, req []*protos.InvokeRequest) (*protos.InvokeResponse, error) {
	if ctx == "" || ctx.GetLog() == nil || len(req) < 1 {
		return common.ErrParameter
	}
	log := ctx.GetLog()

	// 生成沙盒
	contract.ContextConfig

	// 预执行

	return nil, nil
}

// 提交交易到交易池(xuperos引擎同时更新到状态机和交易池)
func (t *Chain) SubmitTx(ctx xctx.XContext, tx *lpb.Transaction) error {
	if tx == nil || ctx == nil || ctx.GetLog() == nil || tx.GetTxid() == "" {
		return common.ErrParameter
	}
	log := ctx.GetLog()

	// 防止重复提交交易
	if _, exist := t.txIdCache.Get(string(tx.GetTxid())); exist {
		log.Warn("tx already exist,ignore", "txid", utils.F(tx.GetTxid()))
		return common.ErrTxAlreadyExist
	}
	t.txIdCache.Set(string(tx.GetTxid()), true, TxIdCacheExpired)

	txProc := NewTxProcessor(t.ctx, ctx)
	// 验证交易
	err := txProc.VerifyTx(tx)
	if err != nil {
		log.Error("verify tx error", "txid", utils.F(tx.GetTxid()), "err", err)
		return common.ErrTxVerifyFailed.More("err:%v", err)
	}

	// 提交交易
	err = txProc.SubmitTx(tx)
	if err != nil {
		log.Error("submit tx error", "txid", utils.F(tx.GetTxid()), "err", err)
		if !err.Equal(common.ErrTxAlreadyExist) {
			t.txIdCache.Delete(string(tx.GetTxid()))
		}
		return common.ErrSubmitTxFailed.More("err:%v", err)
	}

	log.Info("submit tx succ", "txid", utils.F(tx.GetTxid()))
	return nil
}

// 初始化链运行依赖上下文
func (t *Chain) initChainCtx() error {
	// 1.实例化账本
	leg, err := t.relyAgent.CreateLedger()
	if err != nil {
		t.log.Error("open ledger failed", "bcName", t.ctx.BCName, "err", err)
		return fmt.Errorf("open ledger failed")
	}
	t.ctx.Ledger = leg

	// 2.实例化加密组件
	// 从账本查询加密算法类型
	ctype, err := agent.NewLedgerAgent(t.ctx).GetCryptoType()
	if err != nil {
		t.log.Error("query crypto type failed", "bcName", t.ctx.BCName, "err", err)
		return fmt.Errorf("query crypto type failed")
	}
	crypt, err := t.relyAgent.CreateCrypto(ctype)
	if err != nil {
		t.log.Error("create crypto client failed", "error", err)
		return fmt.Errorf("create crypto client failed")
	}
	t.ctx.Crypto = crypt

	// 3.实例化状态机
	stat, err := t.relyAgent.CreateState(t.ctx.Ledger, t.ctx.Crypto)
	if err != nil {
		t.log.Error("open state failed", "bcName", t.ctx.BCName, "err", err)
		return fmt.Errorf("open state failed")
	}
	t.ctx.State = stat

	// 4.加载节点账户信息
	keyPath := t.ctx.EngCtx.EnvCfg.GenDataAbsPath(t.ctx.EngCtx.EnvCfg.KeyDir)
	addr, err := xaddress.LoadAddrInfo(keyPath, t.ctx.Crypto)
	if err != nil {
		t.log.Error("load node addr info error", "bcName", t.ctx.BCName, "keyPath", keyPath, "err", err)
		return fmt.Errorf("load node addr info error")
	}
	t.ctx.Address = addr

	// 5.合约
	contractObj, err := t.relyAgent.CreateContract()
	if err != nil {
		t.log.Error("create contract manager error", "bcName", ctx.BCName, "err", err)
		return fmt.Errorf("create contract manager error")
	}
	t.ctx.Contract = contractObj
	// 设置合约manager到状态机
	t.ctx.State.SetContractMG(t.ctx.Contract)

	// 6.Acl
	aclObj, err := t.relyAgent.CreateAcl()
	if err != nil {
		t.log.Error("create acl error", "bcName", ctx.BCName, "err", err)
		return fmt.Errorf("create acl error")
	}
	t.ctx.Acl = aclObj
	// 设置acl manager到状态机
	t.ctx.State.SetAclMG(t.ctx.Acl)

	// 7.共识
	cons, err := t.relyAgent.CreateConsensus()
	if err != nil {
		t.log.Error("create consensus error", "bcName", ctx.BCName, "err", err)
		return fmt.Errorf("create consensus error")
	}
	t.ctx.Consensus = cons

	return nil
}
