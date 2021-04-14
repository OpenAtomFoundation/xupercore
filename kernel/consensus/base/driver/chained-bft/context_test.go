package chained_bft

import (
	"bytes"
	"container/list"
	"testing"
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
		Genesis:    rootNode,
		Root:       rootNode,
		HighQC:     rootNode,
		CommitQC:   rootNode,
		OrphanList: list.New(),
		OrphanMap:  make(map[string]bool),
		Log:        NewFakeLogger("nodeA"),
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
	QC1 := CreateQC([]byte{1}, 1, []byte{0}, 0)
	node1 := &ProposalNode{
		In: QC1,
	}
	if err := tree.updateQcStatus(node1); err != nil {
		t.Error("TestUpdateQcStatus empty parent error", "err", err)
		return nil
	}
	if len(tree.Root.Sons) == 0 {
		t.Error("TestUpdateQcStatus add son error")
		return nil
	}
	QC12 := CreateQC([]byte{2}, 1, []byte{0}, 0)
	node12 := CreateNode(QC12)
	if err := tree.updateQcStatus(node12); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return nil
	}
	if len(tree.Root.Sons) != 2 {
		t.Error("TestUpdateQcStatus add son error", "len", len(tree.Root.Sons))
		return nil
	}
	QC2 := CreateQC([]byte{3}, 2, []byte{1}, 1)
	node2 := CreateNode(QC2)
	if err := tree.updateQcStatus(node2); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return nil
	}
	if len(node1.Sons) != 1 {
		t.Error("TestUpdateQcStatus add son error", "node1", node1.In.GetProposalId(), node1.Sons[0].In.GetProposalId(), node1.Sons[1].In.GetProposalId())
		return nil
	}
	return tree
}

func CreateQC(id []byte, view int64, parent []byte, parentView int64) *QuorumCert {
	return &QuorumCert{
		VoteInfo: &VoteInfo{
			ProposalId:   id,
			ProposalView: view,
			ParentId:     parent,
			ParentView:   parentView,
		},
		LedgerCommitInfo: &LedgerCommitInfo{
			CommitStateId: id,
		},
	}
}

func CreateNode(inQC *QuorumCert) *ProposalNode {
	return &ProposalNode{
		In: inQC,
	}
}

func TestUpdateHighQC(t *testing.T) {
	tree := prepareTree(t)
	id2 := []byte{3}
	tree.updateHighQC(id2)
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

func TestEnforceUpdateHighQC(t *testing.T) {
	tree := prepareTree(t)
	tree.updateHighQC([]byte{3})
	tree.enforceUpdateHighQC([]byte{1})
	if tree.GetHighQC().In.GetProposalView() != 1 {
		t.Error("enforceUpdateHighQC update highQC error", "height", tree.GetHighQC().In.GetProposalView())
		return
	}
}

// TestUpdateCommit Tree如下
// root 0
// |    \
// node1 1 node12
// |
// node2 2
// |
// node3 3
// |
// node4 4
func TestUpdateCommit(t *testing.T) {
	tree := prepareTree(t)
	tree.updateHighQC([]byte{3})

	QC3 := CreateQC([]byte{4}, 3, []byte{3}, 2)
	node1 := &ProposalNode{
		In: QC3,
	}
	if err := tree.updateQcStatus(node1); err != nil {
		t.Error("updateQcStatus error")
		return
	}
	QC4 := CreateQC([]byte{5}, 4, []byte{4}, 3)
	node2 := &ProposalNode{
		In: QC4,
	}
	if err := tree.updateQcStatus(node2); err != nil {
		t.Error("updateQcStatus node2 error")
		return
	}
	tree.updateCommit([]byte{5})
	if tree.Root.In.GetProposalView() != 1 {
		t.Error("updateCommit error")
	}
}

// TestDFSQueryNode Tree如下
//           --------------------root ([]byte{0}, 0)-----------------------
//            		  |       |                          |
// (([]byte{1}, 1)) node1 node12 ([]byte{2}, 1) orphan4<[]byte{10}, 1>
//                    |                                  |            \
//  ([]byte{3}, 2)  node2				        orphan2<[]byte{30}, 2> orphan3<[]byte{35}, 2>
//														 |
// 												orphan1<[]byte{40}, 3>
//
func TestInsertOrphan(t *testing.T) {
	tree := prepareTree(t)
	orphan1 := &ProposalNode{
		In: CreateQC([]byte{40}, 3, []byte{30}, 2),
	}
	tree.updateQcStatus(orphan1)
	e1 := tree.OrphanList.Front()
	o1, ok := e1.Value.(*ProposalNode)
	if !ok {
		t.Error("OrphanList type error1!")
	}
	if o1.In.GetProposalView() != 3 {
		t.Error("OrphanList insert error!")
	}
	orphan2 := &ProposalNode{
		In: CreateQC([]byte{30}, 2, []byte{10}, 1),
	}
	tree.updateQcStatus(orphan2)
	e1 = tree.OrphanList.Front()
	o1, ok = e1.Value.(*ProposalNode)
	if !ok {
		t.Error("OrphanList type error2!")
	}
	if !bytes.Equal(o1.In.GetProposalId(), []byte{30}) {
		t.Error("OrphanList insert error2!", "id", o1.In.GetProposalId())
	}
	orphan3 := &ProposalNode{
		In: CreateQC([]byte{35}, 2, []byte{10}, 1),
	}
	tree.updateQcStatus(orphan3)
	e1 = tree.OrphanList.Front()
	o1, _ = e1.Value.(*ProposalNode)
	e2 := e1.Next()
	o2, _ := e2.Value.(*ProposalNode)
	if !bytes.Equal(o1.In.GetProposalId(), []byte{30}) {
		t.Error("OrphanList insert error3!", "id", o1.In.GetProposalId())
	}
	if !bytes.Equal(o2.In.GetProposalId(), []byte{35}) {
		t.Error("OrphanList insert error4!", "id", o1.In.GetProposalId())
	}
	orphan4 := &ProposalNode{
		In: CreateQC([]byte{10}, 1, []byte{0}, 0),
	}
	tree.updateQcStatus(orphan4)
	if tree.OrphanList.Len() != 0 {
		t.Error("OrphanList adopt error!")
	}
}
