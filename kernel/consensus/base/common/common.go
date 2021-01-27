package utils

import (
	"encoding/json"
	"errors"

	"github.com/golang/protobuf/proto"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	bftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/lib/logs"
)

var (
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
	EmptyJustify     = errors.New("Justify is empty.")
	InvalidJustify   = errors.New("Justify structure is invalid.")
)

// AddressEqual 判断两个validators地址是否相等
func AddressEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, _ := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// initQCTree 创建了smr需要的QC树存储，该Tree存储了目前待commit的QC信息
func InitQCTree(startHeight int64, ledger cctx.LedgerRely, log logs.Logger) *chainedBft.QCPendingTree {
	// 初始状态应该是start高度的前一个区块为genesisQC，即tipBlock
	// 由于重启后对于smr的内存存储(如qcVoteMap)将会清空，因此只能拿tipBlock获取tipHeight - 1高度的签名，重做tipBlock
	g, err := ledger.QueryBlockByHeight(startHeight - 1)
	if err != nil {
		return nil
	}
	gQC := &chainedBft.QuorumCert{
		VoteInfo: &chainedBft.VoteInfo{
			ProposalId:   g.GetBlockid(),
			ProposalView: g.GetHeight(),
		},
		LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
			CommitStateId: g.GetBlockid(),
		},
	}
	gNode := &chainedBft.ProposalNode{
		In: gQC,
	}
	r := ledger.GetTipBlock()
	// 当前为初始状态
	if r.GetHeight() == startHeight-1 {
		return &chainedBft.QCPendingTree{
			Genesis:  gNode,
			Root:     gNode,
			HighQC:   gNode,
			CommitQC: gNode,
			Log:      log,
		}
	}
	// 重启状态时将tipBlock-1装载到HighQC中
	r, err = ledger.QueryBlock(r.GetPreHash())
	if err != nil {
		return nil
	}
	rQC := &chainedBft.QuorumCert{
		VoteInfo: &chainedBft.VoteInfo{
			ProposalId:   r.GetBlockid(),
			ProposalView: r.GetHeight(),
		},
		LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
			CommitStateId: r.GetBlockid(),
		},
	}
	rNode := &chainedBft.ProposalNode{
		In: rQC,
	}
	return &chainedBft.QCPendingTree{
		Genesis:  gNode,
		Root:     rNode,
		HighQC:   rNode,
		CommitQC: rNode,
		Log:      log,
	}
}

/////////// lpb兼容逻辑 //////////

// 历史共识存储字段
type ConsensusStorage struct {
	Justify     *lpb.QuorumCert `json:"justify,omitempty"`
	CurTerm     int64           `json:"curTerm,omitempty"`
	CurBlockNum int64           `json:"curBlockNum,omitempty"`
}

// ParseOldQCStorage 将有Justify结构的老共识结构解析出来
func ParseOldQCStorage(storage []byte) (*ConsensusStorage, error) {
	old := &ConsensusStorage{}
	if err := json.Unmarshal(storage, &old); err != nil {
		return nil, err
	}
	return old, nil
}

// OldQCToNew 为老的QC pb结构转化为新的QC结构
func OldQCToNew(storage []byte) (*chainedBft.QuorumCert, error) {
	oldS, err := ParseOldQCStorage(storage)
	if err != nil {
		return nil, err
	}
	oldQC := oldS.Justify
	if oldQC == nil {
		return nil, InvalidJustify
	}
	justifyBytes := oldQC.ProposalMsg
	justifyQC := &lpb.QuorumCert{}
	err = proto.Unmarshal(justifyBytes, justifyQC)
	if err != nil {
		return nil, err
	}
	newQC := &chainedBft.QuorumCert{
		VoteInfo: &chainedBft.VoteInfo{
			ProposalId:   oldQC.ProposalId,
			ProposalView: oldQC.ViewNumber,
			ParentId:     justifyQC.ProposalId,
			ParentView:   justifyQC.ViewNumber,
		},
	}
	SignInfos := OldSignToNew(storage)
	newQC.SignInfos = SignInfos
	return newQC, nil
}

// NewToOldQC 为新的QC pb结构转化为老pb结构
func NewToOldQC(new *chainedBft.QuorumCert) (*lpb.QuorumCert, error) {
	oldParentQC := &lpb.QuorumCert{
		ProposalId: new.VoteInfo.ParentId,
		ViewNumber: new.VoteInfo.ParentView,
	}
	b, err := proto.Marshal(oldParentQC)
	if err != nil {
		return nil, err
	}
	oldQC := &lpb.QuorumCert{
		ProposalId:  new.VoteInfo.ProposalId,
		ViewNumber:  new.VoteInfo.ProposalView,
		ProposalMsg: b,
	}
	sign := NewSignToOld(new.GetSignsInfo())
	ss := &lpb.QCSignInfos{
		QCSignInfos: sign,
	}
	oldQC.SignInfos = ss
	return oldQC, nil
}

// OldSignToNew 老的签名结构转化为新的签名结构
func OldSignToNew(storage []byte) []*bftPb.QuorumCertSign {
	oldS, err := ParseOldQCStorage(storage)
	if err != nil {
		return nil
	}
	oldQC := oldS.Justify
	if oldQC == nil || oldQC.GetSignInfos() == nil {
		return nil
	}
	old := oldQC.GetSignInfos().QCSignInfos
	var newS []*bftPb.QuorumCertSign
	for _, s := range old {
		newS = append(newS, &bftPb.QuorumCertSign{
			Address:   s.Address,
			PublicKey: s.PublicKey,
			Sign:      s.Sign,
		})
	}
	return newS
}

// NewSignToOld 新的签名结构转化为老的签名结构
func NewSignToOld(new []*bftPb.QuorumCertSign) []*lpb.SignInfo {
	var oldS []*lpb.SignInfo
	for _, s := range new {
		oldS = append(oldS, &lpb.SignInfo{
			Address:   s.Address,
			PublicKey: s.PublicKey,
			Sign:      s.Sign,
		})
	}
	return oldS
}
