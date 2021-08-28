package evm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	_ "github.com/xuperchain/xupercore/bcs/contract/evm"
	_ "github.com/xuperchain/xupercore/bcs/contract/native"
	_ "github.com/xuperchain/xupercore/bcs/contract/xvm"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/protos"

	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	_ "github.com/xuperchain/xupercore/kernel/contract/manager"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
)

func TestEVMProxy(t *testing.T) {
	var contractConfig = &contract.ContractConfig{
		EnableUpgrade: true,
		Xkernel: contract.XkernelConfig{
			Enable: true,
			Driver: "default",
		},
		Native: contract.NativeConfig{
			Enable: true,
			Driver: "native",
		},
		EVM: contract.EVMConfig{
			Enable: true,
			Driver: "evm",
		},
		LogDriver: mock.NewMockLogger(),
	}
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()
	_, err := NewEVMProxy(m)
	if err != nil {
		t.Error(err)
		return
	}

	bin, err := ioutil.ReadFile("testdata/counter.bin")
	if err != nil {
		t.Error(err)
		return
	}
	abi, err := ioutil.ReadFile("testdata/counter.abi")
	if err != nil {
		t.Error(err)
		return
	}

	args := map[string][]byte{
		"contract_abi": abi,
	}
	data, err := hex.DecodeString(string((bin)))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := th.Deploy("evm", "counter", "counter", data, args)
	if err != nil {
		t.Fatal(err)
	}
	signedTx := []byte("0xf867808082520894f97798df751deb4b6e39d4cf998ee7cd4dcb9acc880de0b6b3a76400008025a0f0d2396973296cd6a71141c974d4a851f5eae8f08a8fba2dc36a0fef9bd6440ca0171995aa750d3f9f8e4d0eac93ff67634274f3c5acf422723f49ff09a6885422")
	var txHash []byte
	t.Run("SendRawTransaction", func(t *testing.T) {
		th.SetUtxoReader(sandbox.NewUTXOReaderFromInput([]*protos.TxInput{
			{
				FromAddr: []byte("2C2D14A9A3F0D078AC8B38E3043D78CA8BC11029"),
				Amount:   big.NewInt(9999).Bytes(),
			},
		}))

		resp, err = th.Invoke("xkernel", "$evm", "SendRawTransaction", map[string][]byte{
			"signed_tx": signedTx,
		})
		if err != nil {
			t.Error(err)
			return
		}

		txHash = resp.Body
	})
	t.Run("GetTransactionReceipt", func(t *testing.T) {
		resp, err := th.Invoke("xkernel", "$evm", "GetTransactionReceipt", map[string][]byte{
			//  for xuper-sdk-go
			"tx_hash": []byte(hex.EncodeToString(txHash)),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(resp.Body, signedTx) {
			t.Error(err)
			return
		}
	})

	t.Run("BalanceOf", func(t *testing.T) {
		addressStr := "f97798df751deb4b6e39d4cf998ee7cd4dcb9acc"
		resp, err := th.Invoke("xkernel", "$evm", "BalanceOf", map[string][]byte{
			"address": []byte(addressStr),
		})
		if err != nil {
			t.Error(err)
			return
		}
		balcne1, ok := new(big.Int).SetString(string(resp.Body), 10)
		if !ok {
			t.Error(err)
			return
		}
		if balcne1.Uint64() != 1 {
			fmt.Println()
			t.Error("balance error")
		}
	})
	_ = resp
}

//  DOTO Add TxHash Unit Test for TDD
func TestVerifySignature(t *testing.T) {
	var nonce uint64 = 0
	var gasPrice uint64 = 0
	var gasLimit uint64 = 21000
	toString := "f97798df751deb4b6e39d4cf998ee7cd4dcb9acc"
	to, err := hex.DecodeString(toString)
	if err != nil {
		t.Error(err)
		return
	}
	valueStr := "0de0b6b3a7640000"
	value, err := hex.DecodeString(valueStr)
	if err != nil {
		t.Error(err)
		return
	}
	dataStr := ""
	data, err := hex.DecodeString(dataStr)
	if err != nil {
		t.Error(err)
		return
	}

	chainID := 1
	var V uint64 = 37
	net := uint64(chainID)
	RStr := "f0d2396973296cd6a71141c974d4a851f5eae8f08a8fba2dc36a0fef9bd6440c"
	R, err := hex.DecodeString(RStr)
	if err != nil {
		t.Error(err)
		return
	}
	SStr := "171995aa750d3f9f8e4d0eac93ff67634274f3c5acf422723f49ff09a6885422"
	S, err := hex.DecodeString(SStr)
	if err != nil {
		t.Error(err)
		return
	}
	p := &proxy{}
	if err := p.verifySignature(nonce, gasPrice, gasLimit, to, value, data, net, V, S, R); err != nil {
		t.Error(err)
		return
	}
}
