package utils

const (
	StatusOK = 200
)

const (
	balanceBucket     = "balanceOf_"
	totalSupplyBucket = "totalSupply"

	timerBucket    = "timer_"
	tasksSeparator = "\x01"

	proposalBucket = "proposal_"

	proposalVoteKey = "proposal_vote"
	proposalIDKey   = "proposal_id"
)

// GetBalanceBucket return the balance bucket name
func GetBalanceBucket() string {
	return balanceBucket
}

// GetBalanceBucket return the balance bucket name
func GetTotalSupplyBucket() string {
	return totalSupplyBucket
}

// MakeContractMethodKey generate contract and account mapping key
func MakeTotalSupplyKey() string {
	return totalSupplyBucket
}

// MakeContractMethodKey generate contract and account mapping key
func MakeAccountBalanceKey(account string) string {
	return balanceBucket + account
}

// GetTimerBucket return the balance bucket name
func GetTimerBucket() string {
	return timerBucket
}

// MakeTimerBlockHeightKey generate timer and blockHeight mapping key
func MakeTimerBlockHeightKey(blockHeight string) string {
	return timerBucket + blockHeight
}

// MakeTimerBlockHeightKey generate timer and blockHeight mapping key
func MakeTimerBlockHeightTaskKey(blockHeight string, taskID string) string {
	return timerBucket + blockHeight + taskID
}

// MakeTimerBlockHeightPrefix generate timer and blockHeight prefix
func MakeTimerBlockHeightPrefix(blockHeight string) string {
	return timerBucket + blockHeight
}

// GetProposalBucket return the proposal bucket name
func GetProposalBucket() string {
	return proposalBucket
}

// GetProposalIDKey return the proposal_id key name
func GetProposalIDKey() []byte {
	return []byte(proposalIDKey)
}

// MakeProposalKey generate proposal key
func MakeProposalKey(proposalID string) string {
	return proposalBucket + proposalID
}

// MakeProposalVoteKey generate proposal vote key
func MakeProposalVoteKey(proposalID string) string {
	return proposalBucket + proposalVoteKey + proposalID
}

// PrefixRange returns key range that satisfy the given prefix
func PrefixRange(prefix []byte) ([]byte, []byte) {
	var limit []byte
	for i := len(prefix) - 1; i >= 0; i-- {
		c := prefix[i]
		if c < 0xff {
			limit = make([]byte, i+1)
			copy(limit, prefix)
			limit[i] = c + 1
			break
		}
	}
	return prefix, limit
}
