package chained_bft

import (
	"bytes"
	"container/list"
	"path/filepath"
	"testing"
	"time"

	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	chainedBftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
	"github.com/xuperchain/xupercore/kernel/network"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	LogPath  = filepath.Join(utils.GetCurFileDir(), "/main/test")
	NodePath = filepath.Join(utils.GetCurFileDir(), "../../../../mock/p2pv2")

	NodeA   = "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"
	NodeAIp = "/ip4/127.0.0.1/tcp/38201/p2p/Qmf2HeHe4sspGkfRCTq6257Vm3UHzvh2TeQJHHvHzzuFw6"
	PubKeyA = `{"Curvname":"P-256","X":36505150171354363400464126431978257855318414556425194490762274938603757905292,"Y":79656876957602994269528255245092635964473154458596947290316223079846501380076}`
	PriKeyA = `{"Curvname":"P-256","X":36505150171354363400464126431978257855318414556425194490762274938603757905292,"Y":79656876957602994269528255245092635964473154458596947290316223079846501380076,"D":111497060296999106528800133634901141644446751975433315540300236500052690483486}`

	NodeB   = "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"
	NodeBIp = "/ip4/127.0.0.1/tcp/38202/p2p/QmQKp8pLWSgV4JiGjuULKV1JsdpxUtnDEUMP8sGaaUbwVL"
	PubKeyB = `{"Curvname":"P-256","X":12866043091588565003171939933628544430893620588191336136713947797738961176765,"Y":82755103183873558994270855453149717093321792154549800459286614469868720031056}`
	PriKeyB = `{"Curvname":"P-256","X":12866043091588565003171939933628544430893620588191336136713947797738961176765,"Y":82755103183873558994270855453149717093321792154549800459286614469868720031056,"D":74053182141043989390619716280199465858509830752513286817516873984288039572219}`

	NodeC   = "iYjtLcW6SVCiousAb5DFKWtWroahhEj4u"
	NodeCIp = "/ip4/127.0.0.1/tcp/38203/p2p/QmZXjZibcL5hy2Ttv5CnAQnssvnCbPEGBzqk7sAnL69R1E"
	PubKeyC = `{"Curvname":"P-256","X":71906497517774261659269469667273855852584750869988271615606376825756756449950,"Y":55040402911390674344019238894549124488349793311280846384605615474571192214233}`
	PriKeyC = `{"Curvname":"P-256","X":71906497517774261659269469667273855852584750869988271615606376825756756449950,"Y":55040402911390674344019238894549124488349793311280846384605615474571192214233,"D":88987246094484003072412401376409995742867407472451866878930049879250160571952}`
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

func InitQcTee(log logs.Logger) *QCPendingTree {
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
		Genesis:    rootNode,
		Root:       rootNode,
		HighQC:     rootNode,
		CommitQC:   rootNode,
		Log:        log,
		OrphanList: list.New(),
		OrphanMap:  make(map[string]bool),
	}
}

func NewFakeCryptoClient(node string, t *testing.T) (cctx.Address, cctx.CryptoClient) {
	var priKeyStr, pubKeyStr, addr string
	switch node {
	case "node":
		addr = NodeA
		pubKeyStr = PubKeyA
		priKeyStr = PriKeyA
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
	a, cc := NewFakeCryptoClient(node, t)
	cryptoClient := cCrypto.NewCBFTCrypto(&a, cc)
	pacemaker := &DefaultPaceMaker{}
	q := InitQcTee(log)
	saftyrules := &DefaultSaftyRules{
		Crypto: cryptoClient,
		QcTree: q,
	}
	election := &ElectionA{
		addrs: []string{NodeA, NodeB, NodeC},
	}
	s := NewSmr("xuper", a.Address, log, p2p, cryptoClient, pacemaker, saftyrules, election, q)
	if s == nil {
		t.Error("NewSmr1 error")
		return nil
	}
	return s
}

func TestSMR(t *testing.T) {
	logA := NewFakeLogger("nodeA")
	logB := NewFakeLogger("nodeB")
	logC := NewFakeLogger("nodeC")
	pA, ctxA, err := kmock.NewP2P("nodeA")
	pB, ctxB, err := kmock.NewP2P("nodeB")
	pC, ctxC, err := kmock.NewP2P("nodeC")
	pA.Init(ctxA)
	pB.Init(ctxB)
	pC.Init(ctxC)
	sA := NewSMR("nodeA", logA, pA, t)
	sB := NewSMR("nodeB", logB, pB, t)
	sC := NewSMR("nodeC", logC, pC, t)
	go pA.Start()
	go pB.Start()
	go pC.Start()
	go sA.Start()
	go sB.Start()
	go sC.Start()
	time.Sleep(time.Second * 10)

	// 模拟第一个Proposal交互
	err = sA.ProcessProposal(1, []byte{1}, []string{NodeA, NodeB, NodeC})
	if err != nil {
		t.Error("ProcessProposal error", "error", err)
		return
	}
	time.Sleep(time.Second * 10)
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

	// 检查B节点，B节点收集A发起的1轮qc，A的票应该有3张，B应该进入2轮
	nodeBH := sB.qcTree.GetHighQC()
	biV := nodeBH.In.GetProposalView()
	if biV != 1 {
		t.Error("update qcTree error", "biV", biV)
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
	err = sB.ProcessProposal(2, []byte{2}, []string{NodeA, NodeB, NodeC})
	if err != nil {
		t.Error("ProcessProposal error", "error", err)
		return
	}
	time.Sleep(time.Second * 10)
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

	err = sA.ProcessProposal(2, []byte{3}, []string{NodeA, NodeB, NodeC})
	if err != nil {
		t.Error("ProcessProposal error", "error", err)
		return
	}
	time.Sleep(time.Second * 10)
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
}
