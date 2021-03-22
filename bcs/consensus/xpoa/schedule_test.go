package xpoa

import (
	"testing"
	"time"
)

var (
	initValidators = []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
)

func newSchedule(address string, validators []string, enableBFT bool) (*xpoaSchedule, error) {
	c, err := prepare()
	return &xpoaSchedule{
		address:        address,
		period:         3000,
		blockNum:       10,
		validators:     validators,
		initValidators: initValidators,
		enableBFT:      enableBFT,
		ledger:         c.Ledger,
		log:            c.XLog,
	}, err
}

func TestGetLeader(t *testing.T) {
	s, err := newSchedule("dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", initValidators, true)
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
