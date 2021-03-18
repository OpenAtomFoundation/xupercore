package single

import (
	"encoding/json"
	"testing"
	"time"

	bmock "github.com/xuperchain/xupercore/bcs/consensus/mock"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
)

func getSingleConsensusConf() []byte {
	c := map[string]string{
		"version": "0",
		"miner":   bmock.Miner,
		"period":  "3000",
	}
	j, _ := json.Marshal(c)
	return j
}

func prepare() (*cctx.ConsensusCtx, error) {
	l := kmock.NewFakeLedger(getSingleConsensusConf())
	cCtx, err := bmock.NewConsensusCtx(l)
	cCtx.Ledger = l

	return cCtx, err
}

func getConsensusConf() def.ConsensusConfig {
	return def.ConsensusConfig{
		ConsensusName: "single",
		Config:        string(getSingleConsensusConf()),
		StartHeight:   1,
		Index:         0,
	}
}

func getWrongConsensusConf() def.ConsensusConfig {
	return def.ConsensusConfig{
		ConsensusName: "single2",
		Config:        string(getSingleConsensusConf()),
		StartHeight:   1,
		Index:         0,
	}
}

func TestNewSingleConsensus(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("TestNewSingleConsensus", "err", err)
		return
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cCtx, conf)
	if i == nil {
		t.Error("NewSingleConsensus error")
		return
	}
	if i := NewSingleConsensus(*cCtx, getWrongConsensusConf()); i != nil {
		t.Error("NewSingleConsensus check name error")
	}
	i.Stop()
	i.Start()
	i.ProcessBeforeMiner(time.Now().UnixNano())
	cCtx.XLog = nil
	i = NewSingleConsensus(*cCtx, conf)
	if i != nil {
		t.Error("NewSingleConsensus nil logger error")
	}
}

func TestGetConsensusStatus(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("TestNewSingleConsensus", "err", err)
		return
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cCtx, conf)
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
	m := ValidatorsInfo{}
	err = json.Unmarshal(vb, &m)
	if err != nil {
		t.Error("GetCurrentValidatorsInfo unmarshal error", "error", err)
		return
	}
	if m.Validators[0] != bmock.Miner {
		t.Error("GetCurrentValidatorsInfo error", "m", m, "vb", vb)
	}
}

func TestCompeteMaster(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("TestNewSingleConsensus", "err", err)
		return
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cCtx, conf)
	isMiner, shouldSync, _ := i.CompeteMaster(2)
	if isMiner && shouldSync {
		t.Error("TestCompeteMaster error")
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("TestNewSingleConsensus", "err", err)
		return
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cCtx, conf)
	f, err := bmock.NewBlock(2, cCtx.Crypto, cCtx.Address)
	if err != nil {
		t.Error("NewBlock error", "error", err)
		return
	}
	ok, err := i.CheckMinerMatch(&cCtx.BaseCtx, f)
	if !ok || err != nil {
		t.Error("TestCheckMinerMatch error", "error", err, cCtx.Address.PrivateKey)
	}
	_, _, err = i.ProcessBeforeMiner(time.Now().UnixNano())
	if err != nil {
		t.Error("ProcessBeforeMiner error", "error", err)
	}
}
