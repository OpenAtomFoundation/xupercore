package mock

import (
	"errors"
	"path/filepath"
	"time"

	log "github.com/xuperchain/log15"
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	BcName = "xuper5"
	nodeIp = "/ip4/127.0.0.1/tcp/47101/p2p/QmVcSF4F7rTdsvUJqsik98tXRXMBUqL5DSuBpyYKVhjuG4"
	priKey = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571,"D":29079635126530934056640915735344231956621504557963207107451663058887647996601}`
	PubKey = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571}`
	Miner  = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"

	blockSetItemErr = errors.New("item invalid")
)

func NewFakeLogger() logs.Logger {
	confFile := utils.GetCurFileDir()
	confFile = filepath.Join(confFile, "config/log.yaml")
	logDir := utils.GetCurFileDir()
	logDir = filepath.Join(logDir, "logs")

	logs.InitLog(confFile, logDir)
	log, _ := logs.NewLogger("", "consensus_test")
	return log
}

// NewConsensusCtx 返回除ledger以外的所有所需的共识上下文
func NewConsensusCtx(ledger *kmock.FakeLedger) (*cctx.ConsensusCtx, error) {
	cc, a, err := NewCryptoClient()
	if err != nil {
		return nil, err
	}
	return &cctx.ConsensusCtx{
		BcName: "xuper",
		Ledger: ledger,
		BaseCtx: xcontext.BaseCtx{
			XLog: NewFakeLogger(),
		},
		Contract: &kmock.FakeManager{
			R: &kmock.FakeRegistry{
				M: make(map[string]contract.KernMethod),
			},
		},
		Crypto:  cc,
		Address: a,
	}, nil
}

func NewCryptoClient() (cctx.CryptoClient, *cctx.Address, error) {
	cc, err := client.CreateCryptoClientFromJSONPrivateKey([]byte(priKey))
	if err != nil {
		log.Error("CreateCryptoClientFromJSONPrivateKey error", "error", err)
	}
	sk, err := cc.GetEcdsaPrivateKeyFromJsonStr(priKey)
	if err != nil {
		return nil, nil, err
	}
	pk, err := cc.GetEcdsaPublicKeyFromJsonStr(PubKey)
	if err != nil {
		return nil, nil, err
	}
	a := &cctx.Address{
		Address:       Miner,
		PrivateKeyStr: priKey,
		PublicKeyStr:  PubKey,
		PrivateKey:    sk,
		PublicKey:     pk,
	}
	return cc, a, nil
}

func NewBlock(height int, c cctx.CryptoClient, a *cctx.Address) (*kmock.FakeBlock, error) {
	b := &kmock.FakeBlock{
		Proposer:         a.Address,
		Height:           int64(height),
		Blockid:          []byte{byte(height)},
		ConsensusStorage: []byte{},
		Timestamp:        time.Now().UnixNano(),
		PublicKey:        a.PrivateKeyStr,
	}
	s, err := c.SignECDSA(a.PrivateKey, b.Blockid)
	if err == nil {
		b.Sign = s
	}
	return b, err
}

func NewBlockWithStorage(height int, c cctx.CryptoClient, a *cctx.Address, s []byte) (*kmock.FakeBlock, error) {
	b := &kmock.FakeBlock{
		Proposer:         a.Address,
		Height:           int64(height),
		Blockid:          []byte{byte(height)},
		ConsensusStorage: s,
		Timestamp:        time.Now().UnixNano(),
		PublicKey:        a.PrivateKeyStr,
		PreHash:          []byte{byte(height - 1)},
	}
	s, err := c.SignECDSA(a.PrivateKey, b.Blockid)
	if err == nil {
		b.Sign = s
	}
	return b, err
}
