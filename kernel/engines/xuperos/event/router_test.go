package event

import (
	"encoding/hex"
	"github.com/xuperchain/xupercore/protos"
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestRouteBlockTopic(t *testing.T) {
	ledger := newMockBlockStore()
	block := newBlockBuilder().Block()
	ledger.AppendBlock(block)

	router := NewRounterFromChainMG(ledger)

	filter := &protos.BlockFilter{
		Range: &protos.BlockRange{
			Start: "0",
		},
	}
	buf, err := proto.Marshal(filter)
	if err != nil {
		t.Fatal(err)
	}
	encfunc, iter, err := router.Subscribe(protos.SubscribeType_BLOCK, buf)
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	iter.Next()
	fblock := iter.Data().(*protos.FilteredBlock)

	_, err = encfunc(fblock)
	if err != nil {
		t.Fatal(err)
	}

	if fblock.GetBlockid() != hex.EncodeToString(block.GetBlockid()) {
		t.Fatalf("block not equal, expect %x got %s", block.GetBlockid(), fblock.GetBlockid())
	}
}

func TestRouteBlockTopicRaw(t *testing.T) {
	ledger := newMockBlockStore()
	block := newBlockBuilder().Block()
	ledger.AppendBlock(block)

	router := NewRounterFromChainMG(ledger)

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
	fblock := iter.Data().(*protos.FilteredBlock)

	if fblock.GetBlockid() != hex.EncodeToString(block.GetBlockid()) {
		t.Fatalf("block not equal, expect %x got %s", block.GetBlockid(), fblock.GetBlockid())
	}
}
