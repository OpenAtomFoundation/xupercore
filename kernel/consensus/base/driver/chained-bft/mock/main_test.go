package mock

import (
	"bytes"
	"testing"

	"github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/storage"
)

func TestUpdateHighQC(t *testing.T) {
	tree := PrepareTree(t)
	id2 := []byte{3}
	tree.UpdateHighQC(id2)
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
	tree := PrepareTree(t)
	tree.UpdateHighQC([]byte{3})
	tree.EnforceUpdateHighQC([]byte{1})
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
	tree := PrepareTree(t)
	tree.UpdateHighQC([]byte{3})

	QC3 := MockCreateQC([]byte{4}, 3, []byte{3}, 2)
	node1 := &storage.ProposalNode{
		In: QC3,
	}
	if err := tree.UpdateQcStatus(node1); err != nil {
		t.Error("updateQcStatus error")
		return
	}
	QC4 := MockCreateQC([]byte{5}, 4, []byte{4}, 3)
	node2 := &storage.ProposalNode{
		In: QC4,
	}
	if err := tree.UpdateQcStatus(node2); err != nil {
		t.Error("updateQcStatus node2 error")
		return
	}
	tree.UpdateCommit([]byte{5})
	if tree.GetRootQC().In.GetProposalView() != 1 {
		t.Error("updateCommit error")
	}
}

/*
TestDFSQueryNode Tree如下

		--------------------root ([]byte{0}, 0)-----------------------
		                  |       |                          |
	   (([]byte{1}, 1)) node1   node12 ([]byte{2}, 1) orphan4<[]byte{10}, 1>
	                      |                                  |               \
	    ([]byte{3}, 2)  node2				          orphan2<[]byte{30}, 2>  orphan3<[]byte{35}, 2>
															 |
													  orphan1<[]byte{40}, 3>
*/
func TestInsertOrphan(t *testing.T) {
	tree := PrepareTree(t)
	orphan1 := &storage.ProposalNode{
		In: MockCreateQC([]byte{40}, 3, []byte{30}, 2),
	}
	tree.UpdateQcStatus(orphan1)
	orphan := tree.MockGetOrphan()
	e1 := orphan.Front()
	o1, ok := e1.Value.(*storage.ProposalNode)
	if !ok {
		t.Error("OrphanList type error1!")
	}
	if o1.In.GetProposalView() != 3 {
		t.Error("OrphanList insert error!")
	}
	orphan2 := &storage.ProposalNode{
		In: MockCreateQC([]byte{30}, 2, []byte{10}, 1),
	}
	tree.UpdateQcStatus(orphan2)
	e1 = orphan.Front()
	o1, ok = e1.Value.(*storage.ProposalNode)
	if !ok {
		t.Error("OrphanList type error2!")
	}
	if !bytes.Equal(o1.In.GetProposalId(), []byte{30}) {
		t.Error("OrphanList insert error2!", "id", o1.In.GetProposalId())
	}
	orphan3 := &storage.ProposalNode{
		In: MockCreateQC([]byte{35}, 2, []byte{10}, 1),
	}
	tree.UpdateQcStatus(orphan3)
	e1 = orphan.Front()
	o1, _ = e1.Value.(*storage.ProposalNode)
	e2 := e1.Next()
	o2, _ := e2.Value.(*storage.ProposalNode)
	if !bytes.Equal(o1.In.GetProposalId(), []byte{30}) {
		t.Error("OrphanList insert error3!", "id", o1.In.GetProposalId())
	}
	if !bytes.Equal(o2.In.GetProposalId(), []byte{35}) {
		t.Error("OrphanList insert error4!", "id", o1.In.GetProposalId())
	}
	orphan4 := &storage.ProposalNode{
		In: MockCreateQC([]byte{10}, 1, []byte{0}, 0),
	}
	tree.UpdateQcStatus(orphan4)
	if orphan.Len() != 0 {
		t.Error("OrphanList adopt error!")
	}
}
