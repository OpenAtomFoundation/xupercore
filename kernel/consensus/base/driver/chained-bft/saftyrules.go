package chained_bft

import (
	"errors"

	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
)

var (
	EmptyVoteSignErr   = errors.New("No signature in vote.")
	InvalidVoteAddr    = errors.New("Vote address is not a validator in the target validators.")
	InvalidVoteSign    = errors.New("Vote sign is invalid compared with its publicKey")
	TooLowVoteView     = errors.New("Vote received is lower than local lastVoteRound.")
	TooLowVParentView  = errors.New("Vote's parent received is lower than local preferredRound.")
	TooLowProposalView = errors.New("Proposal received is lower than local lastVoteRound.")
	EmptyParentQC      = errors.New("Parent qc is empty.")
	NoEnoughVotes      = errors.New("Parent qc doesn't have enough votes.")
)

type saftyRulesInterface interface {
	UpdatePreferredRound(qc QuorumCertInterface) bool
	VoteProposal(proposalId []byte, proposalRound int64, parentQc QuorumCertInterface) bool
	CheckVoteMsg(qc QuorumCertInterface, logid string, validators []string) error
	CalVotesThreshold(input, sum int) bool
	IsQuorumCertValidate(proposal, parent QuorumCertInterface, justifyValidators []string) error
}

type DefaultSaftyRules struct {
	// lastVoteRound 存储着本地最近一次投票的轮数
	lastVoteRound int64
	// preferredRound 存储着本地PendingTree
	// 即有[两个子孙节点的节点]
	// 若本地有相同高度的节点，则自然排序后选出preferredRound
	preferredRound int64
	Crypto         *cCrypto.CBFTCrypto
}

func (s *DefaultSaftyRules) UpdatePreferredRound(qc QuorumCertInterface) bool {
	if qc.GetParentView() > s.preferredRound {
		s.preferredRound = qc.GetParentView()
	}
	// TODO: 检查LedgerInfo是否一致
	return true
}

// VoteProposal 返回是否需要发送voteMsg给下一个Leader
// DefaultSaftyRules 并没有严格比对proposalRound和parentRound的相邻自增关系
// 但需要注意的是，在上层bcs的实现中，由于共识操纵了账本回滚。因此实际上safetyrules需要proposalRound和parentRound严格相邻的
// 因此由于账本的可回滚性，因此lastVoteRound和preferredRound比对时，仅需比对新来的数据是否小于local数据-3即可
// 此处-3代表数据已经落盘
func (s *DefaultSaftyRules) VoteProposal(proposalId []byte, proposalRound int64, parentQc QuorumCertInterface) bool {
	if proposalRound < s.lastVoteRound-3 {
		return false
	}
	if parentQc.GetProposalView() < s.preferredRound-3 {
		return false
	}
	s.increaseLastVoteRound(proposalRound)
	return true
}

// CheckVoteMsg 检查logid、voteInfoHash是否正确
func (s *DefaultSaftyRules) CheckVoteMsg(qc QuorumCertInterface, logid string, validators []string) error {
	// 检查签名, vote目前为单个签名，因此只需要验证第一个即可，验证的内容为签名信息是否在合法的validators里面
	signs := qc.GetSignsInfo()
	if len(signs) == 0 {
		return EmptyVoteSignErr
	}
	// 是否是来自有效的候选人
	if !isInSlice(signs[0].GetAddress(), validators) {
		return InvalidVoteAddr
	}
	// 签名和公钥是否匹配
	if ok, err := s.Crypto.VerifyVoteMsgSign(signs[0], qc.GetProposalId()); !ok {
		return err
	}
	// 检查voteinfo信息, proposalView小于lastVoteRound，parentView不小于preferredRound
	if qc.GetProposalView() < s.lastVoteRound-3 {
		return TooLowVoteView
	}
	if qc.GetParentView() < s.preferredRound-3 {
		return TooLowVParentView
	}
	// TODO: 检查commit消息
	return nil
}

func (s *DefaultSaftyRules) increaseLastVoteRound(round int64) {
	if round > s.lastVoteRound {
		s.lastVoteRound = round
	}
}

func (s *DefaultSaftyRules) CalVotesThreshold(input, sum int) bool {
	// 计算最大恶意节点数
	f := (sum - 1) / 3
	if f == 0 {
		return input == sum
	}
	return input >= 2*f+1
}

// IsQuorumCertValidate 判断justify，即需check的block的parentQC是否合法
// 需要注意的是，在上层bcs的实现中，由于共识操纵了账本回滚。因此实际上safetyrules需要proposalRound和parentRound严格相邻的
// 因此在此proposal和parent的QC稍微宽松检查
func (s *DefaultSaftyRules) IsQuorumCertValidate(proposal, parent QuorumCertInterface, justifyValidators []string) error {
	if proposal.GetProposalView() < s.lastVoteRound-3 {
		return TooLowProposalView
	}
	// step2: verify justify's votes
	// verify justify sign number
	if parent.GetProposalId() == nil {
		return EmptyParentQC
	}

	// 检查justify的所有vote签名
	justifySigns := parent.GetSignsInfo()
	if !s.CalVotesThreshold(len(justifySigns), len(justifyValidators)) {
		return NoEnoughVotes
	}
	for _, v := range justifySigns {
		if !isInSlice(v.GetAddress(), justifyValidators) {
			return InvalidVoteAddr
		}
		// 签名和公钥是否匹配
		if ok, _ := s.Crypto.VerifyVoteMsgSign(v, parent.GetProposalId()); !ok {
			return InvalidVoteSign
		}
	}
	return nil
}

func isInSlice(target string, s []string) bool {
	for _, v := range s {
		if target == v {
			return true
		}
	}
	return false
}
