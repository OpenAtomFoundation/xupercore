package xpoa

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	bmock "github.com/xuperchain/xupercore/bcs/consensus/mock"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
)

func TestUnmarshalConfig(t *testing.T) {
	cStr := `{
		"period": 3000,
		"block_num": 10,
		"init_proposer": {
			"address": ["f3prTg9itaZY6m48wXXikXdcxiByW7zgk", "U9sKwFmgJVfzgWcfAG47dKn1kLQTqeZN3", "RUEMFGDEnLBpnYYggnXukpVfR9Skm59ph"]
		}
	}`
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
        "init_proposer": {
            "address" : ["dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"]
        }
	}`
}

func getBFTXpoaConsensusConf() string {
	return `{
        "period":3000,
        "block_num":10,
        "init_proposer": {
            "address" : ["dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"]
        },
		"bft_config":{}
	}`
}

func prepare(config string) (*cctx.ConsensusCtx, error) {
	l := kmock.NewFakeLedger([]byte(config))
	cCtx, err := bmock.NewConsensusCtx(l)
	cCtx.Ledger = l
	p, ctxN, err := kmock.NewP2P("node")
	p.Init(ctxN)
	cCtx.Network = p
	cCtx.XLog = bmock.NewFakeLogger()
	return cCtx, err
}

func getConfig(config string) def.ConsensusConfig {
	return def.ConsensusConfig{
		ConsensusName: "xpoa",
		Config:        config,
		StartHeight:   1,
		Index:         0,
	}
}

func TestNewXpoaConsensus(t *testing.T) {
	cCtx, err := prepare(getXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getXpoaConsensusConf()))
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig(getXpoaConsensusConf()))
		return
	}
}

func TestCompeteMaster(t *testing.T) {
	cCtx, err := prepare(getXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getXpoaConsensusConf()))
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig(getXpoaConsensusConf()))
		return
	}
	_, _, err = i.CompeteMaster(3)
	if err != nil {
		t.Error("CompeteMaster error")
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cCtx, err := prepare(getXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getXpoaConsensusConf()))
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig(getXpoaConsensusConf()))
		return
	}
	b3 := kmock.NewBlock(3)
	c := cCtx.BaseCtx
	i.CheckMinerMatch(&c, b3)
}

func TestProcessBeforeMiner(t *testing.T) {
	cCtx, err := prepare(getXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getXpoaConsensusConf()))
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig(getXpoaConsensusConf()))
		return
	}
	i.ProcessBeforeMiner(time.Now().UnixNano())
}

func TestProcessConfirmBlock(t *testing.T) {
	cCtx, err := prepare(getXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getXpoaConsensusConf()))
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig(getXpoaConsensusConf()))
		return
	}
	b3 := kmock.NewBlock(3)
	if err := i.ProcessConfirmBlock(b3); err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
	}
}

func TestGetJustifySigns(t *testing.T) {
	cCtx, err := prepare(getXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getXpoaConsensusConf()))
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig(getXpoaConsensusConf()))
		return
	}
	xpoa, _ := i.(*xpoaConsensus)
	l, _ := xpoa.election.ledger.(*kmock.FakeLedger)
	l.Put(kmock.NewBlock(3))
	l.SetConsensusStorage(1, SetXpoaStorage(1, nil))
	b, err := l.QueryBlockByHeight(3)
	xpoa.GetJustifySigns(b)
}

func justify(height int64) *lpb.QuorumCert {
	var m []byte
	var err error
	if height-1 >= 0 {
		parent := &lpb.QuorumCert{
			ProposalId: []byte{byte(height - 1)},
			ViewNumber: height - 1,
		}
		m, err = proto.Marshal(parent)
		if err != nil {
			return nil
		}
	}
	return &lpb.QuorumCert{
		ProposalId:  []byte{byte(height)},
		ViewNumber:  height,
		ProposalMsg: m,
	}
}

func TestBFT(t *testing.T) {
	cCtx, err := prepare(getBFTXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getBFTXpoaConsensusConf()))
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig(getBFTXpoaConsensusConf()))
		return
	}
	xpoa, _ := i.(*xpoaConsensus)
	l, _ := xpoa.election.ledger.(*kmock.FakeLedger)
	xpoa.election.address = "now=dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	// 1, 2区块storage修复
	l.SetConsensusStorage(1, SetXpoaStorage(1, justify(1)))
	l.SetConsensusStorage(2, SetXpoaStorage(2, justify(2)))

	b3 := kmock.NewBlock(3)
	b3.SetTimestamp(1616481092 * int64(time.Millisecond))
	l.Put(b3)
	l.SetConsensusStorage(3, SetXpoaStorage(3, justify(3)))
	b33, _ := l.QueryBlockByHeight(3)
	xpoa.CheckMinerMatch(&cCtx.BaseCtx, b33)
	xpoa.ProcessBeforeMiner(1616481107 * int64(time.Millisecond))
	err = xpoa.ProcessConfirmBlock(b33)
	if err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
		return
	}
}
