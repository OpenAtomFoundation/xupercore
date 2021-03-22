package tdpos

import (
	"encoding/json"
	"fmt"
	"testing"

	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
)

func Test(t *testing.T) {
	// map[string]map[string]int64
	resRaw := NewNominateValue()
	testValue := make(map[string]int64)
	testValue["NodeB"] = 1
	resRaw["NodeA"] = testValue
	res, err := json.Marshal(&resRaw)
	if err != nil {
		t.Error("Marshal error ", err)
		return
	}

	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		t.Error("Unmarshal err ", err)
		return
	}
	t.Log("nominateValue: ", nominateValue)
}

func TestNewSchedule(t *testing.T) {
	cStr := getTdposConsensusConf()
	tdposCfg, err := buildConfigs([]byte(cStr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	if s := NewSchedule(tdposCfg, cCtx.XLog, cCtx.Ledger); s == nil {
		t.Error("NewSchedule error.")
	}
}

func TestGetLeader(t *testing.T) {
	cStr := getTdposConsensusConf()
	tdposCfg, err := buildConfigs([]byte(cStr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	s := NewSchedule(tdposCfg, cCtx.XLog, cCtx.Ledger)
	if s == nil {
		t.Error("NewSchedule error.")
	}
	s.GetLeader(3)
}

func TermKey1() []byte {
	t := NewTermValue()
	t = append(t, termItem{
		Validators: []string{"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"},
		Term:       1,
		Height:     1,
	})
	t = append(t, termItem{
		Validators: []string{"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY", "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"},
		Term:       2,
		Height:     4,
	})
	tb, err := json.Marshal(&t)
	if err != nil {
		return nil
	}

	tt := NewTermValue()
	err = json.Unmarshal(tb, &tt)
	fmt.Printf("##### %v %v %v %v", t, tt[0].Validators, tt[1].Validators, tt)
	return tb
}

func NominateKey1() []byte {
	n := NewNominateValue()
	m1 := make(map[string]int64)
	m1["TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"] = 10
	n["TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"] = m1
	m2 := make(map[string]int64)
	m2["SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"] = 10
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

func VoteKey1() []byte {
	v := NewvoteValue()
	v["TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"] = 10
	vb, err := json.Marshal(&v)
	if err != nil {
		return nil
	}
	return vb
}

func VoteKey2() []byte {
	v := NewvoteValue()
	v["SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"] = 5
	vb, err := json.Marshal(&v)
	if err != nil {
		return nil
	}
	return vb
}

func VoteKey3() []byte {
	v := NewvoteValue()
	v["akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"] = 15
	vb, err := json.Marshal(&v)
	if err != nil {
		return nil
	}
	return vb
}

func SetTdposStorage(term int64) []byte {
	s := common.ConsensusStorage{
		CurTerm:     term,
		CurBlockNum: 3,
	}
	b, err := json.Marshal(&s)
	if err != nil {
		return nil
	}
	return b
}

func TestCalHisValidators(t *testing.T) {
	cStr := getTdposConsensusConf()
	tdposCfg, err := buildConfigs([]byte(cStr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	s := NewSchedule(tdposCfg, cCtx.XLog, cCtx.Ledger)
	if s == nil {
		t.Error("NewSchedule error.")
	}
	// 1. 构造term存储
	l, _ := s.ledger.(*kmock.FakeLedger)
	l.Put(kmock.NewBlock(3))
	l.Put(kmock.NewBlock(4))
	l.Put(kmock.NewBlock(5))
	l.Put(kmock.NewBlock(6))
	// 2. 整理Block的共识存储
	l.SetConsensusStorage(1, SetTdposStorage(1))
	l.SetConsensusStorage(2, SetTdposStorage(1))
	l.SetConsensusStorage(3, SetTdposStorage(1))
	l.SetConsensusStorage(4, SetTdposStorage(2))
	l.SetConsensusStorage(5, SetTdposStorage(2))
	l.SetConsensusStorage(6, SetTdposStorage(3))
	l.SetSnapshot(contractBucket, []byte(termKey), TermKey1())
	// 3. 构造nominate存储
	l.SetSnapshot(contractBucket, []byte(nominateKey), NominateKey1())
	// 4. 构造vote存储
	l.SetSnapshot(contractBucket, []byte(voteKeyPrefix+"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"), VoteKey1())
	l.SetSnapshot(contractBucket, []byte(voteKeyPrefix+"SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"), VoteKey2())
	l.SetSnapshot(contractBucket, []byte(voteKeyPrefix+"akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"), VoteKey3())
	// 5. 调用查看
	v1, err := s.calHisValidators(6, 1)
	if err != nil {
		t.Error("calHisValidators error1.", "err", err)
		return
	}
	if !common.AddressEqual(v1, []string{"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"}) {
		t.Error("AddressEqual error1.", "v1", v1)
		return
	}
	v2, err := s.calHisValidators(6, 2)
	if err != nil {
		t.Error("calHisValidators error2.", "err", err)
		return
	}
	if !common.AddressEqual(v2, []string{"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY", "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"}) {
		t.Error("AddressEqual error2.", "v2", v2)
		return
	}
	v3, err := s.calHisValidators(6, 3)
	if err != nil {
		t.Error("calHisValidators error3.", "err", err)
		return
	}
	if !common.AddressEqual(v3, []string{"akf7qunmeaqb51Wu418d6TyPKp4jdLdpV", "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"}) {
		t.Error("AddressEqual error3.", "v3", v1)
		return
	}
}
