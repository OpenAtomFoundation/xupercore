package utils

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	bftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	bftStorage "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/storage"
	"github.com/xuperchain/xupercore/kernel/contract"
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
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func CleanProduceMap(isProduce map[int64]bool, period int64) {
	// 删除已经落盘的所有key
	if len(isProduce) <= MaxMapSize {
		return
	}
	t := time.Now().UnixNano() / int64(time.Millisecond)
	key := t / period
	for k := range isProduce {
		if k <= key-int64(MaxMapSize) {
			delete(isProduce, k)
		}
	}
}

// /////////////////// lpb兼容逻辑 /////////////////////
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
func OldQCToNew(storage []byte) (bftStorage.QuorumCertInterface, error) {
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
	newQC := bftStorage.NewQuorumCert(
		&bftStorage.VoteInfo{
			ProposalId:   oldQC.ProposalId,
			ProposalView: oldQC.ViewNumber,
			ParentId:     justifyQC.ProposalId,
			ParentView:   justifyQC.ViewNumber,
		}, nil, OldSignToNew(storage))
	return newQC, nil
}

// NewToOldQC 为新的QC pb结构转化为老pb结构
func NewToOldQC(new *bftStorage.QuorumCert) (*lpb.QuorumCert, error) {
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
