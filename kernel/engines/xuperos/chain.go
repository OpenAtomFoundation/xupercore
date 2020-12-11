package xuperos

import (
	"bytes"
	"fmt"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/reader"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xaddress"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/agent"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"

	"github.com/xuperchain/xuperchain/core/global"
	"github.com/xuperchain/xuperchain/core/pb"
)

type Chain struct {
	// 链上下文
	ctx *common.ChainCtx
	// log
	log logs.Logger
	// 矿工
	miner *miner
	// 交易处理
	processor *txProcessor
	// 账本同步
	keeper *LedgerKeeper
	// 依赖代理组件
	relyAgent common.ChainRelyAgent

	// 读组件
	reader reader.Reader
}

// 从本地存储加载链
func LoadChain(engCtx *common.EngineCtx, bcName string) (*Chain, error) {
	if engCtx == nil || bcName == "" {
		return nil, fmt.Errorf("load chains failed because param error", "bc_name", bcName)
	}

	// 实例化链日志句柄
	log, err := logs.NewLogger("", bcName)
	if err != nil {
		return nil, fmt.Errorf("new logger failed.err:%v", err)
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

	// 初始化链环境上下文
	err = chainObj.initChainCtx()
	if err != nil {
		t.log.Error("init chain ctx failed", "bc_name", bcName, "err", err)
		return nil, fmt.Errorf("init chain ctx failed")
	}

	// 账本同步
	chain.keeper = NewLedgerKeeper(ctx)
	// 实例化矿工
	chain.miner = NewMiner(ctx, chain.keeper)
	// 交易处理器
	chain.processor = NewTxProcessor(ctx)

	chain.reader = reader.NewReader(chain)
	return chain, nil
}

// 供单测时设置rely agent为mock agent，非并发安全
func (t *Chain) SetRelyAgent(agent def.ChainRelyAgent) error {
	if agent == nil {
		return ErrParamError
	}

	t.relyAgent = agent
	return nil
}

func (t *Chain) Start() {
	// 启动矿工
	t.miner.start()
}

func (t *Chain) Stop() {
	// 停止矿工
	t.miner.stop()
}

func (t *Chain) Context() *def.ChainCtx {
	return t.ctx
}

func (t *Chain) Status() int {
	return t.ctx.Status
}

func (t *Chain) Reader() reader.Reader {
	return t.reader
}

// 交易预执行
func (t *Chain) PreExec(request *pb.InvokeRPCRequest) (*pb.InvokeResponse, error) {
	return t.ctx.State.PreExec(request)
}

// 交易和区块结构由账本定义
func (t *Chain) ProcTx(request *pb.TxStatus) error {
	if t.Status() != global.Normal {
		t.log.Error("chain status not ready", "logid", request.Header.Logid)
		return def.ErrBlockChainNotReady
	}

	// 验证交易
	txValid, err := t.processor.verifyTx(request.Tx)
	if !txValid {
		t.log.Error("verify tx error", "logid", request.Header.Logid, "txid", global.F(request.Tx.Txid), "error", err)
		return err
	}

	// 提交交易
	err = t.processor.submitTx(request.Tx)
	if err != nil {
		t.log.Error("submit tx error", "logid", request.Header.Logid, "txid", global.F(request.Tx.Txid), "error", err)
		return err
	}

	return nil
}

// 处理新区块
func (t *Chain) ProcBlock(in *pb.Block) error {
	hd := &global.XContext{Timer: global.NewXTimer()}
	if t.ctx.Ledger.ExistBlock(in.GetBlock().GetBlockid()) {
		t.log.Debug("block is exist", "logid", in.Header.Logid, "cost", hd.Timer.Print())
		return nil
	}

	if bytes.Equal(t.ctx.State.GetLatestBlockId(), in.GetBlock().GetPreHash()) {
		t.log.Trace("appending block in SendBlock", "time", time.Now().UnixNano(), "bcName", t.ctx.BCName, "tipID", global.F(t.ctx.State.GetLatestBlockId()))
		ctx := CreateLedgerTaskCtx([]*SimpleBlock{
			&SimpleBlock{
				internalBlock: in.GetBlock(),
				logid:         in.GetHeader().GetLogid() + "_" + in.GetHeader().GetFromNode()},
		}, nil, hd)
		t.keeper.PutTask(ctx, Appending, -1)
		return nil
	}

	t.log.Trace("sync blocks in SendBlock", "time", time.Now().UnixNano(), "bcName", t.ctx.BCName, "tipID", global.F(t.ctx.State.GetLatestBlockId()))
	ctx := CreateLedgerTaskCtx(nil, []string{in.GetHeader().GetFromNode()}, hd)
	t.keeper.PutTask(ctx, Syncing, -1)
	return nil
}

func (t *Chain) initChainCtx() error {
	// 1.实例化账本
	leg, err := t.relyAgent.CreateLedger(false)
	if err != nil {
		t.log.Error("open ledger failed", "bcName", t.ctx.BCName, "err", err)
		return fmt.Errorf("open ledger failed")
	}
	t.ctx.Ledger = leg

	// 2.实例化状态机
	stat, err := t.relyAgent.CreateState(t.ctx.Ledger)
	if err != nil {
		t.log.Error("open state failed", "bcName", t.ctx.BCName, "err", err)
		return fmt.Errorf("open state failed")
	}
	t.ctx.State = stat

	// 3.实例化加密组件
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

	// 6.Acl
	aclObj, err := t.relyAgent.CreateAcl()
	if err != nil {
		t.log.Error("create acl error", "bcName", ctx.BCName, "err", err)
		return fmt.Errorf("create acl error")
	}
	t.ctx.Acl = aclObj

	// 7.共识
	cons, err := t.relyAgent.CreateConsensus()
	if err != nil {
		t.log.Error("create consensus error", "bcName", ctx.BCName, "err", err)
		return err
	}
	t.ctx.Consensus = cons

	return nil
}
