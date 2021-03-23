package xpoa

import (
	"encoding/json"
	"testing"
	"time"

	bmock "github.com/xuperchain/xupercore/bcs/consensus/mock"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
)

func TestUnmarshalConfig(t *testing.T) {
	cStr := "{\"period\": 3000,\"block_num\": 10,\"init_proposer\": [{\"address\": \"f3prTg9itaZY6m48wXXikXdcxiByW7zgk\",\"neturl\": \"127.0.0.1:47102\"},{\"address\": \"U9sKwFmgJVfzgWcfAG47dKn1kLQTqeZN3\",\"neturl\": \"127.0.0.1:47103\"},{\"address\": \"RUEMFGDEnLBpnYYggnXukpVfR9Skm59ph\",\"neturl\": \"127.0.0.1:47104\"}]}"
	config := &xpoaConfig{}
	err := json.Unmarshal([]byte(cStr), config)
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	if config.Period != 3000 {
		t.Error("Config unmarshal err", "v", config.Period)
	}
}

func getXpoaConsensusConf() string {
	return `{
        "period":3000,
        "block_num":10,
        "contract_name":"xpoa_validates",
        "method_name":"get_validates",
        "init_proposer": [{
            	"address" : "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN",
				"neturl" : "/ip4/127.0.0.1/tcp/47101/p2p/QmVcSF4F7rTdsvUJqsik98tXRXMBUqL5DSuBpyYKVhjuG4"
            },
            {
                "address" : "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT", 
				"neturl" : "/ip4/127.0.0.1/tcp/47102/p2p/Qmd1sJ4s7JTfHvetfjN9vNE5fhkLESs42tYHc5RYUBPnEv"
            }
        ]
	}`
}

func prepare() (*cctx.ConsensusCtx, error) {
	l := kmock.NewFakeLedger([]byte(getXpoaConsensusConf()))
	cCtx, err := bmock.NewConsensusCtx(l)
	cCtx.Ledger = l
	p, ctxN, err := kmock.NewP2P("node")
	p.Init(ctxN)
	cCtx.Network = p
	cCtx.XLog = bmock.NewFakeLogger()
	return cCtx, err
}

func getConfig() def.ConsensusConfig {
	return def.ConsensusConfig{
		ConsensusName: "xpoa",
		Config:        getXpoaConsensusConf(),
		StartHeight:   1,
		Index:         0,
	}
}

func TestNewXpoaConsensus(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig())
		return
	}
}

func TestCompeteMaster(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig())
		return
	}
	_, _, err = i.CompeteMaster(3)
	if err != nil {
		t.Error("CompeteMaster error")
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig())
		return
	}
	b3 := kmock.NewBlock(3)
	c := cCtx.BaseCtx
	f, err := i.CheckMinerMatch(&c, b3)
	if !f { // verifyBlock通过miner检验
		t.Error("CheckMinerMatch error")
	}
}

func TestProcessBeforeMiner(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig())
		return
	}
	i.ProcessBeforeMiner(time.Now().UnixNano())
}

func TestProcessConfirmBlock(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig())
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig())
		return
	}
	b3 := kmock.NewBlock(3)
	if err := i.ProcessConfirmBlock(b3); err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
	}
}
