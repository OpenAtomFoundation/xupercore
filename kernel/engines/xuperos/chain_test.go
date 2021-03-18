package xuperos

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/common/xaddress"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

var (
	adminTxId   = []byte(``)
	adminAmount = big.NewInt(0)
)

func init() {
	adminAmount, _ = big.NewInt(0).SetString("100000000000000000000", 10)
	adminTxId, _ = hex.DecodeString(`5aa155b99f5f405c6c05238abbc3163bd22d8452181405b3508d80b2ae646e0e`)
}

func mockTransferTx(chain common.Chain) (*lpb.Transaction, error) {
	conf := chain.Context().EngCtx.EnvCfg
	addr, err := xaddress.LoadAddrInfo(conf.GenDataAbsPath(conf.KeyDir), chain.Context().Crypto)
	if err != nil {
		return nil, err
	}

	amount, ok := big.NewInt(0).SetString("10000", 10)
	if !ok {
		return nil, fmt.Errorf("amount error")
	}

	change := big.NewInt(0).Sub(adminAmount, amount)

	tx := &lpb.Transaction{
		Version:     1,
		Coinbase:    false,
		Desc:        []byte(`mock transfer`),
		Nonce:       utils.GenNonce(),
		Timestamp:   time.Now().UnixNano(),
		Initiator:   addr.Address,
		AuthRequire: []string{addr.Address},
		TxInputs: []*protos.TxInput{
			{
				RefTxid:   adminTxId,
				RefOffset: 0,
				FromAddr:  []byte(addr.Address),
				Amount:    adminAmount.Bytes(),
			},
		},
		TxOutputs: []*protos.TxOutput{
			{
				ToAddr: []byte(addr.Address),
				Amount: change.Bytes(),
			}, {
				ToAddr: []byte(`SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co`),
				Amount: amount.Bytes(),
			},
		},
	}

	// 签名
	sign, err := txhash.ProcessSignTx(chain.Context().Crypto, tx, []byte(addr.PrivateKeyStr))
	if err != nil {
		return nil, err
	}
	signs := []*protos.SignatureInfo{
		{
			PublicKey: addr.PublicKeyStr,
			Sign:      sign,
		},
	}
	tx.InitiatorSigns = signs
	tx.AuthRequireSigns = signs

	// txID
	tx.Txid, err = txhash.MakeTransactionID(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func TestChain_SubmitTx(t *testing.T) {
	engine, err := MockEngine("p2pv2/node1/conf/env.yaml")
	if err != nil {
		t.Logf("%v", err)
		return
	}
	defer engine.Exit()

	chain, err := engine.Get("xuper")
	if err != nil {
		t.Errorf("get chain error: %v", err)
		return
	}

	tx, err := mockTransferTx(chain)
	if err != nil {
		t.Errorf("mock tx error: %v", err)
		return
	}

	err = chain.SubmitTx(chain.Context(), tx)
	if err != nil {
		t.Errorf("submit tx error: %v", err)
		return
	}
}

func mockContractTx(chain common.Chain) (*lpb.Transaction, error) {
	conf := chain.Context().EngCtx.EnvCfg
	addr, err := xaddress.LoadAddrInfo(conf.GenDataAbsPath(conf.KeyDir), chain.Context().Crypto)
	if err != nil {
		return nil, err
	}

	reqs := []*protos.InvokeRequest{
		{
			ModuleName:   "xkernel",
			ContractName: "$acl",
			MethodName:   "NewAccount",
			Args: map[string][]byte{
				"account_name": []byte("1234567890123456"),
				"acl":          []byte(`{"pm": {"rule": 1,"acceptValue": 1.0},"aksWeight": {"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY": 1}}`),
			},
		},
	}
	response, err := chain.PreExec(chain.Context(), reqs, addr.Address, []string{addr.Address})
	if err != nil {
		return nil, err
	}

	amount := big.NewInt(response.GasUsed)
	change := big.NewInt(0).Sub(adminAmount, amount)

	tx := &lpb.Transaction{
		Version:     1,
		Coinbase:    false,
		Desc:        []byte(`mock contract`),
		Nonce:       utils.GenNonce(),
		Timestamp:   time.Now().UnixNano(),
		Initiator:   addr.Address,
		AuthRequire: []string{addr.Address},
		TxInputs: []*protos.TxInput{
			{
				RefTxid:   adminTxId,
				RefOffset: 0,
				FromAddr:  []byte(addr.Address),
				Amount:    adminAmount.Bytes(),
			},
		},
		TxOutputs: []*protos.TxOutput{
			{
				ToAddr: []byte(addr.Address),
				Amount: change.Bytes(),
			}, {
				ToAddr: []byte(`$`),
				Amount: amount.Bytes(),
			},
		},
		TxInputsExt:      response.Inputs,
		TxOutputsExt:     response.Outputs,
		ContractRequests: response.Requests,
	}

	// 签名
	sign, err := txhash.ProcessSignTx(chain.Context().Crypto, tx, []byte(addr.PrivateKeyStr))
	if err != nil {
		return nil, err
	}
	signs := []*protos.SignatureInfo{
		{
			PublicKey: addr.PublicKeyStr,
			Sign:      sign,
		},
	}
	tx.InitiatorSigns = signs
	tx.AuthRequireSigns = signs

	// txID
	tx.Txid, err = txhash.MakeTransactionID(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func TestChain_PreExec(t *testing.T) {
	engine, err := MockEngine("p2pv2/node1/conf/env.yaml")
	if err != nil {
		t.Logf("%v", err)
		return
	}
	defer engine.Exit()

	chain, err := engine.Get("xuper")
	if err != nil {
		t.Errorf("get chain error: %v", err)
		return
	}

	tx, err := mockContractTx(chain)
	if err != nil {
		t.Errorf("mock tx error: %v", err)
		return
	}

	err = chain.SubmitTx(chain.Context(), tx)
	if err != nil {
		t.Errorf("submit tx error: %v", err)
		return
	}
}
