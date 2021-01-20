package utils

import (
	"errors"

	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/lib/logs"
)

var (
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
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
func InitQCTree(ledger cctx.LedgerRely, log logs.Logger) *chainedBft.QCPendingTree {
	// 初始状态，应该是start高度的前一个区块为genesisQC, 或者是重启后的tip作为rootQC
	r := ledger.GetTipBlock()
	rQC := &chainedBft.QuorumCert{
		VoteInfo: &chainedBft.VoteInfo{
			ProposalId:   r.GetBlockid(),
			ProposalView: r.GetHeight(),
		},
		LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
			CommitStateId: r.GetBlockid(),
		},
	}
	rNode := &chainedBft.ProposalNode{
		In: rQC,
	}
	return &chainedBft.QCPendingTree{
		Genesis:  rNode,
		Root:     rNode,
		HighQC:   rNode,
		CommitQC: rNode,
		Log:      log,
	}
}
