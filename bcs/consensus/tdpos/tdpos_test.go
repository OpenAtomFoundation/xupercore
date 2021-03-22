package tdpos

import (
	"testing"
	"time"

	bmock "github.com/xuperchain/xupercore/bcs/consensus/mock"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
)

func getTdposConsensusConf() string {
	return `{
        "timestamp": "1559021720000000000",
        "proposer_num": "2",
        "period": "3000",
        "alternate_interval": "3000",
        "term_interval": "6000",
        "block_num": "20",
        "vote_unit_price": "1",
        "init_proposer": {
            "1": ["TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY", "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"]
        },
        "init_proposer_neturl": {
            "1": ["/ip4/127.0.0.1/tcp/38101/p2p/Qmf2HeHe4sspGkfRCTq6257Vm3UHzvh2TeQJHHvHzzuFw6e", "/ip4/127.0.0.1/tcp/38102/p2p/QmQKp8pLWSgV4JiGjuULKV1JsdpxUtnDEUMP8sGaaUbwVL"]
        }
	}`
}

func prepare() (*cctx.ConsensusCtx, error) {
	l := kmock.NewFakeLedger([]byte(getTdposConsensusConf()))
	cCtx, err := bmock.NewConsensusCtx(l)
	cCtx.Ledger = l
	p, ctxN, err := kmock.NewP2P("node")
	p.Init(ctxN)
	cCtx.Network = p
	return cCtx, err
}

func TestUnmarshalConfig(t *testing.T) {
	cStr := getTdposConsensusConf()
	_, err := buildConfigs([]byte(cStr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
}

func getConfig() def.ConsensusConfig {
	return def.ConsensusConfig{
		ConsensusName: "tdpos",
		Config:        getTdposConsensusConf(),
		StartHeight:   1,
		Index:         0,
	}
}

func TestNewTdposConsensus(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig())
		return
	}
}

func TestCompeteMaster(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig())
		return
	}
	_, _, err = i.CompeteMaster(3)
	if err != nil {
		t.Error("CompeteMaster error", "err", err)
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig())
		return
	}
	b3 := kmock.NewBlock(3)
	l, _ := cCtx.Ledger.(*kmock.FakeLedger)
	l.SetConsensusStorage(1, SetTdposStorage(1))
	l.SetConsensusStorage(2, SetTdposStorage(1))
	l.SetConsensusStorage(3, SetTdposStorage(1))
	c := cCtx.BaseCtx
	i.CheckMinerMatch(&c, b3)
}

func TestProcessBeforeMiner(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig())
		return
	}
	_, _, err = i.ProcessBeforeMiner(time.Now().UnixNano())
	if err != timeoutBlockErr {
		t.Error("ProcessBeforeMiner error", "err", err)
	}
}

func TestProcessConfirmBlock(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig())
		return
	}
	b3 := kmock.NewBlock(3)
	l, _ := cCtx.Ledger.(*kmock.FakeLedger)
	l.SetConsensusStorage(1, SetTdposStorage(1))
	l.SetConsensusStorage(2, SetTdposStorage(1))
	l.SetConsensusStorage(3, SetTdposStorage(1))
	if err := i.ProcessConfirmBlock(b3); err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
	}
}
