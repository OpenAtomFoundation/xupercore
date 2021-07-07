package utxo_test

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	ledger_pkg "github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/meta"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo"
	txn "github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/mock"
	crypto_client "github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/logs"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
	"github.com/xuperchain/xupercore/protos"
)

// common test data
const (
	BobAddress   = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	AliceAddress = "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"
)

var GenesisConf = []byte(`
		{
    "version": "1",
    "predistribution": [
        {
            "address": "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN",
            "quota": "100000000000000000000"
        }
    ],
    "maxblocksize": "16",
    "award": "1000000",
    "decimals": "8",
    "award_decay": {
        "height_gap": 31536000,
        "ratio": 1
    },
    "gas_price": {
        "cpu_rate": 1000,
        "mem_rate": 1000000,
        "disk_rate": 1,
        "xfee_rate": 1
    },
    "new_account_resource_amount": 1000,
    "genesis_consensus": {
        "name": "single",
        "config": {
            "miner": "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN",
            "period": 3000
        }
    }
}
    `)

func TestBasicFunc(t *testing.T) {
	workspace, dirErr := ioutil.TempDir("/tmp", "")
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))

	lctx, err := ledger_pkg.NewLedgerCtx(econf, "xuper")
	if err != nil {
		t.Fatal(err)
	}
	lctx.EnvCfg.ChainDir = workspace

	ledger, err := ledger_pkg.CreateLedger(lctx, GenesisConf)
	if err != nil {
		t.Fatal(err)
	}
	//创建链的时候分配财富
	tx, err := txn.GenerateRootTx([]byte(`
       {
        "version" : "1"
        , "consensus" : {
                "miner" : "0x00000000000"
        }
        , "predistribution":[
                {
                        "address" : "` + BobAddress + `",
                        "quota" : "100000000"
                },
				{
                        "address" : "` + AliceAddress + `",
                        "quota" : "200000000"
                }

        ]
        , "maxblocksize" : "128"
        , "period" : "5000"
        , "award" : "1000"
		} 
    `))
	if err != nil {
		t.Fatal(err)
	}

	block, _ := ledger.FormatRootBlock([]*pb.Transaction{tx})
	t.Logf("blockid %x", block.Blockid)
	confirmStatus := ledger.ConfirmBlock(block, true)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail")
	}

	crypt, err := crypto_client.CreateCryptoClient(crypto_client.CryptoTypeDefault)
	if err != nil {
		t.Fatal(err)
	}

	sctx, err := context.NewStateCtx(econf, "xuper", ledger, crypt)
	if err != nil {
		t.Fatal(err)
	}

	sctx.EnvCfg.ChainDir = workspace
	stateHandle, _ := state.NewState(sctx)

	// test for HasTx
	exist, _ := stateHandle.HasTx(tx.Txid)
	t.Log("Has tx ", tx.Txid, exist)
	err = stateHandle.DoTx(tx)
	if err != nil {
		t.Log("coinbase do tx error ", err.Error())
	}

	playErr := stateHandle.Play(block.Blockid)
	if playErr != nil {
		t.Fatal(playErr)
	}

	metaHandle, err := meta.NewMeta(sctx, stateHandle.GetLDB())
	if err != nil {
		t.Fatal(err)
	}
	utxoHandle, err := utxo.NewUtxo(sctx, metaHandle, stateHandle.GetLDB())
	if err != nil {
		t.Fatal(err)
	}
	balance, err := utxoHandle.GetBalance(BobAddress)
	utxoHandle.AddBalance([]byte(BobAddress), big.NewInt(10000000))
	balance, err = utxoHandle.GetBalance(BobAddress)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("get balance", balance.String())

	tx1 := &pb.Transaction{}
	tx1.Nonce = "nonce"
	tx1.Timestamp = time.Now().UnixNano()
	tx1.Desc = []byte("desc")
	tx1.Version = 1
	tx1.AuthRequire = append(tx.AuthRequire, BobAddress)
	tx1.Initiator = BobAddress
	tx1.Coinbase = false
	totalNeed := big.NewInt(0) // 需要支付的总额
	amountBig := big.NewInt(0)
	amountBig.SetString("10", 10) // 10进制转换大整数
	totalNeed.Add(totalNeed, amountBig)
	totalNeed.Add(totalNeed, amountBig)
	txOutput := &protos.TxOutput{}
	txOutput.ToAddr = []byte(AliceAddress)
	txOutput.Amount = amountBig.Bytes()
	txOutput.FrozenHeight = 0
	tx.TxOutputs = append(tx.TxOutputs, txOutput)
	txInputs, _, utxoTotal, err := utxoHandle.SelectUtxos(BobAddress, totalNeed, true, false)
	if err != nil {
		t.Fatal(err)
	}
	tx1.TxInputs = txInputs
	if utxoTotal.Cmp(totalNeed) > 0 {
		delta := utxoTotal.Sub(utxoTotal, totalNeed)
		txOutput := &protos.TxOutput{}
		txOutput.ToAddr = []byte(BobAddress) // 收款人就是汇款人自己
		txOutput.Amount = delta.Bytes()
		tx1.TxOutputs = append(tx1.TxOutputs, txOutput)
	}
	err = utxoHandle.CheckInputEqualOutput(tx1)
	if err != nil {
		t.Log(err)
	}
	txInputs, _, utxoTotal, err = utxoHandle.SelectUtxosBySize(BobAddress, true, false)
	if err != nil {
		t.Fatal(err)
	}
	total := utxoHandle.GetTotal()
	t.Log("total", total.String())
	utxoHandle.UpdateUtxoTotal(big.NewInt(200), stateHandle.NewBatch(), true)
	utxoHandle.UpdateUtxoTotal(big.NewInt(100), stateHandle.NewBatch(), false)
	total = utxoHandle.GetTotal()
	t.Log("total", total.String())

	txInputs, _, utxoTotal, err = utxoHandle.SelectUtxosBySize(BobAddress, false, false)
	if err != nil {
		t.Fatal(err)
	}

	accounts, err := utxoHandle.QueryAccountContainAK(BobAddress)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("accounts", accounts)
	}

	utxoHandle.SubBalance([]byte(BobAddress), big.NewInt(100))

	_, err = utxoHandle.QueryContractStatData()
	if err != nil {
		t.Fatal(err)
	}
	keys := utxo.MakeUtxoKey([]byte("U_TEST"), "1000")
	t.Log("keys", keys)

	cs, err := utxoHandle.GetAccountContracts("XC1111111111111111@xuper")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("contracts", cs)
	record, err := utxoHandle.QueryUtxoRecord("XC1111111111111111@xuper", 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("records", record)
}
