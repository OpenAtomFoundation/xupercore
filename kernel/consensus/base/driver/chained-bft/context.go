package chained_bft

import (
	"bytes"
	"errors"

	chainedBftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var _ QuorumCertInterface = (*QuorumCert)(nil)

var (
	NoValidQC = errors.New("Target QC is empty.")
)

// 本文件定义了chained-bft下有关的数据结构和接口
// QuorumCertInterface 规定了pacemaker和saftyrules操作的qc接口
// QuorumCert 为一个QuorumCertInterface的实现 TODO: smr彻底接口化是否可能?
// QCPendingTree 规定了smr内存存储的组织形式，其为一个QC树状结构

// quorumCert 是HotStuff的基础结构，它表示了一个节点本地状态以及其余节点对该状态的确认
type QuorumCert struct {
	// 本次qc的vote对象，该对象中嵌入了上次的QCid，因此删除原有的ProposalMsg部分
	VoteInfo *VoteInfo
	// 当前本地账本的状态
	LedgerCommitInfo *LedgerCommitInfo
	// SignInfos is the signs of the leader gathered from replicas of a specifically certType.
	SignInfos []*chainedBftPb.QuorumCertSign
}

func (qc *QuorumCert) GetProposalView() int64 {
	return qc.VoteInfo.ProposalView
}

func (qc *QuorumCert) GetProposalId() []byte {
	return qc.VoteInfo.ProposalId
}

func (qc *QuorumCert) GetParentProposalId() []byte {
	return qc.VoteInfo.ParentId
}

func (qc *QuorumCert) GetParentView() int64 {
	return qc.VoteInfo.ParentView
}

func (qc *QuorumCert) GetSignsInfo() []*chainedBftPb.QuorumCertSign {
	return qc.SignInfos
}

// VoteInfo 包含了本次和上次的vote对象
type VoteInfo struct {
	// 本次vote的对象
	ProposalId   []byte
	ProposalView int64
	// 本地上次vote的对象
	ParentId   []byte
	ParentView int64
}

// ledgerCommitInfo 表示的是本地账本和QC存储的状态，包含一个commitStateId和一个voteInfoHash
// commitStateId 表示本地账本状态，TODO: = 本地账本merkel root
// voteInfoHash 表示本地vote的vote_info的哈希，即本地QC的最新状态
type LedgerCommitInfo struct {
	CommitStateId []byte
	VoteInfoHash  []byte
}

//
type QuorumCertInterface interface {
	GetProposalView() int64
	GetProposalId() []byte
	GetParentProposalId() []byte
	GetParentView() int64
	GetSignsInfo() []*chainedBftPb.QuorumCertSign
}

// PendingTree 是一个内存内的QC状态存储树，仅存放目前未commit(即可能触发账本回滚)的区块信息
// 当PendingTree中的某个节点有[严格连续的]三代子孙后，将出发针对该节点的账本Commit操作
// 本数据结构替代原有Chained-BFT的三层QC存储，即proposalQC,generateQC和lockedQC
type QCPendingTree struct {
	Genesis   *ProposalNode // Tree中第一个Node
	Root      *ProposalNode
	HighQC    *ProposalNode // Tree中最高的QC指针
	GenericQC *ProposalNode
	LockedQC  *ProposalNode
	CommitQC  *ProposalNode

	Log logs.Logger
}

type ProposalNode struct {
	In QuorumCertInterface
	// Parent QuorumCertInterface
	Sons   []*ProposalNode
	Parent *ProposalNode
}

func (t *QCPendingTree) GetRootQC() *ProposalNode {
	return t.Root
}

func (t *QCPendingTree) GetHighQC() *ProposalNode {
	return t.HighQC
}

func (t *QCPendingTree) GetGenericQC() *ProposalNode {
	return t.GenericQC
}

func (t *QCPendingTree) GetCommitQC() *ProposalNode {
	return t.CommitQC
}

func (t *QCPendingTree) GetLockedQC() *ProposalNode {
	return t.LockedQC
}

// 更新本地qcTree, insert新节点, 将新节点parentQC和本地HighQC对比，如有必要进行更新
func (t *QCPendingTree) updateQcStatus(node *ProposalNode) error {
	if t.DFSQueryNode(node.In.GetProposalId()) != nil {
		t.Log.Debug("QCPendingTree::updateQcStatus::has been inserted", "search", utils.F(node.In.GetProposalId()))
		return nil
	}
	if err := t.insert(node); err != nil {
		t.Log.Error("QCPendingTree::updateQcStatus insert err", "err", err)
		return err
	}
	if node.Parent != nil {
		t.updateHighQC(node.Parent.In.GetProposalId())
	}
	t.Log.Debug("QCPendingTree::updateQcStatus", "insert new", utils.F(node.In.GetProposalId()), "height", node.In.GetProposalView(), "highQC", utils.F(t.GetHighQC().In.GetProposalId()))
	return nil
}

// updateHighQC 对比QC树，将本地HighQC和输入id比较，高度更高的更新为HighQC，此时连同GenericQC、LockedQC、CommitQC一起修改
func (t *QCPendingTree) updateHighQC(inProposalId []byte) {
	node := t.DFSQueryNode(inProposalId)
	if node == nil {
		t.Log.Error("QCPendingTree::updateHighQC::DFSQueryNode nil!")
		return
	}
	// 若新验证过的node和原HighQC高度相同，使用新验证的node
	if node.In.GetProposalView() < t.GetHighQC().In.GetProposalView() {
		return
	}
	// 更改HighQC以及一系列的GenericQC、LockedQC和CommitQC
	t.HighQC = node
	t.Log.Debug("QCPendingTree::updateHighQC", "HighQC height", t.HighQC.In.GetProposalView(), "HighQC", utils.F(t.HighQC.In.GetProposalId()))
	if node.Parent == nil {
		return
	}
	t.GenericQC = node.Parent
	t.Log.Debug("QCPendingTree::updateHighQC", "GenericQC height", t.GenericQC.In.GetProposalView(), "GenericQC", utils.F(t.GenericQC.In.GetProposalId()))
	// 找grand节点，标为LockedQC
	if node.Parent.Parent == nil {
		return
	}
	t.LockedQC = node.Parent.Parent
	t.Log.Debug("QCPendingTree::updateHighQC", "LockedQC height", t.LockedQC.In.GetProposalView(), "LockedQC", utils.F(t.LockedQC.In.GetProposalId()))
	// 找grandgrand节点，标为CommitQC
	if node.Parent.Parent.Parent == nil {
		return
	}
	t.CommitQC = node.Parent.Parent.Parent
	t.Log.Debug("QCPendingTree::updateHighQC", "CommitQC height", t.CommitQC.In.GetProposalView(), "CommitQC", utils.F(t.CommitQC.In.GetProposalId()))
}

// enforceUpdateHighQC 强制更改HighQC指针，用于错误时回滚，注意: 本实现没有timeoutQC因此需要此方法
func (t *QCPendingTree) enforceUpdateHighQC(inProposalId []byte) error {
	node := t.DFSQueryNode(inProposalId)
	if node == nil {
		t.Log.Debug("QCPendingTree::enforceUpdateHighQC::DFSQueryNode nil")
		return NoValidQC
	}
	// 更改HighQC以及一系列的GenericQC、LockedQC和CommitQC
	t.HighQC = node
	t.GenericQC = nil
	t.LockedQC = nil
	t.CommitQC = nil
	t.Log.Debug("QCPendingTree::enforceUpdateHighQC", "HighQC height", t.HighQC.In.GetProposalView(), "HighQC", utils.F(t.HighQC.In.GetProposalId()))
	if node.Parent == nil {
		return nil
	}
	t.GenericQC = node.Parent
	t.Log.Debug("QCPendingTree::enforceUpdateHighQC", "GenericQC height", t.GenericQC.In.GetProposalView(), "GenericQC", utils.F(t.GenericQC.In.GetProposalId()))
	// 找grand节点，标为LockedQC
	if node.Parent.Parent == nil {
		return nil
	}
	t.LockedQC = node.Parent.Parent
	t.Log.Debug("QCPendingTree::enforceUpdateHighQC", "LockedQC height", t.LockedQC.In.GetProposalView(), "LockedQC", utils.F(t.LockedQC.In.GetProposalId()))
	// 找grandgrand节点，标为CommitQC
	if node.Parent.Parent.Parent == nil {
		return nil
	}
	t.CommitQC = node.Parent.Parent.Parent
	t.Log.Debug("QCPendingTree::enforceUpdateHighQC", "CommitQC height", t.CommitQC.In.GetProposalView(), "CommitQC", utils.F(t.CommitQC.In.GetProposalId()))
	return nil
}

// insert 向本地QC树Insert一个ProposalNode，如有必要，连同HighQC、GenericQC、LockedQC、CommitQC一起修改
func (t *QCPendingTree) insert(node *ProposalNode) error {
	if node.Parent != nil {
		node.Parent.Sons = append(node.Parent.Sons, node)
	}
	return nil
}

// updateCommit 此方法向存储接口发送一个ProcessCommit，通知存储落盘，此时的block将不再被回滚
// 同时此方法将原先的root更改为commit node，因为commit node在本BFT中已确定不会回滚
func (t *QCPendingTree) updateCommit(p QuorumCertInterface) {
	// t.Ledger.ConsensusCommit(p.GetProposalId())
	node := t.DFSQueryNode(p.GetProposalId())
	parent := node.Parent
	node.Parent = nil
	if parent != nil {
		parent.Sons = nil
	}
	t.Root = node
	// TODO: commitQC/lockedQC/genericQC/highQC是否有指向原root及以上的Node
}

// DFSQueryNode实现的比较简单，从root节点开始寻找，后续有更优方法可优化
func (t *QCPendingTree) DFSQueryNode(id []byte) *ProposalNode {
	return DFSQuery(t.Root, id)
}

func DFSQuery(node *ProposalNode, target []byte) *ProposalNode {
	if bytes.Equal(node.In.GetProposalId(), target) {
		return node
	}
	if node.Sons == nil {
		return nil
	}
	for _, node := range node.Sons {
		if n := DFSQuery(node, target); n != nil {
			return n
		}
	}
	return nil
}
