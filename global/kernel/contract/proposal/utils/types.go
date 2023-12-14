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
	XPOSKernelContract        = "$xpos"
)

// Govern Token Balance
// TotalBalance = AvailableBalance + LockedBalance
// 目前包括tdpos和oridinary两种场景
// 用户的可转账余额是min(AvailableBalances)
type GovernTokenBalance struct {
	TotalBalance  *big.Int            `json:"total_balance"`
	LockedBalance map[string]*big.Int `json:"locked_balances"`
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
	Height   int64                  `json:"height"`
	Module   string                 `json:"module"`
	Contract string                 `json:"contract"`
	Method   string                 `json:"method"`
	Args     map[string]interface{} `json:"args"`
}

func NewGovernTokenBalance() *GovernTokenBalance {
	balance := &GovernTokenBalance{
		TotalBalance:  big.NewInt(0),
		LockedBalance: make(map[string]*big.Int),
	}

	balance.LockedBalance[GovernTokenTypeOrdinary] = big.NewInt(0)
	balance.LockedBalance[GovernTokenTypeTDPOS] = big.NewInt(0)

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
