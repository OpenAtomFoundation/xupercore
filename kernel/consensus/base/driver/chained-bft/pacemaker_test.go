package chained_bft

import (
	"testing"
)

func TestPaceMaker(t *testing.T) {
	p := &DefaultPaceMaker{
		StartView:   0,
		currentView: 0,
	}
	qc := &QuorumCert{
		VoteInfo: &VoteInfo{
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
