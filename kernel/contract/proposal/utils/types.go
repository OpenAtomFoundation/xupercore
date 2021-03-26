package utils

import (
	"encoding/json"
	"math/big"
)

const (
	GovernTokenTypeOrdinary = "ordinary"
	GovernTokenTypeTDPOS    = "tdpos"

	ProposalStatusVoting              = "voting"
	ProposalStatusCancelled           = "cancelled"
	ProposalStatusRejected            = "rejected"
	ProposalStatusPassed              = "passed"
	ProposalStatusCompletedAndFailure = "completed_failure"
	ProposalStatusCompletedAndSuccess = "completed_success"
)

const (
	GovernTokenKernelContract = "$govern_token"
	ProposalKernelContract    = "$proposal"
	TimerTaskKernelContract   = "$timer_task"
	TDPOSKernelContract       = "$tdpos"
)

// Govern Token Balance
// TotalBalance = AvailableBalance + LockedBalance
// 目前包括tdpos和oridinary两种场景
// 用户的可转账余额是min(AvailableBalances)
type GovernTokenBalance struct {
	TotalBalance *big.Int           `json:"total_balance"`
	Balances     map[string]Balance `json:"balances"`
}

type Balance struct {
	AvailableBalance *big.Int `json:"available_balance"`
	LockedBalance    *big.Int `json:"locked_balance"`
}

// Proposal
type Proposal struct {
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
	balance := &GovernTokenBalance{
		TotalBalance: big.NewInt(0),
		Balances:     make(map[string]Balance),
	}

	balance.Balances[GovernTokenTypeOrdinary] = Balance{big.NewInt(0), big.NewInt(0)}
	balance.Balances[GovernTokenTypeTDPOS] = Balance{big.NewInt(0), big.NewInt(0)}

	return balance
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
