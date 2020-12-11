package xuperos

import (
	"bytes"
	"fmt"
	"github.com/xuperchain/xuperchain/core/global"
	"github.com/xuperchain/xuperchain/core/pb"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/reader"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
	"time"
)

type Chain struct {
	// 链级上下文
	ctx *def.ChainCtx
	// log
	log logs.Logger

	// 矿工
	miner *miner
	// 交易处理
	processor *txProcessor
	// 账本同步
	keeper *LedgerKeeper
	// 依赖代理组件
	relyAgent def.ChainRelyAgent

	// 读组件
	reader reader.Reader
}

// 从本地存储加载链
func LoadChain(ctx *def.ChainCtx) (*Chain, error) {
	chain := &Chain{}
	chain.ctx = ctx
	chain.relyAgent = NewChainRelyAgent(chain)

	// 初始化链环境上下文
	err := chain.initChainCtx()
	if err != nil {
		return nil, fmt.Errorf("init chain ctx error: %v", err)
	}

	chain.log = ctx.XLog

	// 注册合约
	RegisterKernMethod()

	// TODO: 注册VAT

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

func (t *Chain) Context() *def.ChainCtx {
	return t.ctx
}

func (t *Chain) Status() int {
	return t.ctx.Status
}

func (t *Chain) Reader() reader.Reader {
	return t.reader
}

func (t *Chain) Start() {
	// 周期repost本地未上链的交易
	go t.processor.repostOfflineTx()

	// 启动矿工
	t.miner.start()
}

func (t *Chain) Stop() {
	// 停止矿工
	t.miner.stop()
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
	ctx := t.ctx
	if ctx == nil || ctx.EnvCfg == nil || ctx.EngCfg == nil {
		return fmt.Errorf("ctx or config is nil")
	}

	if ctx.BCName == "" || ctx.DataDir == "" {
		return fmt.Errorf("params is nil, bcName=%s, dataDir=%s", ctx.BCName, ctx.DataDir)
	}

	log, err := logs.NewLogger("", ctx.BCName)
	if err != nil {
		return fmt.Errorf("new logger error: %v", err)
	}

	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()

	// 加密
	ctx.Crypto, err = t.relyAgent.CreateCrypto()
	if err != nil {
		log.Error("create crypto client failed", "error", err)
		return err
	}

	// 账户信息
	keyPath := t.ctx.EngCfg.Miner.KeyPath
	ctx.Address, err = def.LoadAddrInfo(keyPath, ctx.Crypto)
	if err != nil {
		log.Error("load addr info error", "bcName", ctx.BCName, "keyPath", keyPath, "error", err)
		return err
	}

	// 账本
	ctx.Ledger, err = t.relyAgent.CreateLedger()
	if err != nil {
		log.Error("create ledger error", "bcName", ctx.BCName, "error", err)
		return err
	}

	// 状态机
	ctx.State, err = t.relyAgent.CreateState()
	if err != nil {
		log.Error("create state error", "bcName", ctx.BCName, "error", err)
		return err
	}

	// 合约
	ctx.Contract, err = t.relyAgent.CreateContract()
	if err != nil {
		log.Error("create contract error", "bcName", ctx.BCName, "error", err)
		return err
	}

	// 共识
	ctx.Consensus, err = t.relyAgent.CreateConsensus()
	if err != nil {
		log.Error("create consensus error", "bcName", ctx.BCName, "error", err)
		return err
	}

	return nil
}
