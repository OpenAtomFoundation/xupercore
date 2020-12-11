package main

import (
	"path/filepath"
	"time"

	"github.com/xuperchain/xupercore/bcs/network/p2pv2"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	nodePath = filepath.Join(utils.GetCurFileDir(), "/test")

	nodeA   = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	nodeAIp = "/ip4/127.0.0.1/tcp/47101/p2p/QmVcSF4F7rTdsvUJqsik98tXRXMBUqL5DSuBpyYKVhjuG4"
	pubKeyA = "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571}"
	priKeyA = "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571,\"D\":29079635126530934056640915735344231956621504557963207107451663058887647996601}"

	nodeB   = "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"
	nodeBIp = "/ip4/127.0.0.1/tcp/47102/p2p/Qmd1sJ4s7JTfHvetfjN9vNE5fhkLESs42tYHc5RYUBPnEv"
	pubKeyB = `{"Curvname":"P-256","X":38583161743450819602965472047899931736724287060636876073116809140664442044200,"Y":73385020193072990307254305974695788922719491565637982722155178511113463088980}`
	priKeyB = `{"Curvname":"P-256","X":38583161743450819602965472047899931736724287060636876073116809140664442044200,"Y":73385020193072990307254305974695788922719491565637982722155178511113463088980,"D":98698032903818677365237388430412623738975596999573887926929830968230132692775}`

	nodeC   = "akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"
	pubKeyC = `{"Curvname":"P-256","X":82701086955329320728418181640262300520017105933207363210165513352476444381539,"Y":23833609129887414146586156109953595099225120577035152268521694007099206660741}`
	priKeyC = `{"Curvname":"P-256","X":82701086955329320728418181640262300520017105933207363210165513352476444381539,"Y":23833609129887414146586156109953595099225120577035152268521694007099206660741,"D":57537645914107818014162200570451409375770015156750200591470574847931973776404}`
	nodeCIp = "/ip4/127.0.0.1/tcp/47103/p2p/QmUv4Jw8QbW85SHQRiXi2jffFXTXZzRxhW2H34Hq6W4d58"
)

type electionA struct{}

func (e *electionA) GetLeader(round int64) string {
	pos := (round - 1) % 3
	return addresses()[pos]
}

func (e *electionA) GetValidatorsMsgAddr() []string {
	return []string{nodeAIp, nodeBIp, nodeCIp}
}

func (e *electionA) GetValidators(round int64) []string {
	return []string{nodeA, nodeB, nodeC}
}

func (e *electionA) GetIntAddress(a string) string {
	switch a {
	case nodeA:
		return nodeAIp
	case nodeB:
		return nodeBIp
	case nodeC:
		return nodeCIp
	}
	return ""
}

func addresses() []string {
	return []string{nodeA, nodeB, nodeC}
}

func main() {
	go nodeMain(nodeA)
	go nodeMain(nodeB)
	go nodeMain(nodeC)
	for {
	}
}

func initQcTee() *chainedBft.QCPendingTree {
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

func prepareSmr(log logs.Logger, address string, publicKey string, privateKey string, p2p cctx.P2pCtxInConsensus, election chainedBft.ProposerElectionInterface) *chainedBft.Smr {
	cc, err := client.CreateCryptoClientFromJSONPrivateKey([]byte(privateKey))
	if err != nil {
		log.Error("CreateCryptoClientFromJSONPrivateKey error", "error", err)
	}
	sk, _ := cc.GetEcdsaPrivateKeyFromJsonStr(privateKey)
	pk, _ := cc.GetEcdsaPublicKeyFromJsonStr(publicKey)
	a := cctx.Address{
		Address:       address,
		PrivateKeyStr: privateKey,
		PublicKeyStr:  publicKey,
		PrivateKey:    sk,
		PublicKey:     pk,
	}
	cryptoClient := cCrypto.NewCBFTCrypto(a, cc)
	pacemaker := &chainedBft.DefaultPaceMaker{}
	saftyrules := &chainedBft.DefaultSaftyRules{
		Crypto: cryptoClient,
	}
	return chainedBft.NewSmr("xuper", address, log, p2p, cryptoClient, pacemaker, saftyrules, election, initQcTee())
}

// address 为node1，node2，node3
func nodeMain(address string) {
	// 新建p2p实例
	var logName string
	switch address {
	case nodeA:
		logName = "node1"
	case nodeB:
		logName = "node2"
	case nodeC:
		logName = "node3"
	}
	election := &electionA{}
	log := NewFakeLogger(logName)
	log.Info("Begin")
	node1 := p2pv2.NewP2PServerV2()
	addrStr := "/" + logName + "/conf/network.yaml"
	ctx := nctx.MockDomainCtx(filepath.Join(nodePath, addrStr))
	ctx.SetMetricSwitch(true)
	if err := node1.Init(ctx); err != nil {
		log.Error("server init error", "error", err)
	}
	go node1.Start()

	// 新建smr
	var smr *chainedBft.Smr
	switch address {
	case nodeA:
		smr = prepareSmr(log, address, pubKeyA, priKeyA, node1, election)
	case nodeB:
		smr = prepareSmr(log, address, pubKeyB, priKeyB, node1, election)
	case nodeC:
		smr = prepareSmr(log, address, pubKeyC, priKeyC, node1, election)
	}
	smr.RegisterToNetwork()
	go smr.Start()

	log.Info("Smr has been created.")
	go CompeteLoop(smr, log, election.GetValidatorsMsgAddr())
}

func CompeteMaster(smr *chainedBft.Smr) string {
	if smr.GetCurrentView() == 0 {
		return nodeA
	}
	return smr.Election.GetLeader(smr.GetCurrentView())
}

func NewFakeLogger(logid string) logs.Logger {
	confFile := utils.GetCurFileDir()
	confFile = filepath.Join(confFile, "config/log.yaml")
	logDir := utils.GetCurFileDir()
	logDir = filepath.Join(logDir, "logs")

	logs.InitLog(confFile, logDir)
	log, _ := logs.NewLogger(logid, "smr_test")
	return log
}

func CompeteLoop(smr *chainedBft.Smr, log logs.Logger, validators []string) {
	for {
		miner := CompeteMaster(smr)
		log.Info("Compete", "round", smr.GetCurrentView(), "leader", miner, "isMiner", miner == smr.GetAddress(), "address", smr.GetAddress())
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
		if err := smr.ProcessProposal(smr.GetCurrentView(), []byte(string(smr.GetCurrentView())), validators); err != nil {
			log.Error("Smr ProcessProposal error", "error", err)
		}
	}
}
