package reader

import (
    "fmt"
    "github.com/xuperchain/xuperchain/core/global"
    "github.com/xuperchain/xuperchain/core/pb"
    "github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
    "github.com/xuperchain/xupercore/lib/logs"
)

type Ledger interface {
    // Deprecated: use QueryTransaction instead.
    QueryTx(in *pb.TxStatus) *pb.TxStatus
    QueryTransaction(in *pb.TxStatus) *pb.TxStatus
    // Deprecated: use QueryBlock instead.
    GetBlock(in *pb.BlockID) *pb.Block
    QueryBlock(in *pb.BlockID) *pb.Block
    // Deprecated: use QueryBlockByHeight instead.
    GetBlockByHeight(input *pb.BlockHeight) *pb.Block
    QueryBlockByHeight(input *pb.BlockHeight) *pb.Block
    GetBlockChainStatus(in *pb.BCStatus, viewOption pb.ViewOption) *pb.BCStatus
    // Deprecated: use GetTipStatus instead.
    ConfirmTipBlockChainStatus(in *pb.BCStatus) *pb.BCTipStatus
    GetTipStatus(in *pb.BCStatus) *pb.BCTipStatus
}

type ledgerReader struct {
    ctx    *def.ChainCtx
    log    logs.Logger
    chain  def.Chain
    ledger def.XLedger
}

func NewLedgerReader(chain def.Chain) Ledger {
    reader := &ledgerReader{
        ctx:    chain.Context(),
        log:    chain.Context().XLog,
        ledger: chain.Context().Ledger,
        chain:  chain,
    }

    return reader
}

func (t *ledgerReader) QueryTransaction(in *pb.TxStatus) *pb.TxStatus {
    out := &pb.TxStatus{
        Header: in.Header,
        Bcname: in.Bcname,
        Txid: in.Txid,
        Status: pb.TransactionStatus_UNDEFINE,
    }
    out.Header.Error = pb.XChainErrorEnum_SUCCESS

    if t.chain.Status() != global.Normal {
        out.Header.Error = pb.XChainErrorEnum_CONNECT_REFUSE
        t.log.Warn("refused query tx due to chain status not ready", "logid", in.Header.Logid)
        return out
    }

    tx, err := t.ledger.QueryTransaction(out.Txid)
    if err != nil {
        t.log.Warn("ledger query tx error", "logid", in.Header.Logid, "txId", global.F(out.Txid), "error", err)
        out.Status = pb.TransactionStatus_NOEXIST
        if err == ledger.ErrTxNotFound {
            // 查询unconfirmed表
            tx, err = t.ctx.State.QueryTransaction(out.Txid)
            if err != nil {
                t.log.Warn("state query tx error", "logid", in.Header.Logid, "txId", global.F(out.Txid), "error", err)
                return out
            }
            t.log.Debug("state query tx succeeded", "logid", in.Header.Logid, "txId", global.F(out.Txid))
            out.Status = pb.TransactionStatus_UNCONFIRM
            out.Tx = tx
            return out
        }
    } else {
        t.log.Debug("ledger query tx succeeded", "logid", in.Header.Logid, "txId", global.F(out.Txid))
        out.Status = pb.TransactionStatus_CONFIRM
        // 查询block状态，是否被分叉
        ib, err := t.ledger.QueryBlockHeader(tx.Blockid)
        if err != nil {
            t.log.Warn("query block error", "logid", in.Header.Logid, "txId", global.F(out.Txid), "blockId", global.F(tx.Blockid), "error", err)
            out.Header.Error = pb.XChainErrorEnum_UNKNOW_ERROR
        } else {
            t.log.Debug("query block succeeded", "logid", in.Header.Logid, "txId", global.F(out.Txid), "blockId", global.F(tx.Blockid))
            meta := t.ledger.GetMeta()
            out.Tx = tx
            if ib.InTrunk {
                // out.Distance =  height - ib.height
                out.Distance = meta.TrunkHeight - ib.Height
                out.Status = pb.TransactionStatus_CONFIRM
            } else {
                out.Status = pb.TransactionStatus_FURCATION
            }
        }
    }

    return out
}

func (t *ledgerReader) QueryBlock(in *pb.BlockID) *pb.Block {
    out := &pb.Block{
        Header: in.Header,
        Bcname: in.Bcname,
    }
    out.Header.Error = pb.XChainErrorEnum_SUCCESS

    if t.chain.Status() != global.Normal {
        out.Header.Error = pb.XChainErrorEnum_CONNECT_REFUSE
        t.log.Warn("refused query block due to chain status not ready", "logid", in.Header.Logid)
        return out
    }

    block, err := t.ledger.QueryBlock(in.Blockid)
    if err != nil {
        switch err {
        case ledger.ErrBlockNotExist:
            out.Header.Error = pb.XChainErrorEnum_SUCCESS
            out.Status = pb.Block_NOEXIST
            return out
        default:
            t.log.Warn("ledger query block error", "logid", in.Header.Logid, "error", err)
            out.Header.Error = pb.XChainErrorEnum_UNKNOW_ERROR
            return out
        }
    } else {
        t.log.Debug("need content", "logid", in.Header.Logid, "needContent", in.NeedContent)
        if in.NeedContent {
            out.Block = block
        }
        if block.InTrunk {
            out.Status = pb.Block_TRUNK
        } else {
            out.Status = pb.Block_BRANCH
        }
    }
    return out
}

func (t *ledgerReader) GetBlockChainStatus(in *pb.BCStatus, viewOption pb.ViewOption) *pb.BCStatus {
    if in.GetHeader() == nil {
        in.Header = global.GHeader()
    }
    out := &pb.BCStatus{
        Header: in.Header,
        Bcname: in.Bcname,
    }
    out.Header.Error = pb.XChainErrorEnum_SUCCESS

    if t.chain.Status() != global.Normal {
        out.Header.Error = pb.XChainErrorEnum_CONNECT_REFUSE
        t.log.Warn("refused a connection a function call GetBlock", "logid", in.Header.Logid)
        return out
    }

    meta := t.ledger.GetMeta()
    if viewOption == pb.ViewOption_NONE || viewOption == pb.ViewOption_LEDGER || viewOption == pb.ViewOption_BRANCHINFO {
        out.Meta = meta
    }
    if viewOption == pb.ViewOption_NONE {
        block, err := t.ledger.QueryBlock(meta.TipBlockid)
        if err != nil {
            t.log.Warn("query block error", "logid", in.Header.Logid, "error", err, "blockId", meta.TipBlockid)
            out.Header.Error = HandlerLedgerError(err)
            return out
        }
        out.Block = block
    }
    if viewOption == pb.ViewOption_NONE || viewOption == pb.ViewOption_BRANCHINFO {
        // fetch all branches info
        branchManager, err := t.ledger.GetBranchInfo([]byte("0"), int64(0))
        if err != nil {
            t.log.Warn("query branch error", "logid", in.Header.Logid, "error", err)
            out.Header.Error = HandlerLedgerError(err)
            return out
        }
        for _, branchID := range branchManager {
            out.BranchBlockid = append(out.BranchBlockid, fmt.Sprintf("%x", branchID))
        }
    }
    if viewOption == pb.ViewOption_NONE || viewOption == pb.ViewOption_UTXOINFO {
        utxoMeta := t.ctx.State.GetMeta()
        out.UtxoMeta = utxoMeta
    }

    return out
}

func (t *ledgerReader) GetTipStatus(in *pb.BCStatus) *pb.BCTipStatus {
    out := &pb.BCTipStatus{Header: global.GHeader()}
    out.Header.Error = pb.XChainErrorEnum_SUCCESS
    meta := t.ledger.GetMeta()
    if string(in.Block.GetBlockid()) == string(meta.TipBlockid) {
        out.IsTrunkTip = true
    } else {
        out.IsTrunkTip = false
    }
    return out
}

func (t *ledgerReader) QueryBlockByHeight(in *pb.BlockHeight) *pb.Block {
    out := &pb.Block{
        Header: in.Header,
        Bcname: in.Bcname,
    }
    out.Header.Error = pb.XChainErrorEnum_SUCCESS
    if t.chain.Status() != global.Normal {
        out.Header.Error = pb.XChainErrorEnum_CONNECT_REFUSE
        t.log.Debug("refused a connection a function call GetBlock", "logid", in.Header.Logid)
        return out
    }

    block, err := t.ledger.QueryBlockByHeight(in.Height)
    if err != nil {
        switch err {
        case ledger.ErrBlockNotExist:
            out.Header.Error = pb.XChainErrorEnum_SUCCESS
            out.Status = pb.Block_NOEXIST
            return out
        default:
            t.log.Warn("get block by height error", "logid", in.Header.Logid, "error", err)
            out.Header.Error = pb.XChainErrorEnum_UNKNOW_ERROR
            return out
        }
    }

    out.Block = block
    if block.InTrunk {
        out.Status = pb.Block_TRUNK
    } else {
        out.Status = pb.Block_BRANCH
    }
    return out
}
