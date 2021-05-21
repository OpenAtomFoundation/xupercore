package xpoa

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
)

var (
	InitValidators = []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	newValidators  = []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT", "iYjtLcW6SVCiousAb5DFKWtWroahhEj4u"}
)

func NewSchedule(address string, validators []string, enableBFT bool) (*xpoaSchedule, error) {
	c, err := prepare(getXpoaConsensusConf())
	return &xpoaSchedule{
		address:        address,
		period:         3000,
		blockNum:       10,
		validators:     validators,
		initValidators: InitValidators,
		enableBFT:      enableBFT,
		ledger:         c.Ledger,
		log:            c.XLog,
	}, err
}

func SetXpoaStorage(term int64, justify *lpb.QuorumCert) []byte {
	s := common.ConsensusStorage{
		Justify:     justify,
		CurTerm:     term,
		CurBlockNum: 3,
	}
	b, err := json.Marshal(&s)
	if err != nil {
		return nil
	}
	return b
}

func ValidateKey1() []byte {
	rawV := &ProposerInfo{
		Address: newValidators,
	}
	rawBytes, err := json.Marshal(rawV)
	if err != nil {
		return nil
	}
	return rawBytes
}

func TestGetLeader(t *testing.T) {
	s, err := NewSchedule("dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", InitValidators, true)
	if err != nil {
		t.Error("newSchedule error.")
		return
	}
	// fake ledger的前2个block都是 dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN 生成
	term, pos, blockPos := s.minerScheduling(time.Now().UnixNano()+s.period*int64(time.Millisecond), len(s.validators))
	if _, err := s.ledger.QueryBlockByHeight(2); err != nil {
		t.Error("QueryBlockByHeight error.")
		return
	}
	l := s.GetLeader(3)
	if s.validators[pos] != l {
		t.Error("GetLeader err", "term", term, "pos", pos, "blockPos", blockPos, "cal leader", l)
	}
}

func TestGetValidates(t *testing.T) {
	s, err := NewSchedule("dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", InitValidators, true)
	if err != nil {
		t.Error("newSchedule error.")
		return
	}
	l, _ := s.ledger.(*kmock.FakeLedger)
	l.Put(kmock.NewBlock(3))
	l.Put(kmock.NewBlock(4))
	l.Put(kmock.NewBlock(5))
	l.Put(kmock.NewBlock(6))
	// 2. 整理Block的共识存储
	l.SetConsensusStorage(1, SetXpoaStorage(1, nil))
	l.SetConsensusStorage(2, SetXpoaStorage(1, nil))
	l.SetConsensusStorage(3, SetXpoaStorage(1, nil))
	l.SetConsensusStorage(4, SetXpoaStorage(2, nil))
	l.SetConsensusStorage(5, SetXpoaStorage(2, nil))
	l.SetConsensusStorage(6, SetXpoaStorage(3, nil))
	l.SetSnapshot(poaBucket, []byte(fmt.Sprintf("0_%s", validateKeys)), ValidateKey1())
	v, err := s.getValidates(6)
	if !common.AddressEqual(v, newValidators) {
		t.Error("AddressEqual error1.", "v", v)
	}
}
