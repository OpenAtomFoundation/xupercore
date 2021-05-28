package tdpos

import (
	"encoding/json"
	"testing"

	"github.com/xuperchain/xupercore/kernel/consensus/mock"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
)

var nominate_key = "tdpos_0_nominate"
var vote_prefix = "tdpos_0_vote_"

func TestIsAuthAddress(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
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
	a["candidate"] = []byte(`TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY`)
	a["amount"] = []byte("1")
	a["height"] = []byte("6")
	return a
}

func NewVoteArgs() map[string][]byte {
	a := make(map[string][]byte)
	a["candidate"] = []byte(`akf7qunmeaqb51Wu418d6TyPKp4jdLdpV`)
	a["amount"] = []byte("1")
	a["height"] = []byte("6")
	return a
}

func NewRevokeNominateArgs() map[string][]byte {
	a := make(map[string][]byte)
	a["candidate"] = []byte(`SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co`)
	a["height"] = []byte("6")
	return a
}

func NewM() map[string]map[string][]byte {
	a := make(map[string]map[string][]byte)
	return a
}

func NominateKey2() []byte {
	n := NewNominateValue()
	m2 := make(map[string]int64)
	m2["SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"] = 10
	m2["TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"] = 10
	n["SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"] = m2
	m3 := make(map[string]int64)
	m3["akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"] = 5
	n["akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"] = m3
	nb, err := json.Marshal(&n)
	if err != nil {
		return nil
	}
	return nb
}

func TestRunNominateCandidate(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	// 1. 构造term存储
	l, _ := cCtx.Ledger.(*kmock.FakeLedger)
	l.Put(kmock.NewBlock(3))
	l.Put(kmock.NewBlock(4))
	l.Put(kmock.NewBlock(5))
	l.Put(kmock.NewBlock(6))
	// 2. 整理Block的共识存储
	l.SetConsensusStorage(1, SetTdposStorage(1, nil))
	l.SetConsensusStorage(2, SetTdposStorage(1, nil))
	l.SetConsensusStorage(3, SetTdposStorage(1, nil))
	l.SetConsensusStorage(4, SetTdposStorage(2, nil))
	l.SetConsensusStorage(5, SetTdposStorage(2, nil))
	l.SetConsensusStorage(6, SetTdposStorage(3, nil))
	// 3. 构造nominate存储
	l.SetSnapshot(tdposBucket, []byte(nominate_key), NominateKey2())
	// 4. 构造vote存储
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"), VoteKey1())
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"), VoteKey2())
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"), VoteKey3())

	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
	tdpos, _ := i.(*tdposConsensus)
	fakeCtx := mock.NewFakeKContext(NewNominateArgs(), NewM())
	_, err = tdpos.runNominateCandidate(fakeCtx)
	if err != nil {
		t.Error("runNominateCandidate error1.", "err", err)
		return
	}
}

func TestRunRevokeCandidate(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	// 1. 构造term存储
	l, _ := cCtx.Ledger.(*kmock.FakeLedger)
	l.Put(kmock.NewBlock(3))
	l.Put(kmock.NewBlock(4))
	l.Put(kmock.NewBlock(5))
	l.Put(kmock.NewBlock(6))
	// 2. 整理Block的共识存储
	l.SetConsensusStorage(1, SetTdposStorage(1, nil))
	l.SetConsensusStorage(2, SetTdposStorage(1, nil))
	l.SetConsensusStorage(3, SetTdposStorage(1, nil))
	l.SetConsensusStorage(4, SetTdposStorage(2, nil))
	l.SetConsensusStorage(5, SetTdposStorage(2, nil))
	l.SetConsensusStorage(6, SetTdposStorage(3, nil))
	// 3. 构造nominate存储
	l.SetSnapshot(tdposBucket, []byte(nominate_key), NominateKey2())
	// 4. 构造vote存储
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"), VoteKey1())
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"), VoteKey2())
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"), VoteKey3())

	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
	tdpos, _ := i.(*tdposConsensus)
	fakeCtx := mock.NewFakeKContext(NewRevokeNominateArgs(), NewM())
	_, err = tdpos.runRevokeCandidate(fakeCtx)
	if err != nil {
		t.Error("runRevokeCandidate error1.", "err", err)
		return
	}
}

func TestRunVote(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	// 1. 构造term存储
	l, _ := cCtx.Ledger.(*kmock.FakeLedger)
	l.Put(kmock.NewBlock(3))
	l.Put(kmock.NewBlock(4))
	l.Put(kmock.NewBlock(5))
	l.Put(kmock.NewBlock(6))
	// 2. 整理Block的共识存储
	l.SetConsensusStorage(1, SetTdposStorage(1, nil))
	l.SetConsensusStorage(2, SetTdposStorage(1, nil))
	l.SetConsensusStorage(3, SetTdposStorage(1, nil))
	l.SetConsensusStorage(4, SetTdposStorage(2, nil))
	l.SetConsensusStorage(5, SetTdposStorage(2, nil))
	l.SetConsensusStorage(6, SetTdposStorage(3, nil))
	// 3. 构造nominate存储
	l.SetSnapshot(tdposBucket, []byte(nominate_key), NominateKey2())
	// 4. 构造vote存储
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"), VoteKey1())
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"), VoteKey2())
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"), VoteKey3())

	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
	tdpos, _ := i.(*tdposConsensus)
	fakeCtx := mock.NewFakeKContext(NewVoteArgs(), NewM())
	_, err = tdpos.runVote(fakeCtx)
	if err != nil {
		t.Error("runVote error1.", "err", err)
		return
	}
}
func TestRunRevokeVote(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	// 1. 构造term存储
	l, _ := cCtx.Ledger.(*kmock.FakeLedger)
	l.Put(kmock.NewBlock(3))
	l.Put(kmock.NewBlock(4))
	l.Put(kmock.NewBlock(5))
	l.Put(kmock.NewBlock(6))
	// 2. 整理Block的共识存储
	l.SetConsensusStorage(1, SetTdposStorage(1, nil))
	l.SetConsensusStorage(2, SetTdposStorage(1, nil))
	l.SetConsensusStorage(3, SetTdposStorage(1, nil))
	l.SetConsensusStorage(4, SetTdposStorage(2, nil))
	l.SetConsensusStorage(5, SetTdposStorage(2, nil))
	l.SetConsensusStorage(6, SetTdposStorage(3, nil))
	// 3. 构造nominate存储
	l.SetSnapshot(tdposBucket, []byte(nominate_key), NominateKey2())
	// 4. 构造vote存储
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"), VoteKey1())
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"), VoteKey2())
	l.SetSnapshot(tdposBucket, []byte(vote_prefix+"akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"), VoteKey3())

	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
	tdpos, _ := i.(*tdposConsensus)
	fakeCtx := mock.NewFakeKContext(NewNominateArgs(), NewM())
	_, err = tdpos.runRevokeVote(fakeCtx)
	if err != nil {
		t.Error("runRevokeVote error1.", "err", err)
		return
	}
}
