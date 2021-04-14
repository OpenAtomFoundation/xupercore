package main

import (
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/xuperchain/xupercore/bcs/network/p2pv2"
	"github.com/xuperchain/xupercore/kernel/common/xconfig"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/network"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	NodePath = filepath.Join(utils.GetCurFileDir(), "/test")

	Node   = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	NodeIp = "/ip4/127.0.0.1/tcp/47101/p2p/QmVcSF4F7rTdsvUJqsik98tXRXMBUqL5DSuBpyYKVhjuG4"
	PubKey = "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571}"
	PriKey = "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571,\"D\":29079635126530934056640915735344231956621504557963207107451663058887647996601}"

	NodeA   = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	NodeAIp = "/ip4/127.0.0.1/tcp/47101/p2p/QmVcSF4F7rTdsvUJqsik98tXRXMBUqL5DSuBpyYKVhjuG4"
	PubKeyA = "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571}"
	PriKeyA = "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571,\"D\":29079635126530934056640915735344231956621504557963207107451663058887647996601}"

	NodeB   = "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"
	NodeBIp = "/ip4/127.0.0.1/tcp/47102/p2p/Qmd1sJ4s7JTfHvetfjN9vNE5fhkLESs42tYHc5RYUBPnEv"
	PubKeyB = `{"Curvname":"P-256","X":38583161743450819602965472047899931736724287060636876073116809140664442044200,"Y":73385020193072990307254305974695788922719491565637982722155178511113463088980}`
	PriKeyB = `{"Curvname":"P-256","X":38583161743450819602965472047899931736724287060636876073116809140664442044200,"Y":73385020193072990307254305974695788922719491565637982722155178511113463088980,"D":98698032903818677365237388430412623738975596999573887926929830968230132692775}`

	NodeC   = "akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"
	PubKeyC = `{"Curvname":"P-256","X":82701086955329320728418181640262300520017105933207363210165513352476444381539,"Y":23833609129887414146586156109953595099225120577035152268521694007099206660741}`
	PriKeyC = `{"Curvname":"P-256","X":82701086955329320728418181640262300520017105933207363210165513352476444381539,"Y":23833609129887414146586156109953595099225120577035152268521694007099206660741,"D":57537645914107818014162200570451409375770015156750200591470574847931973776404}`
	NodeCIp = "/ip4/127.0.0.1/tcp/47103/p2p/QmUv4Jw8QbW85SHQRiXi2jffFXTXZzRxhW2H34Hq6W4d58"
)

type ElectionA struct {
	addrs []string
}

func (e *ElectionA) GetLeader(round int64) string {
	pos := (round - 1) % 3
	return e.addrs[pos]
}

func (e *ElectionA) GetValidatorsMsgAddr() []string {
	return []string{NodeAIp, NodeBIp, NodeCIp}
}

func (e *ElectionA) GetValidators(round int64) []string {
	return []string{NodeA, NodeB, NodeC}
}

func (e *ElectionA) GetIntAddress(a string) string {
	switch a {
	case NodeA:
		return NodeAIp
	case NodeB:
		return NodeBIp
	case NodeC:
		return NodeCIp
	}
	return ""
}

func NewFakeLogger(node string) logs.Logger {
	var Npath string
	switch node {
	case "node":
		Npath = "node"
	case "nodeA":
		Npath = "node1"
	case "nodeB":
		Npath = "node2"
	case "nodeC":
		Npath = "node3"
	}
	path := filepath.Join(NodePath, Npath)
	confFile := filepath.Join(path, "log.yaml")
	logDir := filepath.Join(path, "/logs")

	logs.InitLog(confFile, logDir)
	log, _ := logs.NewLogger(node, "smr_test")
	return log
}

func NewP2P(node string, log logs.Logger) network.Network {
	// 创建p2p
	var Npath string
	switch node {
	case "node":
		Npath = "node"
	case "nodeA":
		Npath = "node1"
	case "nodeB":
		Npath = "node2"
	case "nodeC":
		Npath = "node3"
	}
	path := filepath.Join(NodePath, Npath)
	econfPath := filepath.Join(path, "/env.yaml")
	ecfg, err := xconfig.LoadEnvConf(econfPath)
	if err != nil {
		log.Error("err", err)
	}
	netCtx, err := nctx.NewNetCtx(ecfg)
	if err != nil {
		log.Error("err", err)
	}
	p2p, err := network.NewNetwork(netCtx)
	log.Error("err", err)
	return p2p
}

func NewCryptoClient(node string) (cctx.Address, cctx.CryptoClient) {
	var priKeyStr, pubKeyStr, addr string
	switch node {
	case "node":
		addr = Node
		pubKeyStr = PubKey
		priKeyStr = PriKey
	case "nodeA":
		addr = NodeA
		pubKeyStr = PubKeyA
		priKeyStr = PriKeyA
	case "nodeB":
		addr = NodeB
		pubKeyStr = PubKeyB
		priKeyStr = PriKeyB
	case "nodeC":
		addr = NodeC
		pubKeyStr = PubKeyC
		priKeyStr = PriKeyC
	}
	cc, err := client.CreateCryptoClientFromJSONPrivateKey([]byte(priKeyStr))
	if err != nil {
		return cctx.Address{}, nil
	}
	sk, _ := cc.GetEcdsaPrivateKeyFromJsonStr(priKeyStr)
	pk, _ := cc.GetEcdsaPublicKeyFromJsonStr(pubKeyStr)
	a := cctx.Address{
		Address:       addr,
		PrivateKeyStr: priKeyStr,
		PublicKeyStr:  pubKeyStr,
		PrivateKey:    sk,
		PublicKey:     pk,
	}
	return a, cc
}

func InitQcTee() *chainedBft.QCPendingTree {
	initQC := &chainedBft.QuorumCert{
		VoteInfo: &chainedBft.VoteInfo{
			ProposalId:   []byte{0},
			ProposalView: 0,
		},
		LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
			CommitStateId: []byte{0},
		},
	}
	rootNode := &chainedBft.ProposalNode{
		In: initQC,
	}
	return &chainedBft.QCPendingTree{
		Genesis:  rootNode,
		Root:     rootNode,
		HighQC:   rootNode,
		CommitQC: rootNode,
	}
}

func NewSMR(node string, log logs.Logger, p2p network.Network) *chainedBft.Smr {
	a, cc := NewCryptoClient(node)
	cryptoClient := cCrypto.NewCBFTCrypto(&a, cc)
	pacemaker := &chainedBft.DefaultPaceMaker{}
	q := InitQcTee()
	saftyrules := &chainedBft.DefaultSaftyRules{
		Crypto: cryptoClient,
		QcTree: q,
	}
	election := &ElectionA{
		addrs: []string{NodeA, NodeB, NodeC},
	}
	s := chainedBft.NewSmr("xuper", a.Address, log, p2p, cryptoClient, pacemaker, saftyrules, election, q)
	if s == nil {
		return nil
	}
	return s
}

func main() {
	go nodeMain("nodeA")
	go nodeMain("nodeB")
	go nodeMain("nodeC")
	for {
	}
}

// node 为node1，node2，node3
func nodeMain(node string) {
	election := &ElectionA{
		addrs: []string{NodeA, NodeB, NodeC},
	}
	log := NewFakeLogger(node)
	p := NewP2P(node, log)
	go p.Start()
	s := NewSMR(node, log, p)
	go s.Start()
	go CompeteLoop(s, log, election.GetValidatorsMsgAddr())
}

func CompeteMaster(smr *chainedBft.Smr) string {
	if smr.GetCurrentView() == 0 {
		return NodeA
	}
	return smr.Election.GetLeader(smr.GetCurrentView())
}

func CompeteLoop(smr *chainedBft.Smr, log logs.Logger, validators []string) {
	for {
		miner := CompeteMaster(smr)
		log.Debug("Compete", "round", smr.GetCurrentView(), "leader", miner, "isMiner", miner == smr.GetAddress(), "address", smr.GetAddress())
		if miner != smr.GetAddress() {
			time.Sleep(time.Millisecond * 10)
			continue
		}
		if smr.GetCurrentView() == 0 {
			if err := smr.ProcessProposal(1, []byte("1"), validators); err != nil {
				log.Error("Smr ProcessProposal error", "error", err)
			}
			continue
		}
		if err := smr.ProcessProposal(smr.GetCurrentView(), []byte(fmt.Sprint((smr.GetCurrentView()))), validators); err != nil {
			log.Error("Smr ProcessProposal error", "error", err)
		}
	}
}
