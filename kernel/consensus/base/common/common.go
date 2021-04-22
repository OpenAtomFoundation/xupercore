package utils

import (
	"container/list"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	bftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/lib/logs"
)

var (
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
	EmptyJustify     = errors.New("Justify is empty.")
	InvalidJustify   = errors.New("Justify structure is invalid.")

	MaxMapSize = 1000

	StatusOK         = 200
	StatusBadRequest = 400
	StatusErr        = 500
)

func NewContractOKResponse(json []byte) *contract.Response {
	return &contract.Response{
		Status:  StatusOK,
		Message: "success",
		Body:    json,
	}
}

func NewContractErrResponse(status int, msg string) *contract.Response {
	return &contract.Response{
		Status:  status,
		Message: msg,
	}
}

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
	tip := ledger.GetTipBlock()
	// 当前为初始状态
	if tip.GetHeight() <= startHeight {
		return &chainedBft.QCPendingTree{
			Genesis:    gNode,
			Root:       gNode,
			HighQC:     gNode,
			CommitQC:   gNode,
			Log:        log,
			OrphanList: list.New(),
			OrphanMap:  make(map[string]bool),
		}
	}
	// 重启状态时将root->tipBlock-3, generic->tipBlock-2, highQC->tipBlock-1
	// 若tipBlock<=2, root->genesisBlock, highQC->tipBlock-1
	tipNode := makeTreeNode(ledger, tip.GetHeight())
	if tip.GetHeight() < 3 {
		tree := &chainedBft.QCPendingTree{
			Genesis:    gNode,
			Root:       makeTreeNode(ledger, 0),
			Log:        log,
			OrphanList: list.New(),
			OrphanMap:  make(map[string]bool),
		}
		switch tip.GetHeight() {
		case 0:
			tree.HighQC = tree.Root
			return tree
		case 1:
			tree.HighQC = tree.Root
			tree.HighQC.Sons = append(tree.HighQC.Sons, tipNode)
			return tree
		case 2:
			tree.HighQC = makeTreeNode(ledger, 1)
			tree.HighQC.Sons = append(tree.HighQC.Sons, tipNode)
			tree.Root.Sons = append(tree.Root.Sons, tree.HighQC)
		}
		return tree
	}
	tree := &chainedBft.QCPendingTree{
		Genesis:    gNode,
		Root:       makeTreeNode(ledger, tip.GetHeight()-3),
		GenericQC:  makeTreeNode(ledger, tip.GetHeight()-2),
		HighQC:     makeTreeNode(ledger, tip.GetHeight()-1),
		Log:        log,
		OrphanList: list.New(),
		OrphanMap:  make(map[string]bool),
	}
	// 手动组装Tree结构
	tree.Root.Sons = append(tree.Root.Sons, tree.GenericQC)
	tree.GenericQC.Sons = append(tree.GenericQC.Sons, tree.HighQC)
	tree.HighQC.Sons = append(tree.HighQC.Sons, tipNode)
	return tree
}

func makeTreeNode(ledger cctx.LedgerRely, height int64) *chainedBft.ProposalNode {
	b, err := ledger.QueryBlockByHeight(height)
	if err != nil {
		return nil
	}
	pre, err := ledger.QueryBlockByHeight(height - 1)
	if err != nil {
		return &chainedBft.ProposalNode{
			In: &chainedBft.QuorumCert{
				VoteInfo: &chainedBft.VoteInfo{
					ProposalId:   b.GetBlockid(),
					ProposalView: b.GetHeight(),
				},
				LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
					CommitStateId: b.GetBlockid(),
				},
			},
		}
	}
	return &chainedBft.ProposalNode{
		In: &chainedBft.QuorumCert{
			VoteInfo: &chainedBft.VoteInfo{
				ProposalId:   b.GetBlockid(),
				ProposalView: b.GetHeight(),
				ParentId:     pre.GetBlockid(),
				ParentView:   pre.GetHeight(),
			},
			LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
				CommitStateId: b.GetBlockid(),
			},
		},
	}
}

func CleanProduceMap(isProduce map[int64]bool, period int64) {
	// 删除已经落盘的所有key
	if len(isProduce) <= MaxMapSize {
		return
	}
	t := time.Now().UnixNano() / int64(time.Millisecond)
	key := t / period
	for k, _ := range isProduce {
		if k <= key-int64(MaxMapSize) {
			delete(isProduce, k)
		}
	}
}

///////////////////// lpb兼容逻辑 /////////////////////
// 历史共识存储字段
type ConsensusStorage struct {
	Justify     *lpb.QuorumCert `json:"justify,omitempty"`
	CurTerm     int64           `json:"curTerm,omitempty"`
	CurBlockNum int64           `json:"curBlockNum,omitempty"`
	// TargetBits 是一个trick实现
	// 1. 在bcs层作为一个复用字段，记录ChainedBFT发生回滚时，当前的TipHeight，此处用int32代替int64，理论上可能造成错误
	TargetBits int32 `json:"targetBits,omitempty"`
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
