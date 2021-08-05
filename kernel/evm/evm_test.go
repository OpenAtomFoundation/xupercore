package evm

import (
	"encoding/hex"
	_ "github.com/xuperchain/xupercore/bcs/contract/evm"
	"testing"
)

func TestEVM(t *testing.T) {
	proxy := proxy{}
	t.Run("TestVerifySignature", func(t *testing.T) {
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
		if err := proxy.verifySignature(nonce, gasPrice, gasLimit, to, value, data, net, V, S, R); err != nil {
			t.Error(err)
			return
		}
	},
	)
	//t.Run("ContractCall", func(t *testing.T) {
	// ignore verifySignature
	//proxy.verifySignatureFunc = func(nonce, gasPrice, gasLimit uint64, to, value, data []byte, net, V uint64, S, R []byte) error {
	//	return nil
	//}
	//proxy.sendTransaction()
	//}
}
