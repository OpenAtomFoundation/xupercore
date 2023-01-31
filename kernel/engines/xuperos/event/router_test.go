package event

import (
	"encoding/hex"
	"testing"

	"github.com/golang/protobuf/proto" //nolint:staticcheck

	"github.com/xuperchain/xupercore/protos"
)

func TestRouteBlockTopic(t *testing.T) {
	ledger := newMockBlockStore()
	block := newBlockBuilder().Block()
	ledger.AppendBlock(block)

	router := NewRouterFromChainMgr(ledger)

	filter := &protos.BlockFilter{
		Range: &protos.BlockRange{
			Start: "0",
		},
	}
	buf, err := proto.Marshal(filter)
	if err != nil {
		t.Fatal(err)
	}
	encode, iter, err := router.Subscribe(protos.SubscribeType_BLOCK, buf)
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	iter.Next()
	filteredBlock := iter.Data().(*protos.FilteredBlock)

	_, err = encode(filteredBlock)
	if err != nil {
		t.Fatal(err)
	}

	if filteredBlock.GetBlockid() != hex.EncodeToString(block.GetBlockid()) {
		t.Fatalf("block not equal, expect %x got %s", block.GetBlockid(), filteredBlock.GetBlockid())
	}
}

func TestRouteBlockTopicRaw(t *testing.T) {
	ledger := newMockBlockStore()
	block := newBlockBuilder().Block()
	ledger.AppendBlock(block)

	router := NewRouterFromChainMgr(ledger)

	filter := &protos.BlockFilter{
		Range: &protos.BlockRange{
			Start: "0",
		},
	}

	iter, err := router.RawSubscribe(protos.SubscribeType_BLOCK, filter)
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	iter.Next()
	filteredBlock := iter.Data().(*protos.FilteredBlock)

	if filteredBlock.GetBlockid() != hex.EncodeToString(block.GetBlockid()) {
		t.Fatalf("block not equal, expect %x got %s", block.GetBlockid(), filteredBlock.GetBlockid())
	}
}
