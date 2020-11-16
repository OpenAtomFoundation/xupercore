package p2p

// FilterStrategy defines the supported filter strategies
type FilterStrategy string

// supported filter strategies
const (
	DefaultStrategy           FilterStrategy = "DefaultStrategy"
	BucketsStrategy                          = "BucketsStrategy"
	NearestBucketStrategy                    = "NearestBucketStrategy"
	BucketsWithFactorStrategy                = "BucketsWithFactorStrategy"
	CorePeersStrategy                        = "CorePeersStrategy"
)
