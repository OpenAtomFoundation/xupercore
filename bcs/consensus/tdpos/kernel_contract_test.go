package tdpos

import (
	"testing"

	"github.com/xuperchain/xupercore/kernel/consensus/mock"
)

func TestIsAuthAddress(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	tdpos, _ := i.(*tdposConsensus)
	if !tdpos.isAuthAddress("A", "A", []string{"B", "C"}) {
		t.Error("isAuthAddress error1.")
		return
	}
	if !tdpos.isAuthAddress("B", "A", []string{"B", "C"}) {
		t.Error("isAuthAddress error2.")
	}
}

func NewNominateArgs() map[string][]byte {
	a := make(map[string][]byte)
	a["candidate"] = []byte(`"akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"`)
	a["amount"] = []byte("1")
	return a
}

func NewM() map[string]map[string][]byte {
	a := make(map[string]map[string][]byte)
	return a
}

func TestRunNominateCandidate(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	tdpos, _ := i.(*tdposConsensus)
	fakeCtx := mock.NewFakeKContext(NewNominateArgs(), NewM())
	_, err = tdpos.runNominateCandidate(fakeCtx)
	if err == nil {
		t.Error("runNominateCandidate error1.")
		return
	}
}

func TestRunRevokeCandidate(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	tdpos, _ := i.(*tdposConsensus)
	fakeCtx := mock.NewFakeKContext(NewNominateArgs(), NewM())
	_, err = tdpos.runRevokeCandidate(fakeCtx)
	if err == nil {
		t.Error("runRevokeCandidate error1.")
		return
	}
}

func TestRunVote(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	tdpos, _ := i.(*tdposConsensus)
	fakeCtx := mock.NewFakeKContext(NewNominateArgs(), NewM())
	_, err = tdpos.runVote(fakeCtx)
	if err == nil {
		t.Error("runVote error1.")
		return
	}
}
func TestRunRevokeVote(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	tdpos, _ := i.(*tdposConsensus)
	fakeCtx := mock.NewFakeKContext(NewNominateArgs(), NewM())
	_, err = tdpos.runRevokeVote(fakeCtx)
	if err == nil {
		t.Error("runRevokeVote error1.")
		return
	}
}

func TestRunGetTdposInfos(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	tdpos, _ := i.(*tdposConsensus)
	fakeCtx := mock.NewFakeKContext(NewNominateArgs(), NewM())
	_, err = tdpos.runGetTdposInfos(fakeCtx)
	if err != nil {
		t.Error("runGetTdposInfos error1.")
		return
	}
}
