package reader

import (
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/pb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
)

type ChainStatus struct {
	LedgerMeta *lpb.LedgerMeta
	TipBlock   *lpb.InternalBlock
	UtxoMeta   *lpb.UtxoMeta
}

type SystemStatus struct {
	ChainStatus *ChainStatus
	PeerUrls    []string
}

type ChainReader interface {
	// 获取链状态 (GetBlockChainStatus)
	GetChainStatus() (*ChainStatus, error)
	// 检查是否是主干Tip Block (ConfirmBlockChainStatus)
	IsTrunkTipBlock(blkId []byte) (bool, error)
	// 获取系统状态
	GetSystemStatus() (*ChainStatus, error)
	// 获取节点NetUR
	GetNetURL() (string, error)
}

type chainReader struct {
	chainCtx *common.ChainCtx
	baseCtx  xctx.XContext
	log      logs.Logger
}

func NewChainReader(chainCtx *common.ChainCtx, baseCtx xctx.XContext) ChainReader {
	if chainCtx == nil || baseCtx == nil {
		return nil
	}

	reader := &chainReader{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
		log:      baseCtx.GetLog(),
	}

	return reader
}

func (t *chainReader) GetChainStatus() (*ChainStatus, error) {
	return nil, nil
}

func (t *chainReader) IsTrunkTipBlock(blkId []byte) (bool, error) {
	return false, nil
}

func (t *chainReader) GetSystemStatus() (*ChainStatus, error) {
	return nil, nil
}

func (t *chainReader) GetNetURL() (string, error) {
	return "", nil
}
