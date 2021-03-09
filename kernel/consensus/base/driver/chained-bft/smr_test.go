package chained_bft

import (
	"path/filepath"
	"testing"
	"time"

	_ "github.com/xuperchain/xupercore/bcs/network/p2pv2"
	"github.com/xuperchain/xupercore/kernel/common/xconfig"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/network"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	LogPath  = filepath.Join(utils.GetCurFileDir(), "/main/test")
	NodePath = filepath.Join(utils.GetCurFileDir(), "../../../../mock/p2pv2")

	NodeA   = "gNhga8vLc4JcmoHB2yeef2adBhntkc5d1"
	NodeAIp = "/ip4/127.0.0.1/tcp/47101/p2p/QmVcSF4F7rTdsvUJqsik98tXRXMBUqL5DSuBpyYKVhjuG4"
	PubKeyA = "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571}"
	PriKeyA = "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571,\"D\":29079635126530934056640915735344231956621504557963207107451663058887647996601}"

	NodeB   = "TDYJN5mYuX8KR3RqRUi2MQWW7weQYdrcD"
	NodeBIp = "/ip4/127.0.0.1/tcp/47102/p2p/Qmd1sJ4s7JTfHvetfjN9vNE5fhkLESs42tYHc5RYUBPnEv"
	PubKeyB = `{"Curvname":"P-256","X":38583161743450819602965472047899931736724287060636876073116809140664442044200,"Y":73385020193072990307254305974695788922719491565637982722155178511113463088980}`
	PriKeyB = `{"Curvname":"P-256","X":38583161743450819602965472047899931736724287060636876073116809140664442044200,"Y":73385020193072990307254305974695788922719491565637982722155178511113463088980,"D":98698032903818677365237388430412623738975596999573887926929830968230132692775}`

	NodeC   = "kDyW3By3FreKosnNyjPc18CFW2EafuPV8"
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
	path := filepath.Join(LogPath, Npath)
	confFile := filepath.Join(path, "log.yaml")
	logDir := filepath.Join(path, "/logs")

	logs.InitLog(confFile, logDir)
	log, _ := logs.NewLogger(node, "smr_test")
	return log
}

func InitQcTee() *QCPendingTree {
	initQC := &QuorumCert{
		VoteInfo: &VoteInfo{
			ProposalId:   []byte{0},
			ProposalView: 0,
		},
		LedgerCommitInfo: &LedgerCommitInfo{
			CommitStateId: []byte{0},
		},
	}
	rootNode := &ProposalNode{
		In: initQC,
	}
	return &QCPendingTree{
		Genesis:  rootNode,
		Root:     rootNode,
		HighQC:   rootNode,
		CommitQC: rootNode,
	}
}

func NewP2P(node string, t *testing.T) network.Network {
	// 创建p2p
	var Npath string
	switch node {
	case "nodeA":
		Npath = "node1"
	case "nodeB":
		Npath = "node2"
	case "nodeC":
		Npath = "node3"
	}
	path := filepath.Join(NodePath, Npath)
	econfPath := filepath.Join(path, "conf/env.yaml")
	ecfg, err := xconfig.LoadEnvConf(econfPath)
	if err != nil {
		t.Error("LoadEnvConf error", "error", err)
		return nil
	}
	netCtx, err := nctx.NewNetCtx(ecfg)
	if err != nil {
		t.Error("NewNetCtx error", "error", err)
		return nil
	}
	p2p, err := network.NewNetwork(netCtx)
	return p2p
}

func NewCryptoClient(node string, t *testing.T) (cctx.Address, cctx.CryptoClient) {
	var priKeyStr, pubKeyStr, addr string
	switch node {
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
		t.Error("CreateCryptoClientFromJSONPrivateKey error", "error", err)
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

func NewSMR(node string, log logs.Logger, p2p network.Network, t *testing.T) *Smr {
	a, cc := NewCryptoClient(node, t)
	cryptoClient := cCrypto.NewCBFTCrypto(&a, cc)
	pacemaker := &DefaultPaceMaker{}
	q := InitQcTee()
	saftyrules := &DefaultSaftyRules{
		Crypto: cryptoClient,
		QcTree: q,
	}
	election := &ElectionA{
		addrs: []string{NodeA, NodeB, NodeC},
	}
	s := NewSmr("xuper", a.Address, log, p2p, cryptoClient, pacemaker, saftyrules, election, q, nil)
	if s == nil {
		t.Error("NewSmr1 error")
		return nil
	}
	return s
}

func TestNewSmr(t *testing.T) {
	log := NewFakeLogger("nodeA")
	p := NewP2P("nodeA", t)
	s := NewSMR("nodeA", log, p, t)
	if s == nil {
		t.Error("NewSmr error")
	}
}

func TestProcessProposalSingle(t *testing.T) {
	log := NewFakeLogger("nodeA")
	p := NewP2P("nodeA", t)
	s := NewSMR("nodeA", log, p, t)
	NewP2P("nodeB", t)
	NewSMR("nodeB", log, p, t)
	NewP2P("nodeC", t)
	NewSMR("nodeC", log, p, t)
	// 相同proposalId不能重复提交
	err := s.ProcessProposal(1, []byte{0}, []string{NodeAIp})
	if err != SameProposalNotify {
		t.Error("ProcessProposal error", "error", err)
	}
}

func TestSMR(t *testing.T) {
	logA := NewFakeLogger("nodeA")
	logB := NewFakeLogger("nodeB")
	logC := NewFakeLogger("nodeC")
	pA := NewP2P("nodeA", t)
	go pA.Start()
	pB := NewP2P("nodeB", t)
	go pB.Start()
	pC := NewP2P("nodeC", t)
	go pC.Start()
	sA := NewSMR("nodeA", logA, pA, t)
	go sA.Start()
	sB := NewSMR("nodeB", logB, pB, t)
	go sB.Start()
	sC := NewSMR("nodeC", logC, pC, t)
	go sC.Start()
	time.Sleep(time.Second * 10)

	// 模拟第一个Proposal交互
	err := sA.ProcessProposal(1, []byte{1}, []string{NodeAIp, NodeBIp, NodeCIp})
	if err != nil {
		t.Error("ProcessProposal error", "error", err)
		return
	}
	time.Sleep(time.Second * 30)
	// 检查存储
	// A --- B --- C
	//      收集A
	//            收集B
	// 检查本地qcTree
	nodeAH := sA.qcTree.GetHighQC()
	aiV := nodeAH.In.GetProposalView()
	if aiV != 0 {
		t.Error("update qcTree error", "aiV", aiV)
		return
	}
	/*
		// 检查B节点，B节点收集A发起的1轮qc，A的票应该有3张，B应该进入2轮
		nodeBH := sB.qcTree.GetHighQC()
		biV := nodeBH.In.GetProposalView()
		if biV != 1 {
			t.Error("update qcTree error", "biV", aiV)
			return
		}
		if sB.GetCurrentView() != 2 {
			t.Error("receive B ProcessProposal error", "view", sB.GetCurrentView())
			return
		}
		if sC.GetCurrentView() != 1 {
			t.Error("receive C ProcessProposal error", "view", sC.GetCurrentView())
			return
		}
		// ABC节点应该都存储了新的view=1的node，但是只有B更新了HighQC
		if len(nodeAH.Sons) != 1 {
			t.Error("A qcTree error")
			return
		}
		nodeCH := sC.qcTree.GetHighQC()
		if len(nodeCH.Sons) != 1 {
			t.Error("A qcTree error")
			return
		}

		// 模拟第二个Proposal交互, 此时由B节点发出
		// ABC节点应该都存储了新的view=2的node，但是只有C更新了HighQC
		err = sB.ProcessProposal(2, []byte{2}, []string{NodeAIp, NodeBIp, NodeCIp})
		if err != nil {
			t.Error("ProcessProposal error", "error", err)
			return
		}
		time.Sleep(time.Second * 30)
		nodeAH = sA.qcTree.GetHighQC()
		nodeBH = sB.qcTree.GetHighQC()
		nodeCH = sC.qcTree.GetHighQC()
		if nodeAH.In.GetProposalView() != 1 || nodeBH.In.GetProposalView() != 1 || nodeCH.In.GetProposalView() != 2 {
			t.Error("Round2 update HighQC error", "nodeAH", nodeAH.In.GetProposalView(), "nodeBH", nodeBH.In.GetProposalView(), "nodeCH", nodeCH.In.GetProposalView())
			return
		}

		// 模拟第三个Proposal交互, 此时模拟一个分叉情况，除B之外，A也创建了一个高度为2的块
		// 注意，由于本状态机支持回滚，因此round可重复
		// 注意，为了支持回滚操作，必须调用smr的UpdateJustifyQcStatus
		// 次数round1的全部选票在B手中
		vote := &VoteInfo{
			ProposalId:   []byte{1},
			ProposalView: 1,
			ParentId:     []byte{0},
			ParentView:   0,
		}
		v, ok := sB.qcVoteMsgs.Load(utils.F(vote.ProposalId))
		if !ok {
			t.Error("B votesMsg error")
		}
		signs, ok := v.([]*chainedBftPb.QuorumCertSign)
		if !ok {
			t.Error("B votesMsg transfer error")
		}
		justi := &QuorumCert{
			VoteInfo:  vote,
			SignInfos: signs,
		}
		sA.UpdateJustifyQcStatus(justi)
		sB.UpdateJustifyQcStatus(justi)
		sC.UpdateJustifyQcStatus(justi)

		err = sA.ProcessProposal(2, []byte{3}, []string{NodeAIp, NodeBIp, NodeCIp})
		if err != nil {
			t.Error("ProcessProposal error", "error", err)
			return
		}
		time.Sleep(time.Second * 30)
		nodeCH = sC.qcTree.GetHighQC()
		if !bytes.Equal(nodeCH.In.GetProposalId(), []byte{3}) {
			t.Error("ProcessProposal error", "id", nodeCH.In.GetProposalId())
		}
		nodeBH = sB.qcTree.GetHighQC()
		if len(nodeBH.Sons) != 2 {
			t.Error("ProcessProposal error")
		}
		nodeAH = sA.qcTree.GetHighQC()
		if len(nodeAH.Sons) != 2 {
			t.Error("ProcessProposal error", "highQC", nodeAH.In.GetProposalView())
		}
	*/
}
