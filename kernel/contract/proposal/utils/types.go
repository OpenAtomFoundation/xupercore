package utils

import (
	"encoding/json"
)

const (
	ProposalTypeConsensus = "consensus"
	ProposalTypeOrdinary  = "ordinary"
)

// Proposal
type Proposal struct {
	Module  string                 `json:"module"`
	Method  string                 `json:"method"`
	Args    map[string]interface{} `json:"args"`
	Trigger *TriggerDesc           `json:"trigger"`

	//VoteAmount uint64 `json:"vote_Amount"`
}

// TriggerDesc is the description to trigger a event used by proposal
type TriggerDesc struct {
	Height int64                  `json:"height"`
	Module string                 `json:"module"`
	Method string                 `json:"method"`
	Args   map[string]interface{} `json:"args"`
}

// Parse 解析智能合约json
func Parse(proposalStr string) (*Proposal, error) {
	proposal := &Proposal{}
	err := json.Unmarshal([]byte(proposalStr), proposal)
	if err != nil {
		return nil, err
	}
	return proposal, nil
}
