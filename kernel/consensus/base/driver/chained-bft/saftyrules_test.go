package chained_bft

import (
	"testing"

	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	chainedBftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
)

func TestCalVotesThreshold(t *testing.T) {
	s := DefaultSaftyRules{}
	sum := 3
	if s.CalVotesThreshold(1, sum) {
		t.Error("TestCalVotesThreshold error 1")
	}
	sum = 4
	if !s.CalVotesThreshold(3, sum) {
		t.Error("TestCalVotesThreshold error 2")
	}
	if s.CalVotesThreshold(0, sum) {
		t.Error("TestCalVotesThreshold error 0")
	}

}

func TestCheckPacemaker(t *testing.T) {
	s := &DefaultSaftyRules{}
	if !s.CheckPacemaker(5, 4) {
		t.Error("CheckPacemaker error")
		return
	}
	if s.CheckPacemaker(1, 5) {
		t.Error("CheckPacemaker error 2")
	}
}

func TestIsInSlice(t *testing.T) {
	s := []string{"a", "b", "c"}
	if !isInSlice("a", s) {
		t.Error("isInSlice error")
		return
	}
	if isInSlice("d", s) {
		t.Error("isInSlice error")
	}
}

func TestCheckProposal(t *testing.T) {
	s := &DefaultSaftyRules{
		lastVoteRound:  0,
		preferredRound: 0,
		QcTree:         initQcTree(),
		Log:            NewFakeLogger("nodeA"),
	}
	a, cc := NewFakeCryptoClient("nodeA", t)
	s.Crypto = &cCrypto.CBFTCrypto{
		Address:      &a,
		CryptoClient: cc,
	}
	generic := CreateQC([]byte{1}, 1, []byte{0}, 01)
	msg := &chainedBftPb.ProposalMsg{
		ProposalView: 1,
		ProposalId:   []byte{1},
	}
	r, err := s.Crypto.SignProposalMsg(msg)
	if err != nil {
		t.Error("SignProposalMsg error")
		return
	}
	generic.SignInfos = []*chainedBftPb.QuorumCertSign{r.Sign}
	node1 := &ProposalNode{
		In: generic,
	}
	if err := s.QcTree.updateQcStatus(node1); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return
	}
	proposal := CreateQC([]byte{2}, 2, []byte{1}, 1)
	proposal.VoteInfo.ParentId = []byte{1}
	s.CheckProposal(proposal, generic, []string{"gNhga8vLc4JcmoHB2yeef2adBhntkc5d1"})
	s.VoteProposal([]byte{2}, 2, generic)
	s.CheckVote(generic, "123", []string{"gNhga8vLc4JcmoHB2yeef2adBhntkc5d1"})
}

func TestCheckVote(t *testing.T) {

}
