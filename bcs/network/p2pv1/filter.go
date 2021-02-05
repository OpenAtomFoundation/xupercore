package p2pv1

import "github.com/xuperchain/xupercore/kernel/network/def"

// PeerFilter the interface for filter peers
type PeerFilter interface {
	Filter() ([]string, error)
}

// StaticNodeStrategy a peer filter that contains strategy nodes
type StaticNodeStrategy struct {
	srv *P2PServerV1

	broadcast bool
	bcname    string
}

// Filter return static nodes peers
func (ss *StaticNodeStrategy) Filter() ([]string, error) {
	var peers []string
	if ss.broadcast {
		peers = append(peers, ss.srv.staticNodes[def.BlockChain]...)
	} else {
		peers = append(peers, ss.srv.staticNodes[ss.bcname]...)
	}
	if len(ss.srv.bootNodes) != 0 {
		peers = append(peers, ss.srv.bootNodes...)
	}
	if len(ss.srv.dynamicNodes) != 0 {
		peers = append(peers, ss.srv.dynamicNodes...)
	}
	return peers, nil
}

// MultiStrategy a peer filter that contains multiple filters
type MultiStrategy struct {
	filters []PeerFilter
	peerIDs []string
}

// NewMultiStrategy create instance of MultiStrategy
func NewMultiStrategy(filters []PeerFilter, peerIDs []string) *MultiStrategy {
	return &MultiStrategy{
		filters: filters,
		peerIDs: peerIDs,
	}
}

// Filter return peer IDs with multiple filters
func (cp *MultiStrategy) Filter() ([]string, error) {
	res := make([]string, 0)
	dupCheck := make(map[string]bool)
	// add target peers
	for _, peer := range cp.peerIDs {
		if _, ok := dupCheck[peer]; !ok {
			dupCheck[peer] = true
			res = append(res, peer)
		}
	}
	if len(res) > 0 {
		return res, nil
	}

	// add all filters
	for _, filter := range cp.filters {
		peers, err := filter.Filter()
		if err != nil {
			return res, err
		}
		for _, peer := range peers {
			if _, ok := dupCheck[peer]; !ok {
				dupCheck[peer] = true
				res = append(res, peer)
			}
		}
	}
	return res, nil
}
