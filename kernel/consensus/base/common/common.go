package utils

import (
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

// AddressEqual 判断两个validators地址是否相等
func AddressEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, _ := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// initQCTree 创建了smr需要的QC树存储，该Tree存储了目前待commit的QC信息
func InitQCTree(startHeight int64, ledger cctx.LedgerRely) *chainedBft.QCPendingTree {
	// 初始状态，应该是start高度的前一个区块为genesisQC
	b, _ := ledger.QueryBlockByHeight(startHeight - 1)
	initQC := &chainedBft.QuorumCert{
		VoteInfo: &chainedBft.VoteInfo{
			ProposalId:   b.GetBlockid(),
			ProposalView: startHeight - 1,
		},
		LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
			CommitStateId: b.GetBlockid(),
		},
	}
	rootNode := &chainedBft.ProposalNode{
		In: initQC,
	}
	return &chainedBft.QCPendingTree{
		Genesis:  rootNode,
		Root:     rootNode,
		HighQC:   rootNode,
		CommitQC: rootNode,
	}
}
