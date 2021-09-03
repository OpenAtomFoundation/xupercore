package storage

import (
	pb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
)

// quorumCert 是HotStuff的基础结构，它表示了一个节点本地状态以及其余节点对该状态的确认
type QuorumCert struct {
	// 本次qc的vote对象，该对象中嵌入了上次的QCid，因此删除原有的ProposalMsg部分
	VoteInfo *VoteInfo
	// 当前本地账本的状态
	LedgerCommitInfo *LedgerCommitInfo
	// SignInfos is the signs of the leader gathered from replicas of a specifically certType.
	SignInfos []*pb.QuorumCertSign
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

func (qc *QuorumCert) GetSignsInfo() []*pb.QuorumCertSign {
	return qc.SignInfos
}
