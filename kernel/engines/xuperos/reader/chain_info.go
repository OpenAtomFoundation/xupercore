package reader

import (
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
)

type ChainStatus struct {
	LedgerMeta *lpb.LedgerMeta
	UtxoMeta   *lpb.UtxoMeta
	BranchIds  []string
}

type SystemStatus struct {
	ChainStatus *ChainStatus
	PeerUrls    []string
}

type ChainReader interface {
	// 获取链状态 (GetBlockChainStatus)
	GetChainStatus() (*ChainStatus, *common.Error)
	// 检查是否是主干Tip Block (ConfirmBlockChainStatus)
	IsTrunkTipBlock(blkId []byte) (bool, *common.Error)
	// 获取系统状态
	GetSystemStatus() (*ChainStatus, *common.Error)
	// 获取节点NetUR
	GetNetURL() (string, *common.Error)
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

func (t *chainReader) GetChainStatus() (*ChainStatus, *common.Error) {
	return nil, nil
}

func (t *chainReader) IsTrunkTipBlock(blkId []byte) (bool, *common.Error) {
	return false, nil
}

func (t *chainReader) GetSystemStatus() (*ChainStatus, *common.Error) {
	return nil, nil
}

func (t *chainReader) GetNetURL() (string, *common.Error) {
	return "", nil
}
