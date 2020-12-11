package chained_bft

import (
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/xuperchain/crypto/core/hash"
	"github.com/xuperchain/xuperchain/core/global"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	chainedBftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	xuperp2p "github.com/xuperchain/xupercore/kernel/network/pb"
	"github.com/xuperchain/xupercore/lib/logs"
)

var (
	TooLowNewView      = errors.New("nextView is lower than local pacemaker's currentView.")
	P2PInternalErr     = errors.New("Internal err in p2p module.")
	TooLowNewProposal  = errors.New("Proposal is lower than local pacemaker's currentView.")
	EmptyHighQC        = errors.New("No valid highQC in qcTree.")
	EmptyViewNotify    = errors.New("No NewView Msg valid.")
	SameProposalNotify = errors.New("Same proposal has been made.")
	JustifyVotesEmpty  = errors.New("justify qc's votes are empty.")
)

const (
	// DefaultNetMsgChanSize is the default size of network msg channel
	DefaultNetMsgChanSize = 1000
)

// TODO: TimeoutQC, 持久化存储问题

/* smr 组装了三个模块: pacemaker、saftyrules和propose election
 * smr有自己的存储即PendingTree
 * 原本的ChainedBft(联结smr和本地账本，在preferredVote被确认后, 触发账本commit操作)
 * 被替代成smr和上层bcs账本的·组合实现，以减少不必要的代码，考虑到chained-bft暂无扩展性
 */
type Smr struct {
	bcName  string
	log     logs.Logger
	address string // 包含一个私钥生成的地址
	// smr定义了自己需要的P2P消息类型
	// p2pMsgChan is the msg channel registered to network
	p2pMsgChan chan *xuperp2p.XuperMessage
	// subscribeList is the Subscriber list of the srm instance
	subscribeList *list.List
	// p2p interface
	p2p cctx.P2pCtxInConsensus
	// cBFTCrypto 封装了本BFT需要的加密相关的接口和方法
	cryptoClient *cCrypto.CBFTCrypto

	// quitCh stop channel
	QuitCh chan bool

	pacemaker  PacemakerInterface
	saftyrules saftyRulesInterface
	Election   ProposerElectionInterface
	qcTree     *QCPendingTree

	// map[proposalId]bool
	localProposal *sync.Map
	// votes of QC in mem, key: voteId, value: []*QuorumCertSign
	qcVoteMsgs *sync.Map
	// new view msg gathered from other replicas, key: viewNumber, value: []*pb.ChainedBftPhaseMessage
	newViewMsgs *sync.Map
}

func NewSmr(bcName, address string, log logs.Logger, p2p cctx.P2pCtxInConsensus, cryptoClient *cCrypto.CBFTCrypto, pacemaker PacemakerInterface,
	saftyrules saftyRulesInterface, election ProposerElectionInterface, qcTree *QCPendingTree) *Smr {
	return &Smr{
		bcName:        bcName,
		log:           log,
		address:       address,
		p2pMsgChan:    make(chan *xuperp2p.XuperMessage, DefaultNetMsgChanSize),
		subscribeList: list.New(),
		p2p:           p2p,
		cryptoClient:  cryptoClient,
		QuitCh:        make(chan bool, 1),
		pacemaker:     pacemaker,
		saftyrules:    saftyrules,
		Election:      election,
		qcTree:        qcTree,
		localProposal: &sync.Map{},
		qcVoteMsgs:    &sync.Map{},
		newViewMsgs:   &sync.Map{},
	}
}

var (
	RegisterErr = errors.New("Register to p2p error")
)

// RegisterToNetwork register msg handler to p2p network
func (s *Smr) RegisterToNetwork() error {
	sub1 := s.p2p.NewSubscriber(xuperp2p.XuperMessage_CHAINED_BFT_NEW_VIEW_MSG, s.p2pMsgChan)
	if err := s.p2p.Register(sub1); err != nil {
		return err
	}
	s.subscribeList.PushBack(sub1)
	sub2 := s.p2p.NewSubscriber(xuperp2p.XuperMessage_CHAINED_BFT_NEW_PROPOSAL_MSG, s.p2pMsgChan)
	if err := s.p2p.Register(sub2); err != nil {
		return err
	}
	s.subscribeList.PushBack(sub2)
	sub3 := s.p2p.NewSubscriber(xuperp2p.XuperMessage_CHAINED_BFT_VOTE_MSG, s.p2pMsgChan)
	if err := s.p2p.Register(sub3); err != nil {
		return err
	}
	s.subscribeList.PushBack(sub3)
	return nil
}

// UnRegisterToNetwork unregister msg handler to p2p network
func (s *Smr) UnRegisterToNetwork() {
	var e *list.Element
	for i := 0; i < s.subscribeList.Len(); i++ {
		e = s.subscribeList.Front()
		next := e.Next()
		sub, _ := e.Value.(p2p.Subscriber)
		if err := s.p2p.UnRegister(sub); err == nil {
			s.subscribeList.Remove(e)
		}
		e = next
	}
}

// Start used to start smr instance and process msg
func (s *Smr) Start() {
	for {
		select {
		case msg := <-s.p2pMsgChan:
			s.handleReceivedMsg(msg)
		case <-s.QuitCh:
			return
		}
	}
}

// stop used to stop smr instance
func (s *Smr) Stop() {
	s.QuitCh <- true
	s.UnRegisterToNetwork()
}

// handleReceivedMsg used to process msg received from network
func (s *Smr) handleReceivedMsg(msg *xuperp2p.XuperMessage) error {
	// filter msg from other chain
	/*
		if msg.GetHeader().GetBcname() != s.bcName {
			return nil
		}
	*/
	switch msg.GetHeader().GetType() {
	case xuperp2p.XuperMessage_CHAINED_BFT_NEW_VIEW_MSG:
		go s.handleReceivedNewView(msg)
	case xuperp2p.XuperMessage_CHAINED_BFT_NEW_PROPOSAL_MSG:
		go s.handleReceivedProposal(msg)
	case xuperp2p.XuperMessage_CHAINED_BFT_VOTE_MSG:
		go s.handleReceivedVoteMsg(msg)
	default:
		s.log.Error("smr::handleReceivedMsg receive unknow type msg", "type", msg.GetHeader().GetType())
		return nil
	}
	return nil
}

/* ProcessNewView是本地Chained-HotStuff实现的特殊逻辑。由上一轮的Leader和其他Replica触发
 * ProcessNewView的作用是其他节点发送一个消息去提醒下一个Proposer，提醒该节点去AdvanceViw并且发起一个新Proposal
 * ATTENTION: 本function的语义是本地节点去提醒下一个Leader, 和HotStuff论文中的NewView无关
 */
func (s *Smr) ProcessNewView(nextView int64, nextLeader string) error {
	// if new view number less than voted view number, return error
	if nextView < s.pacemaker.GetCurrentView() {
		s.log.Error("smr::ProcessNewView::input param nextView err")
		return TooLowNewView
	}

	justifyQC := QuorumCert{}
	justifyBytes, _ := json.Marshal(justifyQC)

	newViewMsg := &chainedBftPb.ProposalMsg{
		ProposalView: nextView,
		JustifyQC:    justifyBytes,
		Sign: &chainedBftPb.QuorumCertSign{
			Address:   s.address,
			PublicKey: s.cryptoClient.PkJson,
		},
	}
	newViewMsg, err := s.cryptoClient.SignProposalMsg(newViewMsg)
	if err != nil {
		s.log.Error("smr::ProcessNewView::SignProposalMsg error", "error", err)
		return err
	}
	netMsg := p2p.NewMessage(xuperp2p.XuperMessage_CHAINED_BFT_NEW_VIEW_MSG, newViewMsg, p2p.WithBCName(s.bcName))
	// 全部预备之后，再调用该接口
	if netMsg == nil {
		s.log.Error("smr::ProcessNewView::NewMessage error")
		return P2PInternalErr
	}
	s.pacemaker.PrepareAdvance(nextView, nextLeader)
	go s.p2p.SendMessage(context.Background(), netMsg, p2p.WithAddresses([]string{s.Election.GetMsgAddress(nextLeader)}))
	return nil
}

/* handleReceivedNewView NewView消息实际是一个“通知类”proposal消息
 */
func (s *Smr) handleReceivedNewView(msg *xuperp2p.XuperMessage) error {
	newViewMsg := &chainedBftPb.ProposalMsg{}
	if err := p2p.Unmarshal(msg, newViewMsg); err != nil {
		s.log.Error("smr::handleReceivedNewView Unmarshal msg error", "logid", msg.GetHeader().GetLogid(), "error", err)
		return err
	}
	s.newViewMsgs.LoadOrStore(global.F(newViewMsg.GetProposalId()), true)
	return nil
}

/* ProcessProposal 即Chained-HotStuff的NewView阶段，LibraBFT的process_proposal阶段
 * 对于一个认为自己当前是Leader的节点，它试图生成一个新的提案，即一个新的QC，并广播
 * 本节点产生一个Proposal，该proposal包含一个最新的round, 最新的proposalId，一个parentQC，并将该消息组合成一个ProposalMsg消息给所有节点
 * 全部完成后leader更新本地localProposal
 */
func (s *Smr) ProcessProposal(viewNumber int64, proposalID []byte, validatesIpInfo []string) error {
	// ATTENTION::TODO:: 由于本次设计面向的是viewNumber可能重复的BFT，因此账本回滚后高度会相同，在此用LockedQC高度为标记
	if s.qcTree.GetLockedQC() != nil && s.pacemaker.GetCurrentView() < s.qcTree.GetLockedQC().In.GetProposalView() {
		return TooLowNewProposal
	}
	if s.qcTree.GetHighQC() == nil {
		return EmptyHighQC
	}
	if _, ok := s.localProposal.Load(global.F(proposalID)); ok {
		return SameProposalNotify
	}
	parentQuorumCert, err := s.reloadJustifyQC()
	if err != nil {
		return err
	}
	parentQuorumCertBytes, err := json.Marshal(parentQuorumCert)
	if err != nil {
		return err
	}
	proposal := &chainedBftPb.ProposalMsg{
		ProposalView: viewNumber,
		ProposalId:   proposalID,
		Timestamp:    time.Now().UnixNano(),
		JustifyQC:    parentQuorumCertBytes,
	}
	propMsg, err := s.cryptoClient.SignProposalMsg(proposal)
	if err != nil {
		s.log.Error("smr::ProcessProposal SignProposalMsg error", "error", err)
		return err
	}
	netMsg := p2p.NewMessage(xuperp2p.XuperMessage_CHAINED_BFT_NEW_PROPOSAL_MSG, propMsg, p2p.WithBCName(s.bcName))
	// 全部预备之后，再调用该接口
	if netMsg == nil {
		s.log.Error("smr::ProcessProposal::NewMessage error")
		return P2PInternalErr
	}
	go s.p2p.SendMessage(context.Background(), netMsg, p2p.WithAddresses(validatesIpInfo))
	s.log.Info("smr:ProcessProposal::new proposal has been made", "address", s.address, "proposalID", global.F(proposalID))
	return nil
}

func (s *Smr) reloadJustifyQC() (*QuorumCert, error) {
	highQC := s.qcTree.GetHighQC()
	v := &VoteInfo{
		ProposalView: highQC.In.GetProposalView(),
		ProposalId:   highQC.In.GetProposalId(),
	}
	// 第一次proposal，highQC==rootQC==genesisQC
	if bytes.Equal(s.qcTree.Genesis.In.GetProposalId(), highQC.In.GetProposalId()) {
		return &QuorumCert{VoteInfo: v}, nil
	}
	// 此时highQC一定有parent， TODO：边界错误
	v.ParentView = highQC.Parent.In.GetProposalView()
	v.ParentId = highQC.Parent.In.GetProposalId()
	// 查看qcTree是否包含当前可以commit的Id
	var commitId []byte
	if s.qcTree.GetCommitQC() != nil {
		commitId = s.qcTree.GetCommitQC().In.GetProposalId()
	}
	// 根据qcTree生成一个parentQC
	// 上一个view的votes
	voteId := GetVoteId(v)
	value, ok := s.qcVoteMsgs.Load(global.F(voteId))
	if !ok {
		return nil, JustifyVotesEmpty
	}
	parentQuorumCert := &QuorumCert{
		VoteInfo: v,
		LedgerCommitInfo: &LedgerCommitInfo{
			CommitStateId: commitId,
		},
	}
	signs, ok := value.([]*chainedBftPb.QuorumCertSign)
	if ok {
		parentQuorumCert.SignInfos = signs
	}
	return parentQuorumCert, nil
}

/* handleReceivedProposal 该阶段在收到一个ProposalMsg后触发，与LibraBFT的process_proposal阶段类似
 * 该阶段分两个角色，一个是认为自己是currentRound的Leader，一个是Replica
 * 1. 比较本地pacemaker是否需要AdvanceRound
 * 2. 查看ProposalMsg消息的合法性，检查qcTree是否需要更新CommitQC
 * 3. 检查本地计算Leader和该新QC的Leader是否相等
 * 4. 验证Leader和本地计算的Leader是否相等
 * 5.向本地PendingTree插入该QC，即更新QC
 * 6.发送一个vote消息给下一个Leader
 */
func (s *Smr) handleReceivedProposal(msg *xuperp2p.XuperMessage) {
	newProposalMsg := &chainedBftPb.ProposalMsg{}
	if err := p2p.Unmarshal(msg, newProposalMsg); err != nil {
		s.log.Error("smr::handleReceivedProposal Unmarshal msg error", "logid", msg.GetHeader().GetLogid(), "error", err)
		return
	}
	if _, ok := s.localProposal.LoadOrStore(global.F(newProposalMsg.GetProposalId()), true); ok {
		return
	}

	s.log.Info("smr::handleReceivedProposal::received a proposal", "logid", msg.GetHeader().GetLogid(),
		"newView", newProposalMsg.GetProposalView(), "newProposalId", global.F(newProposalMsg.GetProposalId()))
	parentQCBytes := newProposalMsg.GetJustifyQC()
	parentQC := &QuorumCert{}
	if err := json.Unmarshal(parentQCBytes, parentQC); err != nil {
		s.log.Error("smr::handleReceivedProposal Unmarshal parentQC error", "error", err)
		return
	}

	newVote := &VoteInfo{
		ProposalId:   newProposalMsg.GetProposalId(),
		ProposalView: newProposalMsg.GetProposalView(),
		ParentId:     parentQC.GetProposalId(),
		ParentView:   parentQC.GetProposalView(),
	}
	isFirstJustify := bytes.Equal(s.qcTree.Genesis.In.GetProposalId(), parentQC.GetProposalId())
	// 若为初始状态，则无需检查justify，否则需要检查qc有效性
	if !isFirstJustify {
		if err := s.saftyrules.IsQuorumCertValidate(&QuorumCert{
			VoteInfo:  newVote,
			SignInfos: []*chainedBftPb.QuorumCertSign{newProposalMsg.GetSign()},
		}, parentQC, s.Election.GetValidators(parentQC.GetProposalView())); err != nil {
			return
		}
	}
	// 本地pacemaker试图更新currentView, 并返回一个是否需要将新消息通知该轮Leader, 是该轮不是下轮！主要解决P2PIP端口不能通知Loop的问题
	sendMsg, _ := s.pacemaker.AdvanceView(parentQC)
	// 通知current Leader
	if sendMsg {
		netMsg := p2p.NewMessage(xuperp2p.XuperMessage_CHAINED_BFT_NEW_PROPOSAL_MSG, newProposalMsg, p2p.WithBCName(s.bcName))
		leader := s.Election.GetLeader(newProposalMsg.GetProposalView())
		// 此处如果失败，仍会执行下层逻辑，因为是多个节点通知该轮Leader，因此若发不出去仍可继续运行
		if leader != "" && netMsg != nil && leader != s.address {
			go s.p2p.SendMessage(context.Background(), netMsg, p2p.WithAddresses([]string{s.Election.GetMsgAddress(leader)}))
		}
	}

	// 获取parentQC对应的本地节点
	parentNode := s.qcTree.DFSQueryNode(parentQC.GetProposalId())
	// 本地safetyrules更新, 如有可以commit的QC，执行commit操作并更新本地rootQC
	valid := s.saftyrules.UpdatePreferredRound(parentQC)
	if parentQC.LedgerCommitInfo != nil && parentQC.LedgerCommitInfo.CommitStateId != nil && valid && parentNode != nil {
		if parentNode.Parent != nil && parentNode.Parent.Parent != nil {
			s.qcTree.updateCommit(parentNode.Parent.Parent.In)
		}
	}
	// 验证Proposal的Round是否和pacemaker的Round相等
	if newProposalMsg.GetProposalView() != s.pacemaker.GetCurrentView() {
		s.log.Error("smr::handleReceivedProposal::error", "error", TooLowNewProposal)
		return
	}
	// 验证本地election计算出来的当前round Leader是否和Proposal的Leader相等
	leader := s.Election.GetLeader(s.pacemaker.GetCurrentView())
	if leader == "" || leader != newProposalMsg.Sign.Address {
		s.log.Error("smr::handleReceivedProposal::leader error", "want", leader, "have", newProposalMsg.Sign.Address)
		return
	}
	// 根据本地saftyrules返回是否需要发送voteMsg给下一个Leader
	if !s.saftyrules.VoteProposal(newProposalMsg.GetProposalId(), newProposalMsg.GetProposalView(), parentQC) {
		return
	}

	// 这个newVoteId表示的是本地最新一次vote的id，生成voteInfo的hash，标识vote消息
	newVoteId := GetVoteId(newVote)
	newLedgerInfo := &LedgerCommitInfo{
		VoteInfoHash: newVoteId,
	}
	newNode := &ProposalNode{
		In: &QuorumCert{
			VoteInfo:         newVote,
			LedgerCommitInfo: newLedgerInfo,
		},
	}
	if parentNode != nil {
		newNode.Parent = parentNode
	}
	// 与proposal.ParentId相比，更新本地qcTree，insert新节点, 包括更新CommitQC等等
	if err := s.qcTree.updateQcStatus(newNode); err != nil {
		return
	}
	// 此时更新node的commitStateId
	if s.qcTree.GetCommitQC() != nil {
		newLedgerInfo.CommitStateId = s.qcTree.GetCommitQC().In.GetProposalId()
	}
	s.log.Info("pacemaker!!!!", "round", s.pacemaker.GetCurrentView())
	nextLeader := s.Election.GetLeader(s.pacemaker.GetCurrentView() + 1)
	if nextLeader == "" {
		s.log.Info("smr::handleReceivedProposal::empty next leader", "next round", s.pacemaker.GetCurrentView()+1)
		return
	}
	logid := fmt.Sprintf("%x", newVoteId) + "_" + s.address
	s.voteProposal(newProposalMsg.GetProposalId(), newVote, newLedgerInfo, nextLeader, logid)
}

/* voteProposal 当Replica收到一个Proposal并对该Proposal检查之后，该节点会针对该QC投票
 * 节点的vote包含一个本次vote的对象的基本信息，和本地上次vote对象的基本信息，和本地账本的基本信息，和一个签名
 */
func (s *Smr) voteProposal(msg []byte, vote *VoteInfo, ledger *LedgerCommitInfo, voteTo, logid string) {
	// 这里存在自己给自己投票的情况，给自己投票，直接操作存储，记得签名
	if voteTo == s.address {
		v, ok := s.qcVoteMsgs.Load(global.F(GetVoteId(vote)))
		var signs []*chainedBftPb.QuorumCertSign
		if ok {
			signs, _ = v.([]*chainedBftPb.QuorumCertSign)
		}
		nextSign, err := s.cryptoClient.SignVoteMsg(msg)
		if err != nil {
			return
		}
		signs = append(signs, nextSign)
		s.qcVoteMsgs.Store(global.F(GetVoteId(vote)), signs)
	}

	voteBytes, err := json.Marshal(vote)
	if err != nil {
		return
	}
	ledgerBytes, err := json.Marshal(ledger)
	if err != nil {
		return
	}
	sig, err := s.cryptoClient.SignVoteMsg(msg)
	if err != nil {
		return
	}
	voteMsg := &chainedBftPb.VoteMsg{
		VoteInfo:         voteBytes,
		LedgerCommitInfo: ledgerBytes,
		Signature:        []*chainedBftPb.QuorumCertSign{sig},
	}
	netMsg := p2p.NewMessage(xuperp2p.XuperMessage_CHAINED_BFT_VOTE_MSG, voteMsg, p2p.WithBCName(s.bcName))
	// 全部预备之后，再调用该接口
	if netMsg == nil {
		s.log.Error("smr::ProcessProposal::NewMessage error")
		return
	}
	go s.p2p.SendMessage(context.Background(), netMsg, p2p.WithAddresses([]string{s.Election.GetMsgAddress(voteTo)}))
	return
}

/* handleReceivedVoteMsg 当前Leader在发送一个proposal消息之后，由下一Leader等待周围replica的投票，收集vote消息
 * 当收到2f+1个vote消息之后，本地pacemaker调用AdvanceView，并更新highQC
 * 该方法针对Leader而言
 */
func (s *Smr) handleReceivedVoteMsg(msg *xuperp2p.XuperMessage) error {
	newVoteMsg := &chainedBftPb.VoteMsg{}
	if err := p2p.Unmarshal(msg, newVoteMsg); err != nil {
		s.log.Error("smr::handleReceivedVoteMsg Unmarshal msg error", "logid", msg.GetHeader().GetLogid(), "error", err)
		return err
	}
	voteQC, err := s.VoteMsgToQC(newVoteMsg)
	if err != nil {
		s.log.Error("smr::handleReceivedVoteMsg VoteMsgToQC error", "error", err)
		return err
	}
	// 检查logid、voteInfoHash是否正确
	if err := s.saftyrules.CheckVoteMsg(voteQC, msg.GetHeader().GetLogid(), s.Election.GetValidators(voteQC.GetProposalView())); err != nil {
		s.log.Error("smr::handleReceivedVoteMsg CheckVoteMsg error", "error", err, "msg", voteQC.GetProposalId())
		return err
	}

	// 存入本地voteInfo内存，查看签名数量是否超过2f+1
	var VoteLen int
	// 注意隐式，若!ok则证明签名数量为1，此时不可能超过2f+1
	v, ok := s.qcVoteMsgs.LoadOrStore(global.F(voteQC.LedgerCommitInfo.VoteInfoHash), voteQC.SignInfos)
	if ok {
		signs, _ := v.([]*chainedBftPb.QuorumCertSign)
		signs = append(signs, voteQC.SignInfos[0])
		VoteLen = len(signs)
		s.qcVoteMsgs.Store(global.F(voteQC.LedgerCommitInfo.VoteInfoHash), signs)
	}
	// 查看签名数量是否达到2f+1, 需要获取justify对应的validators
	if !s.saftyrules.CalVotesThreshold(VoteLen, len(s.Election.GetValidators(voteQC.GetProposalView()))) {
		return nil
	}
	s.log.Info("FULL VOTE!!!!")
	// 更新本地pacemaker AdvanceRound
	s.pacemaker.AdvanceView(voteQC)
	s.log.Info("pacemaker!!!!", "round", s.pacemaker.GetCurrentView())
	// 更新HighQC
	s.qcTree.updateHighQC(voteQC.GetProposalId())
	return nil
}

// VoteMsgToQC 提供一个从VoteMsg转化为quorumCert的方法，注意，两者struct其实相仿
func (s *Smr) VoteMsgToQC(msg *chainedBftPb.VoteMsg) (*QuorumCert, error) {
	voteInfo := &VoteInfo{}
	if err := json.Unmarshal(msg.VoteInfo, voteInfo); err != nil {
		return nil, err
	}
	ledgerCommitInfo := &LedgerCommitInfo{}
	if err := json.Unmarshal(msg.LedgerCommitInfo, ledgerCommitInfo); err != nil {
		return nil, err
	}
	return &QuorumCert{
		VoteInfo:         voteInfo,
		LedgerCommitInfo: ledgerCommitInfo,
		SignInfos:        msg.GetSignature(),
	}, nil
}

func (s *Smr) GetCurrentView() int64 {
	return s.pacemaker.GetCurrentView()
}

func (s *Smr) GetAddress() string {
	return s.address
}

func (s *Smr) BlockToProposalNode(block cctx.BlockInterface) *ProposalNode {
	targetId := block.GetBlockid()
	if node := s.qcTree.DFSQueryNode(targetId); node != nil {
		return node
	}
	preId := block.GetPreHash()
	parentNode := s.qcTree.DFSQueryNode(preId)
	if parentNode != nil {
		return &ProposalNode{
			In: &QuorumCert{
				VoteInfo: &VoteInfo{
					ProposalId:   block.GetBlockid(),
					ProposalView: block.GetHeight(),
					ParentId:     parentNode.In.GetProposalId(),
					ParentView:   parentNode.In.GetProposalView(),
				},
			},
			Parent: parentNode,
		}
	}
	return nil
}

func (s *Smr) UpdateQcStatus(node *ProposalNode) error {
	return s.qcTree.updateQcStatus(node)
}

func (s *Smr) GetSaftyRules() saftyRulesInterface {
	return s.saftyrules
}

func (s *Smr) GetPacemaker() PacemakerInterface {
	return s.pacemaker
}

func (s *Smr) GetHighQC() QuorumCertInterface {
	return s.qcTree.GetHighQC().In
}

func GetVoteId(vote *VoteInfo) []byte {
	voteBytes, _ := json.Marshal(vote)
	// 这个newVoteId表示的是本地最新一次vote的id，生成voteInfo的hash，标识vote消息
	return hash.DoubleSha256([]byte(voteBytes))
}
