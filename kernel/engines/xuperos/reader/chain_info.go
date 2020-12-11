package reader

import (
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
)

type Chain interface {
	ChainState() int
	ChainMode() string
}

type chainReader struct {
	chain def.Chain
}

func NewChainReader(chain def.Chain) Chain {
	reader := &chainReader{
		chain: chain,
	}

	return reader
}

func (t *chainReader) ChainState() int {
	return t.chain.Status()
}

func (t *chainReader) ChainMode() string {
	return t.chain.Context().EngCfg.NodeMode
}
