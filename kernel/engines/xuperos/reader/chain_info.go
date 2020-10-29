package reader

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/def"
)

type ChainInfo interface {
	ChainState()
}

type ChainInfoReader struct {
	engine def.Engine
}

func NewChainInfoReader(engine def.Engine) (ChainInfo, error) {
	if engine == nil {
		return nil, fmt.Errorf("new chain info reader failed because param error")
	}

	reader := &ChainInfoReader{
		engine: engine,
	}

	return reader, nil
}

func (t *ChainInfoReader) ChainState() {

}
