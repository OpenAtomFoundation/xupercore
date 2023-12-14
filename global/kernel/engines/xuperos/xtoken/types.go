package xtoken

import "math/big"

const (
	ProposalVoting = iota + 1 // 从1开始。
	ProposalSuccess
	ProposalFailure
	ProposalInvalid // 参与提案的总票数不足时，提案为此状态。
	ProposalStopped // 提案投票结束
)

const (
	voteAgreeOption = iota + 1
	voteOpposeOption
	voteWaiveOption
)

const (
	NewToken        = "NewToken"
	TotalSupply     = "TotalSupply"
	BalanceOf       = "BalanceOf"
	Transfer        = "Transfer"
	TransferFrom    = "TransferFrom"
	Approve         = "Approve"
	Allowance       = "Allowance"
	AddSupply       = "AddSupply"
	Burn            = "Burn"
	QueryToken      = "QueryToken"
	QueryTokenOwner = "QueryTokenOwner"

	Propose            = "Propose"
	Vote               = "Vote"
	CheckVote          = "CheckVote"
	StopVote           = "StopVote"
	QueryProposal      = "QueryProposal"
	QueryProposalVotes = "QueryProposalVotes"
	QueryTopic         = "QueryTopic"

	AddAdmins   = "AddAdmins"
	DelAdmins   = "DelAdmins"
	QueryAdmins = "QueryAdmins"
	SetFee      = "SetFee"
	GetFee      = "GetFee"

	//Success 成功
	Success = 200
)

const (
	XTokenContract = "XToken"
)

type XToken struct {
	Name                  string              `json:"name"`
	TotalSupply           *big.Int            `json:"totalSupply"`
	AddSupplyEnabled      bool                `json:"addSupplyEnabled"`
	BurnEnabled           bool                `json:"burnEnabled"`
	Decimal               uint64              `json:"decimal"`
	InitialAllocation     map[string]*big.Int `json:"initialAllocation"`
	ConvertibleProportion *big.Int            `json:"convertibleProportion"`
	GenesisProposal       *GenesisProposal    `json:"genesisProposal"`
}

type GenesisProposal struct {
	// 发起提案者账户余额限制。
	ProposeMinToken *big.Int `json:"proposeMinToken"`
	// 提案有效的总投票金额，有效不意味着通过，总投票金额小于此数据时，提案不生效。
	ProposalEffectiveAmount *big.Int    `json:"proposalEffectiveAmount"`
	InitialData             []*Proposal `json:"initialData"`
}

type Proposal struct {
	Topic    string   `json:"topic"` // 同一个 token 下唯一
	ID       *big.Int `json:"id"`    // 同一个 topic 下唯一
	Proposer string   `json:"proposer"`
	Data     string   `json:"data"`
	Status   int      `json:"status"`
}

type Balance struct {
	BalanceTotal *big.Int `json:"balanceTotal"`
	Frozen       *big.Int `json:"frozen"`
}

type CheckVoteResult struct {
	Status      int      `json:"status,omitempty"`
	AgreeCount  *big.Int `json:"agreeCount"`
	OpposeCount *big.Int `json:"opposeCount"`
	WaiveCount  *big.Int `json:"waiveCount"`
}

func KeyOfToken(tokenName string) string {
	return "TOKEN_" + tokenName
}

func KeyOfTokenOwner(tokenName string) string {
	return "TOKEN_" + tokenName + "_owner"
}

func KeyOfAddress2TokenBalance(tokenName, address string) string {
	return "TOKEN_" + address + "_" + tokenName
}

func KeyOfToken2AddressBalance(tokenName, address string) string {
	return "TOKEN_" + tokenName + "_" + address
}

func KeyOfAllowances(tokenName, address string) string {
	return "TOKEN_" + tokenName + "_allowances_" + address
}

func KeyOfTokenProposal(tokenName string) string {
	return "TOKEN_" + tokenName + "_proposal"
}

// 提案投票

func KeyOfProposalID(tokenName, topic string, id *big.Int) string {
	return "G_proposal_" + tokenName + "_" + topic + "_" + id.String()
}

func KeyOfLatestProposalID(tokenName, topic string) string {
	return "G_proposal_" + tokenName + "_" + topic + "_latest"
}

func KeyOfTopicData(tokenName, topic string) string {
	return "G_proposal_" + tokenName + "_" + topic + "_data"
}

// 赞成票前缀
func KeyOfID2AddressAgreeVotePrefix(tokenName, topic string, id *big.Int) string {
	return "G_vote_" + tokenName + "_" + topic + "_" + id.String() + "_agree_"
}

// 反对票前缀
func KeyOfID2AddressOpposeVotePrefix(tokenName, topic string, id *big.Int) string {
	return "G_vote_" + tokenName + "_" + topic + "_" + id.String() + "_oppose_"
}

// 弃权票前缀
func KeyOfID2AddressWaiveVotePrefix(tokenName, topic string, id *big.Int) string {
	return "G_vote_" + tokenName + "_" + topic + "_" + id.String() + "_waive_"
}

func KeyOfID2AddressAgreeVote(tokenName, topic, address string, id *big.Int) string {
	return KeyOfID2AddressAgreeVotePrefix(tokenName, topic, id) + address
}

func KeyOfID2AddressOpposeVote(tokenName, topic, address string, id *big.Int) string {
	return KeyOfID2AddressOpposeVotePrefix(tokenName, topic, id) + address
}

func KeyOfID2AddressWaiveVote(tokenName, topic, address string, id *big.Int) string {
	return KeyOfID2AddressWaiveVotePrefix(tokenName, topic, id) + address
}

// 用户发起的还未结束的提案，存储数据为提案ID以及用户锁定的余额，同一时间只会有同一个提案。
func KeyOfProposer2Proposal(tokenName, address string) string {
	return "G_proposal_" + tokenName + "_" + address + "_proposal"
}

func KeyOfAddressVotingProposal(tokenName, address string) string {
	return "G_voting_" + tokenName + "_" + address
}

// 权限管理

func KeyOfAdmins() string {
	return "admins"
}

func KeyOfFee(method string) string {
	return "Fee_" + method
}

// 此数据为异步计票结果得key不是交易生成的，因此为非交易读写集生成的。
func KeyOfProposalResult(tokenName, topic string, id *big.Int) string {
	return XTokenContract + "/cache_proposal_" + tokenName + "_" + topic + "_" + id.String()
}
