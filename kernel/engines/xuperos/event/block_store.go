package event

import (
	"fmt"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
)

// ChainManager manage multiple block chain
type ChainManager interface {
	// GetBlockStore get BlockStore base bcname(the name of block chain)
	GetBlockStore(bcname string) (BlockStore, error)
}

// BlockStore is the interface of block store
type BlockStore interface {
	// TipBlockHeight returns the tip block height
	TipBlockHeight() (int64, error)
	// WaitBlockHeight wait until the height of current block height >= target
	WaitBlockHeight(target int64) int64
	// QueryBlockByHeight returns block at given height
	QueryBlockByHeight(int64) (*pb.InternalBlock, error)
}

type chainManager struct {
	engine common.Engine
}

// NewChainManager returns ChainManager as the wrapper of xchaincore.XChainMG
func NewChainManager(engine common.Engine) ChainManager {
	return &chainManager{
		engine: engine,
	}
}

func (c *chainManager) GetBlockStore(bcname string) (BlockStore, error) {
	chain, err := c.engine.Get(bcname)
	if err != nil {
		return nil, fmt.Errorf("chain %s not found", bcname)
	}

	return NewBlockStore(chain.Context().Ledger, chain.Context().State), nil
}

type blockStore struct {
	*ledger.Ledger
	*state.State
}

// NewBlockStore wraps ledger and utxovm as a BlockStore
func NewBlockStore(ledger *ledger.Ledger, state *state.State) BlockStore {
	return &blockStore{
		Ledger: ledger,
		State:  state,
	}
}

func (b *blockStore) TipBlockHeight() (int64, error) {
	tipBlockid := b.Ledger.GetMeta().GetTipBlockid()
	block, err := b.Ledger.QueryBlockHeader(tipBlockid)
	if err != nil {
		return 0, err
	}
	return block.GetHeight(), nil
}
