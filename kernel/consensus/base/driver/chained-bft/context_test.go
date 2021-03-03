package chained_bft

import (
	"testing"

	mock "github.com/xuperchain/xupercore/bcs/consensus/mock"
)

func initQcTree() *QCPendingTree {
	initQC := &QuorumCert{
		VoteInfo: &VoteInfo{
			ProposalId:   []byte{0},
			ProposalView: 0,
		},
		LedgerCommitInfo: &LedgerCommitInfo{
			CommitStateId: []byte{0},
		},
	}
	rootNode := &ProposalNode{
		In: initQC,
	}
	return &QCPendingTree{
		Genesis:  rootNode,
		Root:     rootNode,
		HighQC:   rootNode,
		CommitQC: rootNode,
		Log:      mock.NewFakeLogger(),
	}
}

// TestDFSQueryNode Tree如下
// root
// |    \
// node1 node12
// |
// node2
func prepareTree(t *testing.T) *QCPendingTree {
	tree := initQcTree()
	node := tree.DFSQueryNode([]byte{0})
	if node != tree.Root {
		t.Error("DFSQueryNode root node error")
		return nil
	}
	QC1 := CreateQC([]byte{1}, 1)
	node1 := &ProposalNode{
		In:     QC1,
		Parent: tree.Root,
	}
	if err := tree.updateQcStatus(node1); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return nil
	}
	if len(tree.Root.Sons) == 0 {
		t.Error("TestUpdateQcStatus add son error")
		return nil
	}
	QC12 := CreateQC([]byte{2}, 1)
	node12 := CreateNode(tree.Root, QC12)
	if err := tree.updateQcStatus(node12); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return nil
	}
	if len(tree.Root.Sons) != 2 {
		t.Error("TestUpdateQcStatus add son error", "len", len(tree.Root.Sons))
		return nil
	}
	QC2 := CreateQC([]byte{3}, 2)
	node2 := CreateNode(node1, QC2)
	if err := tree.updateQcStatus(node2); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return nil
	}
	if len(node1.Sons) != 1 {
		t.Error("TestUpdateQcStatus add son error", "len", len(node1.Sons))
		return nil
	}
	return tree
}

func CreateQC(id []byte, view int64) *QuorumCert {
	return &QuorumCert{
		VoteInfo: &VoteInfo{
			ProposalId:   id,
			ProposalView: view,
		},
		LedgerCommitInfo: &LedgerCommitInfo{
			CommitStateId: id,
		},
	}
}

func CreateNode(parentNode *ProposalNode, inQC *QuorumCert) *ProposalNode {
	return &ProposalNode{
		In:     inQC,
		Parent: parentNode,
	}
}

func TestUpdateHighQC(t *testing.T) {
	tree := prepareTree(t)
	tree.updateHighQC([]byte{3})
	if tree.GetHighQC().In.GetProposalView() != 2 {
		t.Error("TestUpdateHighQC update highQC error", "height", tree.GetHighQC().In.GetProposalView())
		return
	}
	if tree.GetGenericQC().In.GetProposalView() != 1 {
		t.Error("TestUpdateHighQC update genericQC error", "height", tree.GetGenericQC().In.GetProposalView())
		return
	}
	if tree.GetLockedQC().In.GetProposalView() != 0 {
		t.Error("TestUpdateHighQC update lockedQC error", "height", tree.GetLockedQC().In.GetProposalView())
		return
	}
}
