package state

/*import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xuperchain/xupercore/lib/logs"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"testing"
	"time"

	ledger_pkg "github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	txn "github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/common/xconfig"
	crypto_client "github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/lib/crypto/hash"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/protos"
)

// common test data
const (
	BobAddress      = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	BobPubkey       = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571}`
	BobPrivateKey   = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571,"D":29079635126530934056640915735344231956621504557963207107451663058887647996601}`
	AliceAddress    = "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"
	AlicePubkey     = `{"Curvname":"P-256","X":38583161743450819602965472047899931736724287060636876073116809140664442044200,"Y":73385020193072990307254305974695788922719491565637982722155178511113463088980}`
	AlicePrivateKey = `{"Curvname":"P-256","X":38583161743450819602965472047899931736724287060636876073116809140664442044200,"Y":73385020193072990307254305974695788922719491565637982722155178511113463088980,"D":98698032903818677365237388430412623738975596999573887926929830968230132692775}`

	minerPrivateKey = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571,"D":29079635126530934056640915735344231956621504557963207107451663058887647996601}`
	minerPublicKey  = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571}`
	minerAddress    = `dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN`

	DefaultKVEngine = "default"
)

// Users predefined user
var Users = map[string]struct {
	Address    string
	Pubkey     string
	PrivateKey string
}{
	"bob": {
		Address:    BobAddress,
		Pubkey:     BobPubkey,
		PrivateKey: BobPrivateKey,
	},
	"alice": {
		Address:    AliceAddress,
		Pubkey:     AlicePubkey,
		PrivateKey: AlicePrivateKey,
	},
}

func transfer(from string, to string, t *testing.T, stateHandle *State, ledger *ledger_pkg.Ledger, amount string, preHash []byte, desc string, frozenHeight int64) ([]byte, error) {
	t.Logf("preHash of this block: %x", preHash)

	timer := timer.NewXTimer()
	tx := &pb.Transaction{}
	tx.Nonce = "nonce"
	tx.Timestamp = time.Now().UnixNano()
	tx.Desc = []byte(desc)
	tx.Version = 1
	tx.AuthRequire = append(tx.AuthRequire, Users[from].Address)
	tx.Initiator = Users[from].Address
	tx.Coinbase = false
	totalNeed := big.NewInt(0) // 需要支付的总额
	amountBig := big.NewInt(0)
	amountBig.SetString(amount, 10) // 10进制转换大整数
	if amountBig.Cmp(big.NewInt(0)) < 0 {
		return nil, fmt.Errorf("amount less than 0")
	}
	totalNeed.Add(totalNeed, amountBig)
	txOutput := &protos.TxOutput{}
	txOutput.ToAddr = []byte(Users[to].Address)
	txOutput.Amount = amountBig.Bytes()
	txOutput.FrozenHeight = frozenHeight
	tx.TxOutputs = append(tx.TxOutputs, txOutput)
	// 一般的交易
	txInputs, _, utxoTotal, selectErr := stateHandle.SelectUtxos(Users[from].Address, totalNeed, true, false)
	if selectErr != nil {
		t.Fatal(selectErr)
	}
	tx.TxInputs = txInputs
	// 多出来的utxo需要再转给自己
	if utxoTotal.Cmp(totalNeed) > 0 {
		delta := utxoTotal.Sub(utxoTotal, totalNeed)
		txOutput := &protos.TxOutput{}
		txOutput.ToAddr = []byte(Users[from].Address) // 收款人就是汇款人自己
		txOutput.Amount = delta.Bytes()
		tx.TxOutputs = append(tx.TxOutputs, txOutput)
	}
	signTx, signErr := txhash.ProcessSignTx(stateHandle.sctx.Crypt, tx, []byte(Users[from].PrivateKey))
	if signErr != nil {
		return nil, signErr
	}
	signInfo := &protos.SignatureInfo{
		PublicKey: Users[from].Pubkey,
		Sign:      signTx,
	}
	tx.InitiatorSigns = append(tx.InitiatorSigns, signInfo)
	tx.AuthRequireSigns = tx.InitiatorSigns
	tx.Txid, _ = txhash.MakeTransactionID(tx)

	timer.Mark("GenerateTx")
	verifyOK, vErr := stateHandle.ImmediateVerifyTx(tx, false)
	t.Log("VerifyTX", timer.Print())
	if !verifyOK || vErr != nil {
		t.Log("verify tx fail, ignore in unit test here", vErr)
	}
	// do query tx before do tx
	_, _, err := stateHandle.QueryTx(tx.Txid)
	if err != nil {
		t.Log("query tx ", tx.Txid, "error ", err.Error())
	}

	errDo := stateHandle.DoTx(tx)
	timer.Mark("DoTx")
	if errDo != nil {
		t.Fatal(errDo)
		return nil, errDo
	}
	stateHandle.DoTx(tx)

	// do query tx after do tx
	_, _, err = stateHandle.QueryTx(tx.Txid)
	if err != nil {
		t.Log("query tx ", tx.Txid, "error ", err.Error())
	}

	txlist, packErr := stateHandle.GetUnconfirmedTx(true)
	timer.Mark("GetUnconfirmedTx")
	if packErr != nil {
		return nil, packErr
	}
	//奖励矿工
	awardTx, minerErr := txn.GenerateAwardTx("miner-1", "1000", []byte("award,onyeah!"))
	timer.Mark("GenerateAwardTx")
	if minerErr != nil {
		return nil, minerErr
	}

	// case: award_amount is negative
	_, negativeErr := txn.GenerateAwardTx("miner-1", "-2", []byte("negative award!"))
	if negativeErr != nil {
		t.Log("GenerateAwardTx error ", negativeErr.Error())
	}
	txlist = append(txlist, awardTx)
	ecdsaPk, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	timer.Mark("GenerateKey")
	block, _ := ledger.FormatBlock(txlist, []byte("miner-1"), ecdsaPk, 123456789, 0, 0, preHash, stateHandle.GetTotal())
	timer.Mark("FormatBlock")
	confirmStatus := ledger.ConfirmBlock(block, false)
	timer.Mark("ConfirmBlock")
	if !confirmStatus.Succ {
		t.Log("confirmStatus", confirmStatus)
		return nil, errors.New("fail to confirm block")
	}
	t.Log("performance metric", timer.Print())
	return block.Blockid, nil
}

func TestStateWorkWithLedger(t *testing.T) {
	workspace, dirErr := ioutil.TempDir("/tmp", "")
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	envConf := "../../../../kernel/common/xconfig/conf/env.yaml"
	econf, err := xconfig.LoadEnvConf(envConf)
	if err != nil {
		log.Printf("load env config failed.env_conf:%s err:%v\n", envConf, err)
		t.Fatal(err)
	}
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))

	lctx, err := ledger_pkg.NewLedgerCtx(econf, "xuper")
	if err != nil {
		t.Fatal(err)
	}
	lctx.EnvCfg.ChainDir = workspace
	ledger, err := ledger_pkg.OpenLedger(lctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ledger)
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
                        "quota" : "100"
                },
				{
                        "address" : "` + AliceAddress + `",
                        "quota" : "200"
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
	stateHandle, _ := NewState(sctx)
	_, _, err = stateHandle.QueryTx([]byte("123"))
	if err != txn.ErrTxNotFound {
		t.Fatal("unexpected err", err)
	}
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
	// test for GetLatestBlockid
	tipBlock := stateHandle.GetLatestBlockid()
	t.Log("current tip block ", tipBlock)
	t.Log("last tip block ", block.Blockid)

	bobBalance, _ := stateHandle.GetBalance(BobAddress)
	aliceBalance, _ := stateHandle.GetBalance(AliceAddress)
	if bobBalance.String() != "100" || aliceBalance.String() != "200" {
		t.Fatal("unexpected balance", bobBalance, aliceBalance)
	}
	t.Logf("bob balance: %s, alice balance: %s", bobBalance.String(), aliceBalance.String())
	rootBlockid := block.Blockid
	t.Logf("rootBlockid: %x", rootBlockid)
	//bob再给alice转5
	nextBlockid, blockErr := transfer("bob", "alice", t, stateHandle, ledger, "5", rootBlockid, "", 0)
	if blockErr != nil {
		t.Fatal(blockErr)
	} else {
		t.Logf("next block id: %x", nextBlockid)
	}
	stateHandle.Play(nextBlockid)
	bobBalance, _ = stateHandle.GetBalance(BobAddress)
	aliceBalance, _ = stateHandle.GetBalance(AliceAddress)
	t.Logf("bob balance: %s, alice balance: %s", bobBalance.String(), aliceBalance.String())
	//bob再给alice转6
	nextBlockid, blockErr = transfer("bob", "alice", t, stateHandle, ledger, "6", nextBlockid, "", 0)
	if blockErr != nil {
		t.Fatal(blockErr)
	} else {
		t.Logf("next block id: %x", nextBlockid)
	}
	stateHandle.Play(nextBlockid)
	bobBalance, _ = stateHandle.GetBalance(BobAddress)
	aliceBalance, _ = stateHandle.GetBalance(AliceAddress)
	t.Logf("bob balance: %s, alice balance: %s", bobBalance.String(), aliceBalance.String())

	//再创建一个新账本，从前面一个账本复制数据
	workspace2, dirErr := ioutil.TempDir("/tmp", "")
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	os.RemoveAll(workspace2)
	defer os.RemoveAll(workspace2)
	lctx.EnvCfg.ChainDir = workspace2
	ledger2, lErr := ledger_pkg.OpenLedger(lctx)
	if lErr != nil {
		t.Fatal(lErr)
	}
	pBlockid := ledger.GetMeta().RootBlockid
	for len(pBlockid) > 0 { //这个for完成把第一个账本的数据同步给第二个
		t.Logf("replicating... %x", pBlockid)
		pBlock, pErr := ledger.QueryBlock(pBlockid)
		if pErr != nil {
			t.Fatal(pErr)
		}
		isRoot := bytes.Equal(pBlockid, ledger.GetMeta().RootBlockid)
		cStatus := ledger2.ConfirmBlock(pBlock, isRoot)
		if !cStatus.Succ {
			t.Fatal(cStatus)
		}
		pBlockid = pBlock.NextHash
	}
	sctx.EnvCfg.ChainDir = workspace2
	stateHandle2, _ := NewState(sctx)
	stateHandle2.Play(ledger2.GetMeta().RootBlockid) //先做一下根节点
	dummyBlockid, dummyErr := transfer("bob", "alice", t, stateHandle2, ledger2, "7", ledger2.GetMeta().RootBlockid, "", 0)
	if dummyErr != nil {
		t.Fatal(dummyErr)
	}
	stateHandle2.Play(dummyBlockid)
	stateHandle2.Walk(ledger2.GetMeta().TipBlockid, false) //再游走到末端 ,预期会导致dummmy block回滚
	bobBalance, _ = stateHandle2.GetBalance(BobAddress)
	aliceBalance, _ = stateHandle2.GetBalance(AliceAddress)
	minerBalance, _ := stateHandle2.GetBalance("miner-1")
	t.Logf("bob balance: %s, alice balance: %s, miner-1: %s", bobBalance.String(), aliceBalance.String(), minerBalance.String())
	if bobBalance.String() != "89" || aliceBalance.String() != "211" {
		t.Fatal("unexpected balance", bobBalance, aliceBalance)
	}
	transfer("bob", "alice", t, stateHandle2, ledger2, "7", ledger2.GetMeta().TipBlockid, "", 0)
	transfer("bob", "alice", t, stateHandle2, ledger2, "7", ledger2.GetMeta().TipBlockid, "", 0)
	stateHandle2.Walk(ledger2.GetMeta().TipBlockid, false)
	bobBalance, _ = stateHandle2.GetBalance(BobAddress)
	aliceBalance, _ = stateHandle2.GetBalance(AliceAddress)
	minerBalance, _ = stateHandle2.GetBalance("miner-1")
	t.Logf("bob balance: %s, alice balance: %s, miner-1: %s", bobBalance.String(), aliceBalance.String(), minerBalance.String())
	if bobBalance.String() != "75" || aliceBalance.String() != "225" {
		t.Fatal("unexpected balance", bobBalance, aliceBalance)
	}
	t.Log(ledger.Dump())

	aliceBalance2, _ := stateHandle.GetBalance(AliceAddress)
	t.Log("get alice balance ", aliceBalance2)

	// test for RemoveUtxoCache
	stateHandle.utxo.RemoveUtxoCache("bob", "123")
	// test for GetTotal
	total := stateHandle.GetTotal()
	t.Log("total ", total)
	iter := stateHandle.utxo.ScanWithPrefix([]byte("UWNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT_"))
	for iter.Next() {
		t.Log("ScanWithPrefix  ", iter.Key())
	}

	ledger.Close()
}

func TestCheckCylic(t *testing.T) {
	g := txn.TxGraph{}
	g["tx3"] = []string{"tx1", "tx2"}
	g["tx2"] = []string{"tx1", "tx0"}
	g["tx1"] = []string{"tx0", "tx2"}
	output, cylic, _ := txn.TopSortDFS(g)
	if output != nil {
		t.Fatal("sort fail1")
	}
	t.Log(cylic)
	//if len(cylic) != 2 {
	if cylic == false {
		t.Fatal("sort fail2")
	}
}

func TestFrozenHeight(t *testing.T) {
	workspace, dirErr := ioutil.TempDir("/tmp", "")
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	envConf := "../../../../kernel/common/xconfig/conf/env.yaml"
	econf, err := xconfig.LoadEnvConf(envConf)
	if err != nil {
		log.Printf("load env config failed.env_conf:%s err:%v\n", envConf, err)
		t.Fatal(err)
	}
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	lctx, err := ledger_pkg.NewLedgerCtx(econf, "xuper")
	if err != nil {
		t.Fatal(err)
	}
	lctx.EnvCfg.ChainDir = workspace
	ledger, err := ledger_pkg.OpenLedger(lctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ledger)
	//创建链的时候分配, bob:100, alice:200
	tx, err := txn.GenerateRootTx([]byte(`
       {
        "version" : "1"
        , "consensus" : {
                "miner" : "0x00000000000"
        }
        , "predistribution":[
                {
                        "address" : "` + BobAddress + `",
                        "quota" : "100"
                },
				{
                        "address" : "` + AliceAddress + `",
                        "quota" : "200"
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
	stateHandle, _ := NewState(sctx)
	playErr := stateHandle.Play(block.Blockid)
	if playErr != nil {
		t.Fatal(playErr)
	}
	bobBalance, _ := stateHandle.GetBalance(BobAddress)
	aliceBalance, _ := stateHandle.GetBalance(AliceAddress)
	if bobBalance.String() != "100" || aliceBalance.String() != "200" {
		t.Fatal("unexpected balance", bobBalance, aliceBalance)
	}
	//bob 给alice转100，账本高度=2的时候才能解冻
	nextBlockid, blockErr := transfer("bob", "alice", t, stateHandle, ledger, "100", ledger.GetMeta().TipBlockid, "", 2)
	if blockErr != nil {
		t.Fatal(blockErr)
	} else {
		t.Logf("next block id: %x", nextBlockid)
	}
	// test for GetFrozenBalance
	frozenBalance, frozenBalanceErr := stateHandle.GetFrozenBalance(AliceAddress)
	if frozenBalanceErr != nil {
		t.Log("get frozen balance error ", frozenBalanceErr.Error())
	} else {
		t.Log("alice frozen balance ", frozenBalance)
	}
	//alice给bob转300, 预期失败，因为无法使用被冻住的utxo
	nextBlockid, blockErr = transfer("alice", "bob", t, stateHandle, ledger, "300", ledger.GetMeta().TipBlockid, "", 0)
	if blockErr != utxo.ErrNoEnoughUTXO {
		t.Fatal("unexpected ", blockErr)
	}
	//alice先给自己转1块钱，让块高度增加
	nextBlockid, blockErr = transfer("alice", "alice", t, stateHandle, ledger, "1", ledger.GetMeta().TipBlockid, "", 0)
	if blockErr != nil {
		t.Fatal(blockErr)
	}
	//然后alice再次尝试给bob转300,预期utxo解冻可用了
	nextBlockid, blockErr = transfer("alice", "bob", t, stateHandle, ledger, "300", ledger.GetMeta().TipBlockid, "", 0)
	if blockErr != nil {
		t.Fatal(blockErr)
	}
}

func TestGetSnapShotWithBlock(t *testing.T) {
	workspace, dirErr := ioutil.TempDir("/tmp", "")
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	envConf := "../../../../kernel/common/xconfig/conf/env.yaml"
	econf, err := xconfig.LoadEnvConf(envConf)
	if err != nil {
		log.Printf("load env config failed.env_conf:%s err:%v\n", envConf, err)
		t.Fatal(err)
	}
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	lctx, err := ledger_pkg.NewLedgerCtx(econf, "xuper")
	if err != nil {
		t.Fatal(err)
	}
	lctx.EnvCfg.ChainDir = workspace
	ledger, err := ledger_pkg.OpenLedger(lctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ledger)
	tx, err := txn.GenerateRootTx([]byte(`
       {
        "version" : "1"
        , "consensus" : {
                "miner" : "0x00000000000"
        }
        , "predistribution":[
                {
                        "address" : "` + BobAddress + `",
                        "quota" : "100"
                },
				{
                        "address" : "` + AliceAddress + `",
                        "quota" : "200"
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
	stateHandle, _ := NewState(sctx)
	playErr := stateHandle.Play(block.Blockid)
	if playErr != nil {
		t.Fatal(playErr)
	}
	_, err = stateHandle.CreateSnapshot(block.GetBlockid())
	if err != nil {
		t.Fatal("CreateSnapshot fail")
	}
}*/