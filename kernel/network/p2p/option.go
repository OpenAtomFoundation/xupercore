package p2p

type Option struct {
	Filters   []FilterStrategy

	// 地址路由
	Addresses []string
	// PeerID路由
	PeerIDs   []string
	// 账户路由
	Accounts  []string

	WhiteList map[string]bool

	Percent float32 // percent wait for return
}

// OptionFunc define single Option function for send message
type OptionFunc func(*Option)

// WithWhiteList add whiteList
func WithWhiteList(whiteList map[string]bool) OptionFunc {
	return func(o *Option) {
		o.WhiteList = whiteList
	}
}

// WithFilter add filter strategies to message Option
func WithFilter(filters []FilterStrategy) OptionFunc {
	return func(o *Option) {
		o.Filters = filters
	}
}

// WithPercent add percentage to message Option
func WithPercent(percentage float32) OptionFunc {
	return func(o *Option) {
		o.Percent = percentage
	}
}

// WithAccount send msg to account's node
func WithAccount(accounts []string) OptionFunc {
	return func(o *Option) {
		o.Accounts = accounts
	}
}

// WithAddresses add target peer addresses to message Option
func WithAddresses(peerAddrs []string) OptionFunc {
	return func(o *Option) {
		o.Addresses = peerAddrs
	}
}

// WithPeerIDs add target peer IDs to message Option
func WithPeerIDs(peerIDs []string) OptionFunc {
	return func(o *Option) {
		o.PeerIDs = peerIDs
	}
}

// Apply apply OptionFunc
func Apply(optFunc []OptionFunc) *Option {
	opt := &Option{
		Percent: 1,
	}

	for _, f := range optFunc {
		f(opt)
	}

	if opt.Percent > 1 {
		opt.Percent = 1
	}
	return opt
}
