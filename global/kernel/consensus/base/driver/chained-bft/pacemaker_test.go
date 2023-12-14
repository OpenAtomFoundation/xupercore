package chained_bft

import (
	"testing"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/consensus/base/driver/chained-bft/storage"
)

func TestPaceMaker(t *testing.T) {
	p := &DefaultPaceMaker{
		CurrentView: 0,
	}
	qc := &storage.QuorumCert{
		VoteInfo: &storage.VoteInfo{
			ProposalId:   []byte{1},
			ProposalView: 1,
		},
	}
	p.AdvanceView(qc)
	if qc.GetProposalView() != 1 {
		t.Error("AdvanceView error.")
		return
	}
	if p.GetCurrentView() != 2 {
		t.Error("GetCurrentView error.")
	}
}
