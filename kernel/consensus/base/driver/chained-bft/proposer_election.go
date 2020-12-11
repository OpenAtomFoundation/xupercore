package chained_bft

type ProposerElectionInterface interface {
	GetLeader(round int64) string       // 获取指定round的主节点Address
	GetValidators(round int64) []string // 获取指定round的候选人节点Address
	GetIntAddress(string) string        // 获取候选人地址到网络地址的映射
}
