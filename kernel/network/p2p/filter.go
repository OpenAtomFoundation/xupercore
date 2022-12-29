package p2p

// FilterStrategy defines the supported filter strategies
type FilterStrategy string

// supported filter strategies
const (
	DefaultStrategy           FilterStrategy = "DefaultStrategy"
	BucketsStrategy           FilterStrategy = "BucketsStrategy"
	NearestBucketStrategy     FilterStrategy = "NearestBucketStrategy"
	BucketsWithFactorStrategy FilterStrategy = "BucketsWithFactorStrategy"
	CorePeersStrategy         FilterStrategy = "CorePeersStrategy"
)
