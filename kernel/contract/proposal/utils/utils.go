package utils

const (
	StatusOK        = 200
	StatusException = 500
)

const (
	governTokenBucket = "governToken"
	balanceKey        = "balanceOf"
	totalSupplyKey    = "totalSupply"
	distributedKey    = "distributed"

	timerBucket = "timer"
	taskIDKey   = "id"

	separator = "_"

	proposalBucket  = "proposal"
	proposalIDKey   = "id"
	proposalLockKey = "lock"
)

// GetGovernTokenBucket return the govern token bucket name
func GetGovernTokenBucket() string {
	return governTokenBucket
}

// MakeTotalSupplyKey generate totalsupply key
func MakeTotalSupplyKey() string {
	return totalSupplyKey
}

// GetDistributedKey get contract distributed key
func GetDistributedKey() string {
	return distributedKey
}

// MakeContractMethodKey generate contract and account mapping key
func MakeAccountBalanceKey(account string) string {
	return balanceKey + separator + account
}

// GetTimerBucket return the balance bucket name
func GetTimerBucket() string {
	return timerBucket
}

// GetTaskIDKey return the task_id key name
func GetTaskIDKey() []byte {
	return []byte(taskIDKey)
}

// MakeTimerBlockHeightKey generate timer and blockHeight mapping key
func MakeTimerBlockHeightTaskKey(blockHeight string, taskID string) string {
	return blockHeight + separator + taskID
}

// MakeTimerBlockHeightPrefix generate timer and blockHeight prefix
func MakeTimerBlockHeightPrefix(blockHeight string) string {
	return blockHeight + separator
}

// MakeTimerBlockHeightPrefix generate timer and blockHeight prefix
func MakeTimerBlockHeightPrefixSeparator(blockHeight string) string {
	return blockHeight + separator + separator
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
	return proposalID
}

// MakeProposalLockKey generate proposal lock key
func MakeProposalLockKey(proposalID string, account string) string {
	return proposalLockKey + separator + proposalID + separator + account
}

// MakeTimerBlockHeightPrefix generate timer and blockHeight prefix
func MakeProposalLockPrefix(proposalID string) string {
	return proposalLockKey + separator + proposalID + separator
}

// MakeTimerBlockHeightPrefix generate timer and blockHeight prefix
func MakeProposalLockPrefixSeparator(proposalID string) string {
	return proposalLockKey + separator + proposalID + separator + separator
}

// PrefixRange returns key range that satisfy the given prefix
func PrefixRange(prefix []byte) []byte {
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
	return limit
}
