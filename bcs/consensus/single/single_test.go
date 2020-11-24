package single

import (
	"encoding/json"
	"testing"

	"github.com/xuperchain/xupercore/bcs/consensus/mock"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

func getSingleConsensusConf() []byte {
	c := SingleConfig{
		Miner:   mock.Miner,
		Period:  3000,
		Version: 0,
	}
	j, _ := json.Marshal(c)
	return j
}

func prepare() cctx.ConsensusCtx {
	l := mock.NewFakeLedger(getSingleConsensusConf())
	cCtx := mock.NewConsensusCtx()
	cCtx.Ledger = l
	return cCtx
}

func getConsensusConf() cctx.ConsensusConfig {
	return cctx.ConsensusConfig{
		ConsensusName: "single",
		Config:        string(getSingleConsensusConf()),
		BeginHeight:   1,
		Index:         0,
	}
}

func getWrongConsensusConf() cctx.ConsensusConfig {
	return cctx.ConsensusConfig{
		ConsensusName: "single2",
		Config:        string(getSingleConsensusConf()),
		BeginHeight:   1,
		Index:         0,
	}
}

func TestNewSingleConsensus(t *testing.T) {
	cCtx := prepare()
	conf := getConsensusConf()
	i := NewSingleConsensus(cCtx, conf)
	if i == nil {
		t.Error("NewSingleConsensus error")
		return
	}
	if i := NewSingleConsensus(cCtx, getWrongConsensusConf()); i != nil {
		t.Error("NewSingleConsensus check name error")
	}
}

func TestGetConsensusStatus(t *testing.T) {
	cCtx := prepare()
	conf := getConsensusConf()
	i := NewSingleConsensus(cCtx, conf)
	status, _ := i.GetConsensusStatus()
	if status.GetVersion() != 0 {
		t.Error("GetVersion error")
		return
	}
	if status.GetStepConsensusIndex() != 0 {
		t.Error("GetStepConsensusIndex error")
		return
	}
	if status.GetConsensusBeginInfo() != 1 {
		t.Error("GetConsensusBeginInfo error")
		return
	}
	if status.GetConsensusName() != "single" {
		t.Error("GetConsensusName error")
		return
	}
	vb := status.GetCurrentValidatorsInfo()
	m := MinerInfo{}
	err := json.Unmarshal(vb, &m)
	if err != nil {
		t.Error("GetCurrentValidatorsInfo unmarshal error", "error", err)
		return
	}
	if m.Miner != mock.Miner {
		t.Error("GetCurrentValidatorsInfo error", "m", m, "vb", vb)
	}
}

func TestCompeteMaster(t *testing.T) {
	cCtx := prepare()
	conf := getConsensusConf()
	i := NewSingleConsensus(cCtx, conf)
	isMiner, shouldSync, _ := i.CompeteMaster(2)
	if isMiner && shouldSync {
		t.Error("TestCompeteMaster error")
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cCtx := prepare()
	conf := getConsensusConf()
	i := NewSingleConsensus(cCtx, conf)
	f := mock.NewBlock(2, []byte{})
	ok, err := i.CheckMinerMatch(cCtx.BCtx, f)
	if !ok || err != nil {
		t.Error("TestCheckMinerMatch error")
	}
}
