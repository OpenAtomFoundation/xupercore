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
func InitQCTree(startHeight int64, ledger cctx.LedgerRely, log logs.Logger) *chainedBft.QCPendingTree {
	// 初始状态应该是start高度的前一个区块为genesisQC，即tipBlock
	// 由于重启后对于smr的内存存储(如qcVoteMap)将会清空，因此只能拿tipBlock获取tipHeight - 1高度的签名，重做tipBlock
	g, err := ledger.QueryBlockByHeight(startHeight - 1)
	if err != nil {
		return nil
	}
	gQC := &chainedBft.QuorumCert{
		VoteInfo: &chainedBft.VoteInfo{
			ProposalId:   g.GetBlockid(),
			ProposalView: g.GetHeight(),
		},
		LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
			CommitStateId: g.GetBlockid(),
		},
	}
	gNode := &chainedBft.ProposalNode{
		In: gQC,
	}
	r := ledger.GetTipBlock()
	// 当前为初始状态
	if r.GetHeight() == startHeight-1 {
		return &chainedBft.QCPendingTree{
			Genesis:  gNode,
			Root:     gNode,
			HighQC:   gNode,
			CommitQC: gNode,
			Log:      log,
		}
	}
	// 重启状态时将tipBlock-1装载到HighQC中
	r, err = ledger.QueryBlock(r.GetPreHash())
	if err != nil {
		return nil
	}
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
		Genesis:  gNode,
		Root:     rNode,
		HighQC:   rNode,
		CommitQC: rNode,
		Log:      log,
	}
}
