package reader

import (
	"bytes"
	"fmt"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/xpb"
	"github.com/xuperchain/xupercore/lib/logs"
)

type ChainReader interface {
	// 获取链状态 (GetBlockChainStatus)
	GetChainStatus() (*xpb.ChainStatus, error)
	// 检查是否是主干Tip Block (ConfirmBlockChainStatus)
	IsTrunkTipBlock(blkId []byte) (bool, error)
	// 获取系统状态
	GetSystemStatus() (*xpb.SystemStatus, error)
	// 获取节点NetUR
	GetNetURL() (string, error)
	// 获取共识状态
	GetConsensusStatus() (*xpb.ConsensusStatus, error)
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

func (t *chainReader) GetChainStatus() (*xpb.ChainStatus, error) {
	chainStatus := &xpb.ChainStatus{}
	chainStatus.LedgerMeta = t.chainCtx.Ledger.GetMeta()
	chainStatus.UtxoMeta = t.chainCtx.State.GetMeta()
	branchIds, err := t.chainCtx.Ledger.GetBranchInfo([]byte("0"), int64(0))
	if err != nil {
		t.log.Warn("get branch info error", "err", err)
		return nil, common.ErrChainStatus
	}

	tipBlockId := chainStatus.LedgerMeta.TipBlockid
	chainStatus.Block, err = t.chainCtx.Ledger.QueryBlock(tipBlockId)
	if err != nil {
		t.log.Warn("query block error", "err", err, "blockId", tipBlockId)
		return nil, common.ErrBlockNotExist
	}

	chainStatus.BranchIds = make([]string, len(branchIds))
	for i, branchId := range branchIds {
		chainStatus.BranchIds[i] = fmt.Sprintf("%x", branchId)
	}

	return chainStatus, nil
}

func (t *chainReader) GetConsensusStatus() (*xpb.ConsensusStatus, error) {
	consensus, err := t.chainCtx.Consensus.GetConsensusStatus()
	if err != nil {
		t.log.Warn("get consensus info error", "err", err)
		return nil, common.ErrConsensusStatus
	}
	status := &xpb.ConsensusStatus{
		Version:        fmt.Sprint(consensus.GetVersion()),
		ConsensusName:  consensus.GetConsensusName(),
		StartHeight:    fmt.Sprint(consensus.GetConsensusBeginInfo()),
		ValidatorsInfo: string(consensus.GetCurrentValidatorsInfo()),
	}
	return status, nil
}

func (t *chainReader) IsTrunkTipBlock(blkId []byte) (bool, error) {
	meta := t.chainCtx.Ledger.GetMeta()
	if bytes.Equal(meta.TipBlockid, blkId) {
		return true, nil
	}

	return false, nil
}

func (t *chainReader) GetSystemStatus() (*xpb.SystemStatus, error) {
	systemStatus := &xpb.SystemStatus{}

	chainStatus, err := t.GetChainStatus()
	if err != nil {
		t.log.Warn("get chain status error", "err", err)
		return nil, common.ErrChainStatus
	}
	systemStatus.ChainStatus = chainStatus

	peerInfo := t.chainCtx.EngCtx.Net.PeerInfo()
	peerUrls := make([]string, len(peerInfo.Peer))
	for i, peer := range peerInfo.Peer {
		peerUrls[i] = peer.Address
	}
	systemStatus.PeerUrls = peerUrls

	return systemStatus, nil
}

func (t *chainReader) GetNetURL() (string, error) {
	peerInfo := t.chainCtx.EngCtx.Net.PeerInfo()
	return peerInfo.Address, nil
}
