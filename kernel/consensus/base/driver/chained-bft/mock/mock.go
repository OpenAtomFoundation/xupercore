package mock

import (
	"os"
	"path/filepath"
	"testing"

	chainedBftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	"github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/storage"
	"github.com/xuperchain/xupercore/kernel/mock"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	LogPath  = filepath.Join(utils.GetCurFileDir(), "../main/test")
	NodePath = filepath.Join(utils.GetCurFileDir(), "../../../../../mock/p2pv2")
)

type TestHelper struct {
	basedir string
	Log     logs.Logger
}

func NewTestHelper() (*TestHelper, error) {
	basedir, err := os.MkdirTemp("", "chainedBFT-test")
	if err != nil {
		panic(err)
	}
	// log实例
	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(basedir, "/log")
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), logPath)
	log, _ := logs.NewLogger("", "asyncworker")

	th := &TestHelper{
		basedir: basedir,
		Log:     log,
	}
	return th, nil
}

func (th *TestHelper) Close() {
	os.RemoveAll(th.basedir)
}

func MockInitQcTree() *storage.QCPendingTree {
	initQC := &storage.QuorumCert{
		VoteInfo: &storage.VoteInfo{
			ProposalId:   []byte{0},
			ProposalView: 0,
		},
		LedgerCommitInfo: &storage.LedgerCommitInfo{
			CommitStateId: []byte{0},
		},
	}
	rootNode := &storage.ProposalNode{
		In: initQC,
	}
	th, _ := NewTestHelper()
	defer th.Close()
	return storage.MockTree(rootNode, rootNode, rootNode, nil, nil, rootNode, th.Log)
}

func MockCreateQC(id []byte, view int64, parent []byte, parentView int64) storage.QuorumCertInterface {
	return storage.NewQuorumCert(
		&storage.VoteInfo{
			ProposalId:   id,
			ProposalView: view,
			ParentId:     parent,
			ParentView:   parentView,
		},
		&storage.LedgerCommitInfo{
			CommitStateId: id,
		}, nil)
}

func MockCreateNode(inQC storage.QuorumCertInterface, signs []*chainedBftPb.QuorumCertSign) *storage.ProposalNode {
	return &storage.ProposalNode{
		In: storage.NewQuorumCert(
			&storage.VoteInfo{
				ProposalId:   inQC.GetProposalId(),
				ProposalView: inQC.GetProposalView(),
				ParentId:     inQC.GetParentProposalId(),
				ParentView:   inQC.GetParentView(),
			}, nil, signs),
	}
}

// TestDFSQueryNode Tree如下
// root
// |    \
// node1 node12
// |
// node2
func PrepareTree(t *testing.T) *storage.QCPendingTree {
	tree := MockInitQcTree()
	QC1 := MockCreateQC([]byte{1}, 1, []byte{0}, 0)
	node1 := &storage.ProposalNode{
		In: QC1,
	}
	if err := tree.UpdateQcStatus(node1); err != nil {
		t.Error("TestUpdateQcStatus empty parent error", "err", err)
		return nil
	}
	QC12 := MockCreateQC([]byte{2}, 1, []byte{0}, 0)
	node12 := MockCreateNode(QC12, nil)
	if err := tree.UpdateQcStatus(node12); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return nil
	}
	QC2 := MockCreateQC([]byte{3}, 2, []byte{1}, 1)
	node2 := MockCreateNode(QC2, nil)
	if err := tree.UpdateQcStatus(node2); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return nil
	}
	if len(node1.Sons) != 1 {
		t.Error("TestUpdateQcStatus add son error", "node1", node1.In.GetProposalId(), node1.Sons[0].In.GetProposalId(), node1.Sons[1].In.GetProposalId())
		return nil
	}
	return tree
}
