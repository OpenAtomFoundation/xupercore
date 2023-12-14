package event

import (
	"fmt"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"

	pb "github.com/OpenAtomFoundation/xupercore/global/protos"
)

// Router distribute events according to the event type and filter
type Router struct {
	topics map[pb.SubscribeType]Topic
}

// NewRouterFromChainMgr instance Router from ChainManager
func NewRouterFromChainMgr(manager ChainManager) *Router {
	blockTopic := NewBlockTopic(manager)
	return &Router{
		topics: map[pb.SubscribeType]Topic{
			pb.SubscribeType_BLOCK: blockTopic,
		},
	}
}

// NewRouter instance Router from common.Engine
func NewRouter(engine common.Engine) *Router {
	return NewRouterFromChainMgr(NewChainManager(engine))
}

// EncodeFunc encodes event payload
type EncodeFunc func(x interface{}) ([]byte, error)

// Subscribe route events from pb.SubscribeType and filter buffer
func (r *Router) Subscribe(tp pb.SubscribeType, filterBuf []byte) (EncodeFunc, Iterator, error) {
	topic, ok := r.topics[tp]
	if !ok {
		return nil, nil, fmt.Errorf("subscribe type %s unsupported", tp)
	}
	filter, err := topic.ParseFilter(filterBuf)
	if err != nil {
		return nil, nil, fmt.Errorf("parse filter error: %s", err)
	}
	iter, err := topic.NewIterator(filter)
	return topic.MarshalEvent, iter, err
}

// RawSubscribe route events from pb.SubscribeType and filter struct
func (r *Router) RawSubscribe(tp pb.SubscribeType, filter interface{}) (Iterator, error) {
	topic, ok := r.topics[tp]
	if !ok {
		return nil, fmt.Errorf("subscribe type %s unsupported", tp)
	}
	iter, err := topic.NewIterator(filter)
	return iter, err
}
