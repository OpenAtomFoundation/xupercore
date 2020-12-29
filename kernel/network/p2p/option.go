package p2p

type Option struct {
	Filters   []FilterStrategy
	Addresses []string
	PeerIDs   []string
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
func WithPercent(percent float32) OptionFunc {
	return func(o *Option) {
		o.Percent = percent
	}
}

// WithAddresses add target peer addresses to message Option
func WithAddresses(addresses []string) OptionFunc {
	return func(o *Option) {
		o.Addresses = addresses
	}
}

// WithPeerIDs add target peer IDs to message Option
func WithPeerIDs(peerIDs []string) OptionFunc {
	return func(o *Option) {
		o.PeerIDs = peerIDs
	}
}

func WithAccounts(accounts []string) OptionFunc {
	return func(o *Option) {
		o.Accounts = accounts
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
