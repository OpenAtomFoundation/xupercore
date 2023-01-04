package evm

import (
	"testing"

	"github.com/hyperledger/burrow/crypto"

	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)

func TestNewStateManager(t *testing.T) {

	st := newStateManager(&bridge.Context{
		ContractName: "contractName",
		Method:       "initialize",
	})

	if err := st.UpdateAccount(nil); err != nil {
		t.Fatal(err)
	}

	if err := st.RemoveAccount(crypto.Address{}); err != nil {
		t.Fatal(err)
	}
}
