package utils

import (
	"encoding/json"
	"math/big"
)

const (
	ProposalTypeOrdinary = "ordinary"
	ProposalTypeTDPOS    = "TDPOS"

	ProposalStatusVoting    = "voting"
	ProposalStatusCancelled = "cancelled"
	ProposalStatusFailure   = "failure"
	ProposalStatusSuccess   = "success"
)

const (
	GovernTokenKernelContract = "$govern_token"
	ProposalKernelContract    = "$proposal"
	TimerTaskKernelContract   = "$timer_task"
	TDPOSKernelContract       = "$tdpos"
)

// Govern Token Balance
// TotalBalance = AvailableBalanceForTDPOS + LockedBalanceForTDPOS = AvailableBalanceForProposal + LockedBalanceForProposal
// 用户的可转账余额是min(AvailableBalanceForTDPOS, AvailableBalanceForProposal)
type GovernTokenBalance struct {
	TotalBalance                *big.Int `json:"total_balance"`
	AvailableBalanceForTDPOS    *big.Int `json:"available_balance_for_tdpos"`
	LockedBalanceForTDPOS       *big.Int `json:"locked_balance_for_tdpos"`
	AvailableBalanceForProposal *big.Int `json:"available_balance_for_proposal"`
	LockedBalanceForProposal    *big.Int `json:"locked_balance_for_proposal"`
}

// Proposal
type Proposal struct {
	Module  string                 `json:"module"`
	Method  string                 `json:"method"`
	Args    map[string]interface{} `json:"args"`
	Trigger *TriggerDesc           `json:"trigger"`

	VoteAmount *big.Int `json:"vote_amount"`
	Status     string   `json:"status"`
	Proposer   string   `json:"proposer"`
}

// TriggerDesc is the description to trigger a event used by proposal
type TriggerDesc struct {
	Height int64                  `json:"height"`
	Module string                 `json:"module"`
	Method string                 `json:"method"`
	Args   map[string]interface{} `json:"args"`
}

func NewGovernTokenBalance() *GovernTokenBalance {
	return &GovernTokenBalance{
		TotalBalance:                big.NewInt(0),
		AvailableBalanceForTDPOS:    big.NewInt(0),
		LockedBalanceForTDPOS:       big.NewInt(0),
		AvailableBalanceForProposal: big.NewInt(0),
		LockedBalanceForProposal:    big.NewInt(0),
	}
}

// Parse
func Parse(proposalStr string) (*Proposal, error) {
	proposal := &Proposal{}
	err := json.Unmarshal([]byte(proposalStr), proposal)
	if err != nil {
		return nil, err
	}
	return proposal, nil
}

// UnParse
func UnParse(proposal *Proposal) ([]byte, error) {
	proposalBuf, err := json.Marshal(proposal)
	if err != nil {
		return nil, err
	}
	return proposalBuf, nil
}
