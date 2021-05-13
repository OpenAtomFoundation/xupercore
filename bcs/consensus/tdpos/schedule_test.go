package tdpos

import (
	"encoding/json"
	"testing"
	"time"

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
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	if s := NewSchedule(tdposCfg, cCtx.XLog, cCtx.Ledger, 1); s == nil {
		t.Error("NewSchedule error.")
	}
}

func TestGetLeader(t *testing.T) {
	cStr := getTdposConsensusConf()
	tdposCfg, err := buildConfigs([]byte(cStr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	s := NewSchedule(tdposCfg, cCtx.XLog, cCtx.Ledger, 1)
	if s == nil {
		t.Error("NewSchedule error.")
	}
	s.GetLeader(3)
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

func TestCalHisValidators(t *testing.T) {
	cStr := getTdposConsensusConf()
	tdposCfg, err := buildConfigs([]byte(cStr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	s := NewSchedule(tdposCfg, cCtx.XLog, cCtx.Ledger, 1)
	if s == nil {
		t.Error("NewSchedule error.")
	}
	// 1. 构造term存储
	l, _ := s.ledger.(*kmock.FakeLedger)
	l.Put(kmock.NewBlock(3))
	l.Put(kmock.NewBlock(4))
	l.Put(kmock.NewBlock(5))
	l.Put(kmock.NewBlock(6))
	l.Put(kmock.NewBlock(7))
	l.Put(kmock.NewBlock(8))
	l.Put(kmock.NewBlock(9))
	l.Put(kmock.NewBlock(10))
	l.Put(kmock.NewBlock(11))
	// 2. 整理Block的共识存储
	l.SetConsensusStorage(1, SetTdposStorage(1, nil))
	l.SetConsensusStorage(2, SetTdposStorage(1, nil))
	l.SetConsensusStorage(3, SetTdposStorage(1, nil))
	l.SetConsensusStorage(4, SetTdposStorage(2, nil))

	l.SetConsensusStorage(5, SetTdposStorage(2, nil))
	l.SetConsensusStorage(6, SetTdposStorage(3, nil))
	l.SetConsensusStorage(7, SetTdposStorage(4, nil))
	l.SetConsensusStorage(8, SetTdposStorage(5, nil))
	l.SetConsensusStorage(9, SetTdposStorage(5, nil))
	l.SetConsensusStorage(10, SetTdposStorage(5, nil))
	l.SetConsensusStorage(11, SetTdposStorage(5, nil))
	// [1,2,3,4,4,4,4]

	// 3. 构造nominate存储
	l.SetSnapshot(tdposBucket, []byte(nominateKey), NominateKey1())
	// 4. 构造vote存储
	l.SetSnapshot(tdposBucket, []byte(voteKeyPrefix+"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"), VoteKey1())
	l.SetSnapshot(tdposBucket, []byte(voteKeyPrefix+"SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"), VoteKey2())
	l.SetSnapshot(tdposBucket, []byte(voteKeyPrefix+"akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"), VoteKey3())
	// 5. 调用查看
	v1, err := s.calHisValidators(1)
	if err != nil {
		t.Error("calHisValidators error1.", "err", err)
		return
	}
	if !common.AddressEqual(v1, []string{"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY", "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"}) {
		t.Error("AddressEqual error1.", "v1", v1)
		return
	}
	target, err := s.binarySearch(int64(1), int64(5), int64(1))
	if err != nil {
		t.Error("binarySearch error1.", "err", err)
		return
	}
	if target != 1 {
		t.Error("binarySearch cal err1.", "target", target)
		return
	}
	target, _ = s.binarySearch(int64(1), int64(5), int64(2))
	if target != 4 {
		t.Error("binarySearch cal err2.", "target", target)
		return
	}
	target, _ = s.binarySearch(int64(5), int64(6), int64(3))
	if target != 6 {
		t.Error("binarySearch cal err.", "target", target)
		return
	}
	target, _ = s.binarySearch(int64(7), int64(7), int64(4))
	if target != 7 {
		t.Error("binarySearch cal err.", "target", target)
		return
	}
	target, _ = s.binarySearch(int64(7), int64(8), int64(4))
	if target != 7 {
		t.Error("binarySearch cal err.", "target", target)
		return
	}
	target, _ = s.binarySearch(int64(5), int64(11), int64(5))
	if target != 8 {
		t.Error("binarySearch cal err.", "target", target)
		return
	}
}

func TestMinerScheduling(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	s1 := &tdposSchedule{
		period:            3000,
		blockNum:          20,
		proposerNum:       2,
		alternateInterval: 3000,
		termInterval:      6000,
		initTimestamp:     1559021720000000000,
		log:               cCtx.XLog,
	}
	input := 1618895169 * int64(time.Second)
	term, pos, blockPos := s1.minerScheduling(input)
	if pos != -1 && blockPos != -1 {
		t.Error("minerScheduling cal err.", "term", term, "pos", pos, "blockPos", blockPos)
	}

	input2 := 1618909557 * int64(time.Second)
	s2 := &tdposSchedule{
		period:            3000,
		blockNum:          20,
		proposerNum:       4,
		alternateInterval: 3000,
		termInterval:      6000,
		initTimestamp:     1559021720000000000,
		log:               cCtx.XLog,
	}
	term, pos, blockPos = s2.minerScheduling(input2)
	if pos != -1 && blockPos != -1 {
		t.Error("minerScheduling cal err.", "term", term, "pos", pos, "blockPos", blockPos)
		return
	}
}
