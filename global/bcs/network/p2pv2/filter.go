package p2pv2

import (
	"math/rand"

	"github.com/libp2p/go-libp2p-core/peer"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
)

// PeerFilter the interface for filter peers
type PeerFilter interface {
	Filter() ([]peer.ID, error)
}

// BucketsFilter define filter that get all peers in buckets
type BucketsFilter struct {
	srv *P2PServerV2
}

// Filter 依据Bucket分层广播
func (bf *BucketsFilter) Filter() ([]peer.ID, error) {
	rt := bf.srv.kdht.RoutingTable()
	peers := make([]peer.ID, 0, len(rt.GetPeerInfos()))
	// TODO: 验证逻辑是否一致
	for _, peerInfos := range rt.GetPeerInfos() {
		peers = append(peers, peerInfos.Id)
	}
	return peers, nil
}

// NearestBucketFilter define filter that get nearest peers from a specified peer ID
type NearestBucketFilter struct {
	srv *P2PServerV2
}

// Filter 广播给最近的Bucket
func (nf *NearestBucketFilter) Filter() ([]peer.ID, error) {
	peers := nf.srv.kdht.RoutingTable().NearestPeers(kbucket.ConvertPeerID(nf.srv.id), MaxBroadCastPeers)
	return peers, nil
}

// BucketsFilterWithFactor define filter that get a certain percentage peers in each bucket
type BucketsFilterWithFactor struct {
	srv    *P2PServerV2
	factor float64
}

// Filter 从每个Bucket中挑选占比Factor个peers进行广播
// 对于每一个Bucket,平均分成若干块,每个块抽取若干个节点
/*
 *|<---------------- Bucket ---------------->|
 *--------------------------------------------
 *|        |        |        |        |      |
 *--------------------------------------------
 *       split1   split2    split3   split4 split5
 */
func (nf *BucketsFilterWithFactor) Filter() ([]peer.ID, error) {
	factor := nf.factor
	if factor <= 0 || factor > 1 {
		factor = DefaultBucketsFilterFactor
	}

	rt := nf.srv.kdht.RoutingTable()
	if rt.Size() <= 0 {
		return nil, nil
	}

	totalPeers := rt.ListPeers()
	totalPeersSize := len(totalPeers)
	if totalPeersSize <= 0 {
		return nil, nil
	}

	var filterPeers []peer.ID
	step := int(1.0 / factor)
	splitSize := int(float64(totalPeersSize) / (1.0 / factor))
	pos := 0

	// 处理split1, split2, split3, split4
	for pos = 0; pos < splitSize; pos++ {
		lastPos := pos * step
		// for each split
		for b := lastPos; b < lastPos+step && b < totalPeersSize; b += step {
			randPos := rand.Intn(step) + lastPos
			filterPeers = append(filterPeers, totalPeers[randPos])
		}
	}

	// 处理split5, 挑选一半出来
	for a := pos * step; a < totalPeersSize; a += 2 {
		filterPeers = append(filterPeers, totalPeers[a])
	}

	return filterPeers, nil
}

// StaticNodeStrategy a peer filter that contains strategy nodes
type StaticNodeStrategy struct {
	srv    *P2PServerV2
	bcname string
}

// Filter return static nodes peers
func (ss *StaticNodeStrategy) Filter() ([]peer.ID, error) {
	return ss.srv.staticNodes[ss.bcname], nil
}

// MultiStrategy a peer filter that contains multiple filters
type MultiStrategy struct {
	filters []PeerFilter
	peerIDs []peer.ID
}

// NewMultiStrategy create instance of MultiStrategy
func NewMultiStrategy(filters []PeerFilter, peerIDs []peer.ID) *MultiStrategy {
	return &MultiStrategy{
		filters: filters,
		peerIDs: peerIDs,
	}
}

// Filter return peer IDs with multiple filters
func (cp *MultiStrategy) Filter() ([]peer.ID, error) {
	peerIDs := make([]peer.ID, 0)
	dupCheck := make(map[string]bool)

	// add all filters
	for _, filter := range cp.filters {
		peers, err := filter.Filter()
		if err != nil {
			return peerIDs, err
		}
		for _, peerID := range peers {
			if _, ok := dupCheck[peerID.Pretty()]; !ok {
				dupCheck[peerID.Pretty()] = true
				peerIDs = append(peerIDs, peerID)
			}
		}
	}

	// add extra peers
	for _, peerID := range cp.peerIDs {
		if _, ok := dupCheck[peerID.Pretty()]; !ok {
			dupCheck[peerID.Pretty()] = true
			peerIDs = append(peerIDs, peerID)
		}
	}

	return peerIDs, nil
}
