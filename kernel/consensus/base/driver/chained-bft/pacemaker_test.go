package chained_bft

import (
	"testing"

	"github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/storage"
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
	_, err := p.AdvanceView(qc)
	if err != nil {
		t.Fatal(err)
	}
	if qc.GetProposalView() != 1 {
		t.Fatal("AdvanceView error.")
	}
	if p.GetCurrentView() != 2 {
		t.Error("GetCurrentView error.")
	}
}
