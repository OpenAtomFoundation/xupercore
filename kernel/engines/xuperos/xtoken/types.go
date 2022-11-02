package xtoken

import "math/big"

const (
	ProposalVoting = iota + 1 // 从1开始。
	ProposalSuccess
	Proposalfailure
)

const (
	XTokenContract = "XToken"

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
	ProposeMinToken         *big.Int        `json:"proposeMinToken"`
	FavourRate              uint32          `json:"favourRate"` // 提案投票通过的百分比，1-100。
	InitialData             []*Proposal     `json:"initialData"`
	ExcludeCheckVoteAddress map[string]bool `json:"excludeCheckVoteAddress"` // 不参与计票的地址。TODO 先先不用此字段
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

func KeyOfID2AddressVotePrefix(tokenName, topic string, id *big.Int) string {
	return "G_vote_" + tokenName + "_" + topic + "_" + id.String() + "_"
}

func KeyOfID2AddressVote(tokenName, topic, address string, id *big.Int) string {
	return KeyOfID2AddressVotePrefix(tokenName, topic, id) + address
}

func KeyOfAddress2IDVote(tokenName, topic, address string, id *big.Int) string {
	return "G_vote_" + tokenName + "_" + topic + "_" + address + "_" + id.String()
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
