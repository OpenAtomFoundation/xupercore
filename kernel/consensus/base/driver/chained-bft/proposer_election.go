package chained_bft

type ProposerElectionInterface interface {
	GetLeader(round int64) string
	GetValidatorsMsgAddr() []string
	GetValidators(round int64) []string
	GetMsgAddress(string) string
}
