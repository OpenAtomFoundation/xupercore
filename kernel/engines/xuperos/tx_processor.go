// 交易处理
package xuperos

import (
	"github.com/xuperchain/xuperchain/core/pb"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
	"github.com/xuperchain/xupercore/lib/logs"
)

// 负责交易交易处理相关逻辑封装
type txProcessor struct {
	ctx *def.ChainCtx
	log logs.Logger
}

func NewTxProcessor(ctx *def.ChainCtx) *txProcessor {
	obj := &txProcessor{
		ctx: ctx,
		log: ctx.GetLog(),
	}

	return obj
}

// 验证交易
func (t *txProcessor) VerifyTx(in *pb.Transaction) (bool, error) {
	txId := in.GetTxid()
	if txId == nil {
		return false, ErrTxIdNil
	}

	return t.ctx.State.VerifyTx(in)
}

// 提交交易到状态机和未确认交易池
func (t *txProcessor) SubmitTx(tx *pb.Transaction) error {
	err := t.ctx.State.DoTx(in)
	if err != nil && err != utxo.ErrAlreadyInUnconfirmed {
		t.handled.Delete(string(in.GetTxid()))
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
