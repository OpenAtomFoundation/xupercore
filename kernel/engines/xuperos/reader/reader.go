package reader

import "github.com/xuperchain/xupercore/kernel/engines/xuperos/def"

type Reader interface {
    Chain
    Ledger
    State
    Consensus
}

type reader struct {
    Chain
    Ledger
    State
    Consensus
}

func NewReader(chain def.Chain) Reader {
    r := &reader{
        Chain: NewChainReader(chain),
        Ledger: NewLedgerReader(chain),
        State: NewStateReader(chain),
        Consensus: NewConsensusReader(chain),
    }
    return r
}
