package tdpos

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	bmock "github.com/xuperchain/xupercore/bcs/consensus/mock"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
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

func getBFTTdposConsensusConf() string {
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
	return cCtx, err
}

func TestUnmarshalConfig(t *testing.T) {
	cStr := getTdposConsensusConf()
	_, err := buildConfigs([]byte(cStr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
}

func getConfig(config string) def.ConsensusConfig {
	return def.ConsensusConfig{
		ConsensusName: "tdpos",
		Config:        config,
		StartHeight:   1,
		Index:         0,
	}
}

func TestNewTdposConsensus(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig(getTdposConsensusConf()))
		return
	}
}

func TestCompeteMaster(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig(getTdposConsensusConf()))
		return
	}
	_, _, err = i.CompeteMaster(3)
	if err != nil {
		t.Error("CompeteMaster error", "err", err)
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig(getTdposConsensusConf()))
		return
	}
	b3 := kmock.NewBlock(3)
	l, _ := cCtx.Ledger.(*kmock.FakeLedger)
	l.SetConsensusStorage(1, SetTdposStorage(1, nil))
	l.SetConsensusStorage(2, SetTdposStorage(1, nil))
	l.SetConsensusStorage(3, SetTdposStorage(1, nil))
	c := cCtx.BaseCtx
	i.CheckMinerMatch(&c, b3)
}

func TestProcessBeforeMiner(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig(getTdposConsensusConf()))
		return
	}
	_, _, err = i.ProcessBeforeMiner(time.Now().UnixNano())
	if err != timeoutBlockErr {
		t.Error("ProcessBeforeMiner error", "err", err)
	}
}

func TestProcessConfirmBlock(t *testing.T) {
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig(getTdposConsensusConf()))
	if i == nil {
		t.Error("NewTdposConsensus error", "conf", getConfig(getTdposConsensusConf()))
		return
	}
	b3 := kmock.NewBlock(3)
	l, _ := cCtx.Ledger.(*kmock.FakeLedger)
	l.SetConsensusStorage(1, SetTdposStorage(1, nil))
	l.SetConsensusStorage(2, SetTdposStorage(1, nil))
	l.SetConsensusStorage(3, SetTdposStorage(1, nil))
	if err := i.ProcessConfirmBlock(b3); err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
	}
}

func SetTdposStorage(term int64, justify *lpb.QuorumCert) []byte {
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
	cCtx, err := prepare(getBFTTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewTdposConsensus(*cCtx, getConfig(getBFTTdposConsensusConf()))
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig(getBFTTdposConsensusConf()))
		return
	}
	tdpos, _ := i.(*tdposConsensus)
	l, _ := tdpos.election.ledger.(*kmock.FakeLedger)
	tdpos.election.address = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	// 1, 2区块storage修复
	l.SetConsensusStorage(1, SetTdposStorage(1, justify(1)))
	l.SetConsensusStorage(2, SetTdposStorage(2, justify(2)))

	b3 := kmock.NewBlock(3)
	b3.SetTimestamp(1616481092 * int64(time.Millisecond))
	b3.SetProposer("TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY")
	l.Put(b3)
	l.SetConsensusStorage(3, SetTdposStorage(3, justify(3)))
	b33, _ := l.QueryBlockByHeight(3)
	tdpos.CheckMinerMatch(&cCtx.BaseCtx, b33)
	tdpos.ProcessBeforeMiner(1616481107 * int64(time.Millisecond))
	err = tdpos.ProcessConfirmBlock(b33)
	if err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
		return
	}
}
