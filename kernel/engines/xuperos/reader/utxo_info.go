package reader

import (
    "github.com/xuperchain/xuperchain/core/global"
    "github.com/xuperchain/xuperchain/core/pb"
    "github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
    "github.com/xuperchain/xupercore/lib/logs"
)

type State interface {
    QueryAccountACL(account string) (*pb.Acl, bool, error)
    QueryContractMethodACL(contract string, method string) (*pb.Acl, bool, error)
    QueryAccountContainAK(address string) ([]string, error)

    QueryUtxoRecord(account string, displayCount int64) (*pb.UtxoRecordDetail, error)
    QueryTxFromForbidden(txid []byte) bool

    GetBalance(address string) (string, error)
    GetFrozenBalance(address string) (string, error)
    GetBalanceDetail(address string) (*pb.TokenFrozenDetails, error)

    QueryContractStatData() (*pb.ContractStatDataResponse, error)
    GetAccountContractsStatus(account string, needContent bool) ([]*pb.ContractStatus, error)
}

type stateReader struct {
    ctx    *def.ChainCtx
    log    logs.Logger
    chain  def.Chain
    state  def.XState
}

func NewStateReader(chain def.Chain) State {
    reader := &stateReader{
        ctx:   chain.Context(),
        log:   chain.Context().XLog,
        state: chain.Context().State,
        chain: chain,
    }

    return reader
}

func (t *stateReader) QueryAccountACL(account string) (*pb.Acl, bool, error) {
    if t.chain.Status() != global.Normal {
        return nil, false, def.ErrBlockChainNotReady
    }

    return t.state.QueryAccountACLWithConfirmed(account)
}

func (t *stateReader) QueryContractMethodACL(contract string, method string) (*pb.Acl, bool, error) {
    if t.chain.Status() != global.Normal {
        return nil, false, def.ErrBlockChainNotReady
    }

    return t.state.QueryContractMethodACLWithConfirmed(contract, method)
}

func (t *stateReader) QueryAccountContainAK(address string) ([]string, error) {
    if t.chain.Status() != global.Normal {
        return nil, def.ErrBlockChainNotReady
    }

    return t.state.QueryAccountContainAK(address)
}

func (t *stateReader) QueryContractStatData() (*pb.ContractStatDataResponse, error) {
    out := &pb.ContractStatDataResponse{
        Header: global.GHeader(),
        Bcname: t.ctx.BCName,
    }

    if t.chain.Status() != global.Normal {
        return out, def.ErrBlockChainNotReady
    }

    data, err := t.state.QueryContractStatData()
    if err != nil {
        return out, err
    }

    out.Data = data
    return out, nil
}

func (t *stateReader) QueryUtxoRecord(account string, displayCount int64) (*pb.UtxoRecordDetail, error) {
    defaultUtxoRecord := &pb.UtxoRecordDetail{Header: &pb.Header{}}
    if t.chain.Status() != global.Normal {
        return defaultUtxoRecord, def.ErrBlockChainNotReady
    }

    utxoRecord, err := t.state.QueryUtxoRecord(account, displayCount)
    if err != nil {
        return defaultUtxoRecord, err
    }

    return utxoRecord, nil
}

func (t *stateReader) QueryTxFromForbidden(txid []byte) bool {
    if t.chain.Status() != global.Normal {
        return false
    }

    exist, confirmed, _ := t.state.QueryTxFromForbiddenWithConfirmed(txid)
    // only forbid exist && confirmed transaction
    if exist && confirmed {
        return true
    }

    return false
}

func (t *stateReader) GetBalance(address string) (string, error) {
    if t.chain.Status() != global.Normal {
        return "", def.ErrBlockChainNotReady
    }

    balance, err := t.state.GetBalance(address)
    if err != nil {
        return "", err
    }

    return balance.String(), nil
}

func (t *stateReader) GetFrozenBalance(address string) (string, error) {
    if t.chain.Status() != global.Normal {
        return "", def.ErrBlockChainNotReady
    }

    balance, err := t.state.GetFrozenBalance(address)
    if err != nil {
        return "", err
    }

    return balance.String(), nil
}

func (t *stateReader) GetBalanceDetail(address string) (*pb.TokenFrozenDetails, error) {
    if t.chain.Status() != global.Normal {
        return nil, def.ErrBlockChainNotReady
    }

    tokenDetails, err := t.state.GetBalanceDetail(address)
    if err != nil {
        return nil, err
    }

    tokenFrozenDetails := &pb.TokenFrozenDetails{
        Bcname: t.ctx.BCName,
        Tfd:    tokenDetails,
    }

    return tokenFrozenDetails, nil
}

func (t *stateReader) GetAccountContractsStatus(account string, needContent bool) ([]*pb.ContractStatus, error) {
    contracts, err := t.state.GetAccountContracts(account)
    if err != nil {
        t.log.Warn("get account contracts error", "error", err)
        return nil, err
    }

    out := make([]*pb.ContractStatus, 0, len(contracts))
    for _, v := range contracts {
        if !needContent {
            cs := &pb.ContractStatus{
                ContractName: v,
            }
            out = append(out, cs)
        } else {
            contractStatus, err := t.state.GetContractStatus(v)
            if err != nil {
                return nil, err
            }
            out = append(out, contractStatus)
        }
    }

    return out, nil
}
