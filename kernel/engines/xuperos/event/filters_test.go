package event

import (
	"testing"

	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/protos"
)

func expectTxMatch(t *testing.T, tx *lpb.Transaction, pbfilter *protos.BlockFilter) {
	filter, err := newBlockFilter(pbfilter)
	if err != nil {
		t.Fatal(err)
	}
	if !matchTx(filter, tx) {
		t.Fatal("tx not match")
	}
}

func expectTxNotMatch(t *testing.T, tx *lpb.Transaction, pbfilter *protos.BlockFilter) {
	filter, err := newBlockFilter(pbfilter)
	if err != nil {
		t.Fatal(err)
	}
	if matchTx(filter, tx) {
		t.Fatal("unexpected tx match")
	}
}

func expectEventMatch(t *testing.T, event *protos.ContractEvent, pbfilter *protos.BlockFilter) {
	filter, err := newBlockFilter(pbfilter)
	if err != nil {
		t.Fatal(err)
	}
	if !matchEvent(filter, event) {
		t.Fatal("event not match")
	}
}

func expectEventNotMatch(t *testing.T, event *protos.ContractEvent, pbfilter *protos.BlockFilter) {
	filter, err := newBlockFilter(pbfilter)
	if err != nil {
		t.Fatal(err)
	}
	if matchEvent(filter, event) {
		t.Fatal("unexpected event match")
	}
}

func TestFilterContractName(t *testing.T) {
	tx := newTxBuilder().Invoke("counter", "increase", nil).Tx()
	t.Run("empty", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{})
	})
	t.Run("match", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{
			Contract: "counter",
		})
	})
	t.Run("notMatch", func(tt *testing.T) {
		expectTxNotMatch(tt, tx, &protos.BlockFilter{
			Contract: "erc20",
		})
	})
	t.Run("subMatch", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{
			Contract: "count",
		})
	})
	t.Run("fullMatch", func(tt *testing.T) {
		expectTxNotMatch(tt, tx, &protos.BlockFilter{
			Contract: "^count$",
		})
	})
}

func TestFilterInitiator(t *testing.T) {
	tx := newTxBuilder().Initiator("addr1").Tx()
	t.Run("empty", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{})
	})
	t.Run("notMatch", func(tt *testing.T) {
		expectTxNotMatch(tt, tx, &protos.BlockFilter{
			Initiator: "addr2",
		})
	})
	t.Run("match", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{
			Initiator: "addr1",
		})
	})
}

func TestFilterAuthRequire(t *testing.T) {
	tx := newTxBuilder().AuthRequire("addr1", "addr2").Tx()
	t.Run("empty", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{})
	})
	t.Run("notMatch", func(tt *testing.T) {
		expectTxNotMatch(tt, tx, &protos.BlockFilter{
			AuthRequire: "not_exists",
		})
	})
	t.Run("match", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{
			AuthRequire: "addr1",
		})
		expectTxMatch(tt, tx, &protos.BlockFilter{
			AuthRequire: "addr2",
		})
	})
}

func TestFilterFromAddr(t *testing.T) {
	tx := newTxBuilder().Transfer("fromAddr", "toAddr", "10").Tx()
	t.Run("empty", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{})
	})
	t.Run("notMatch", func(tt *testing.T) {
		expectTxNotMatch(tt, tx, &protos.BlockFilter{
			FromAddr: "addr2",
		})
	})
	t.Run("match", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{
			FromAddr: "fromAddr",
		})
	})
}

func TestFilterToAddr(t *testing.T) {
	tx := newTxBuilder().Transfer("fromAddr", "toAddr", "10").Tx()
	t.Run("empty", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{})
	})
	t.Run("notMatch", func(tt *testing.T) {
		expectTxNotMatch(tt, tx, &protos.BlockFilter{
			ToAddr: "addr2",
		})
	})
	t.Run("match", func(tt *testing.T) {
		expectTxMatch(tt, tx, &protos.BlockFilter{
			ToAddr: "toAddr",
		})
	})
}

func TestFilterEvent(t *testing.T) {
	event := &protos.ContractEvent{
		Contract: "counter",
		Name:     "increase",
	}

	t.Run("empty", func(tt *testing.T) {
		expectEventMatch(tt, event, &protos.BlockFilter{})
	})
	t.Run("notMatch", func(tt *testing.T) {
		expectEventNotMatch(tt, event, &protos.BlockFilter{
			EventName: "get",
		})
	})
	t.Run("match", func(tt *testing.T) {
		expectEventMatch(tt, event, &protos.BlockFilter{
			EventName: "increase",
		})
	})
}
