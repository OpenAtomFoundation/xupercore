// 交易处理
package xuperos

import (
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/pb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/protos"
)

// 负责交易交易处理相关逻辑封装
type txProcessor struct {
	chainCtx *common.ChainCtx
	baseCtx  xctx.XContext
	log      logs.Logger
}

func NewTxProcessor(chainCtx *common.ChainCtx, baseCtx xctx.XContext) *txProcessor {
	obj := &txProcessor{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
	}

	// 如果没设置baseCtx，复用链上下文中的baseCtx
	if obj.baseCtx == nil {
		obj.baseCtx = &xctx.BaseCtx{
			XLog:  chainCtx.GetLog(),
			Timer: chainCtx.GetTimer(),
		}
	}
	obj.log = obj.baseCtx.GetLog()

	return obj
}

// 验证交易
func (t *txProcessor) VerifyTx(tx *lpb.Transaction) error {
	return t.ctx.State.VerifyTx(in)
}

// 提交交易到状态机和未确认交易池
func (t *txProcessor) SubmitTx(tx *lpb.Transaction) error {
	err := t.ctx.State.DoTx(in)
	if err == utxo.ErrAlreadyInUnconfirmed {
		return common.ErrTxAlreadyExist
	}

	return err
}

// 合约预执行
func (t *txProcessor) ContractInvoke(ctxConf *contract.ContextConfig,
	reqs []*protos.InvokeRequest) (*protos.InvokeResponse, error) {

	// check param
	if ctxConf == nil || len(req) < 1 {
		t.log.Warn("contract invoke param error")
		return nil, fmt.Errorf("param error")
	}

	// TODO:1.追加系统内置合约
	// TODO:2.处理合约转账

	// 创建合约运行上下文
	for _, req := range reqs {
		ctx, err := t.ctx.Contract.NewContext(ctxConf)
		if err != nil {
			t.log.Warn("new contract context failed", "err", err)
			return nil, fmt.Errorf("new contract context failed")
		}

		resp, err := ctx.Invoke(req.MethodName, req.Args)
		if err != nil {
			ctx.Release()
			t.log.Warn("contract invoke failed", "err", err, "contract", req.ContractName,
				"method", req.MethodName)
			return nil, fmt.Errorf("contract invoke failed")
		}

		// TODO:增加系统合约判断

		ctx.Release()
	}

	return nil, nil
}
