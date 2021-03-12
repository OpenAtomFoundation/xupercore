package ledger

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"

	//"io/ioutil"
	"math/big"
	//"os"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/mock"
	"github.com/xuperchain/xupercore/lib/logs"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
	"github.com/xuperchain/xupercore/protos"
)

const AliceAddress = "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"
const BobAddress = "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"

func openLedger() (*Ledger, error) {
	workspace, dirErr := ioutil.TempDir("/tmp", "")
	if dirErr != nil {
		return nil, dirErr
	}
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		return nil, err
	}
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))

	lctx, err := NewLedgerCtx(econf, "xuper")
	if err != nil {
		return nil, err
	}
	lctx.EnvCfg.ChainDir = workspace

	genesisConf := []byte(`
		{
    "version": "1",
    "predistribution": [
        {
            "address": "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
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
            "miner": "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
            "period": 3000
        }
    }
}
    `)
	ledgerIns, err := CreateLedger(lctx, genesisConf)
	if err != nil {
		return nil, err
	}
	return ledgerIns, nil
}
func TestOpenClose(t *testing.T) {
	ledger, err := openLedger()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ledger)
	ledger.Close()
}

func TestBasicFunc(t *testing.T) {
	ledger, err := openLedger()
	if err != nil {
		t.Fatal(err)
	}
	t1 := &pb.Transaction{}
	t2 := &pb.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &protos.TxOutput{Amount: []byte("888"), ToAddr: []byte(BobAddress)})
	t1.Coinbase = true
	t1.Desc = []byte(`{"maxblocksize" : "128"}`)
	t1.Txid, _ = txhash.MakeTransactionID(t1)
	t2.TxInputs = append(t2.TxInputs, &protos.TxInput{RefTxid: t1.Txid, RefOffset: 0, FromAddr: []byte(AliceAddress)})
	t2.Txid, _ = txhash.MakeTransactionID(t2)
	ecdsaPk, pkErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	t.Logf("pkSk: %v", ecdsaPk)
	if pkErr != nil {
		t.Fatal("fail to generate publice/private key")
	}
	block, err := ledger.FormatRootBlock([]*pb.Transaction{t1})
	if err != nil {
		t.Fatalf("format block fail, %v", err)
	}
	t.Logf("block id %x", block.Blockid)
	confirmStatus := ledger.ConfirmBlock(block, true)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail")
	}
	hasTx, _ := ledger.HasTransaction(t1.Txid)
	if !hasTx {
		t.Fatal("genesis tx not exist")
	}
	confirmTwice := ledger.ConfirmBlock(block, true)
	if confirmTwice.Succ {
		t.Fatal("confirm should fail, unexpected")
	}
	t1 = &pb.Transaction{}
	t2 = &pb.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &protos.TxOutput{Amount: []byte("666"), ToAddr: []byte(BobAddress)})
	t1.Txid, _ = txhash.MakeTransactionID(t1)
	t2.TxInputs = append(t2.TxInputs, &protos.TxInput{RefTxid: t1.Txid, RefOffset: 0, FromAddr: []byte(AliceAddress)})
	t2.Txid, _ = txhash.MakeTransactionID(t2)
	block2, err := ledger.FormatBlock([]*pb.Transaction{t1, t2},
		[]byte("xchain-Miner-222222"),
		ecdsaPk,
		223456789,
		0,
		0,
		block.Blockid, big.NewInt(0),
	)
	t.Logf("bolock2 id %x", block2.Blockid)
	confirmStatus = ledger.ConfirmBlock(block2, false)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail 2")
	}

	blockCopy, readErr := ledger.QueryBlock(block.Blockid)
	if readErr != nil {
		t.Fatalf("read block fail, %v", readErr)
	} else {
		t.Logf("block detail: %v", proto.MarshalTextString(blockCopy))
	}
	blockByHeight, _ := ledger.QueryBlockByHeight(block.Height)
	if string(blockByHeight.Blockid) != string(blockCopy.Blockid) {
		t.Fatalf("query block by height failed")
	}

	lastBlockHeight := ledger.meta.GetTrunkHeight()
	lastBlock, _ := ledger.QueryBlockByHeight(lastBlockHeight)
	t.Logf("query last block %x", lastBlock.Blockid)
	t.Logf("block1 next hash %x", blockCopy.NextHash)
	blockCopy2, readErr2 := ledger.QueryBlock(blockCopy.NextHash)
	if readErr2 != nil {
		t.Fatalf("read block fail, %v", readErr2)
	} else {
		t.Logf("block2 detail: %v", proto.MarshalTextString(blockCopy2))
	}
	txCopy, txErr := ledger.QueryTransaction(t1.Txid)
	if txErr != nil {
		t.Fatalf("query tx fail, %v", txErr)
	}
	t.Logf("tx detail: %v", txCopy)
	_, err = ledger.QueryBlockByTxid(t1.Txid)
	if err != nil {
		t.Fatal("query by txid err:", err)
	}
	maxBlockSize := ledger.GetMaxBlockSize()
	if maxBlockSize != (128 << 20) {
		t.Fatalf("maxBlockSize unexpected: %v", maxBlockSize)
	}

	// coinbase txs > 1
	t1 = &pb.Transaction{}
	t2 = &pb.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &protos.TxOutput{Amount: []byte("666"), ToAddr: []byte(BobAddress)})
	t1.Coinbase = true
	t1.Desc = []byte("{}")
	t1.Txid, _ = txhash.MakeTransactionID(t1)
	t2.TxInputs = append(t2.TxInputs, &protos.TxInput{RefTxid: t1.Txid, RefOffset: 0, FromAddr: []byte(AliceAddress)})
	t2.Coinbase = true
	t2.Txid, _ = txhash.MakeTransactionID(t2)
	block3, err := ledger.FormatBlock([]*pb.Transaction{t1, t2},
		[]byte("xchain-Miner-222222"),
		ecdsaPk,
		223456789,
		0,
		0,
		block.Blockid, big.NewInt(0),
	)
	t.Logf("bolock3 id %x", block3.Blockid)
	confirmStatus = ledger.ConfirmBlock(block3, false)
	if confirmStatus.Succ {
		t.Fatal("The num of coinbase txs error")
	}

	verifyRes, err := ledger.VerifyBlock(block2, "1")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("verify res:", verifyRes)

	ledger.Close()
}

func TestSplitFunc(t *testing.T) {
	ledger, err := openLedger()
	if err != nil {
		t.Fatal(err)
	}
	t1 := &pb.Transaction{}
	t2 := &pb.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &protos.TxOutput{Amount: []byte("666"), ToAddr: []byte(BobAddress)})
	t1.Coinbase = true
	t1.Desc = []byte("{}")
	t1.Txid, _ = txhash.MakeTransactionID(t1)
	t2.TxInputs = append(t2.TxInputs, &protos.TxInput{RefTxid: t1.Txid, RefOffset: 0, FromAddr: []byte(AliceAddress)})
	t2.Txid, _ = txhash.MakeTransactionID(t2)
	ecdsaPk, pkErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	t.Logf("pkSk: %v", ecdsaPk)
	if pkErr != nil {
		t.Fatal("fail to generate publice/private key")
	}
	block, err := ledger.FormatBlock([]*pb.Transaction{t1, t2},
		[]byte("xchain-Miner-1"),
		ecdsaPk,
		123456789,
		0,
		0,
		[]byte{}, big.NewInt(0),
	)
	if err != nil {
		t.Fatalf("format block fail, %v", err)
	}
	t.Logf("block id %x", block.Blockid)
	confirmStatus := ledger.ConfirmBlock(block, true)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail", confirmStatus.Error)
	}
	t1 = &pb.Transaction{}
	t2 = &pb.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &protos.TxOutput{Amount: []byte("999"), ToAddr: []byte(BobAddress)})
	t1.Txid, _ = txhash.MakeTransactionID(t1)
	t2.TxInputs = append(t2.TxInputs, &protos.TxInput{RefTxid: t1.Txid, RefOffset: 0, FromAddr: []byte(AliceAddress)})
	t2.Txid, _ = txhash.MakeTransactionID(t2)
	block2, err := ledger.FormatBlock([]*pb.Transaction{t1, t2},
		[]byte("xchain-Miner-222222"),
		ecdsaPk,
		223456789,
		0,
		0,
		block.Blockid, big.NewInt(0),
	)
	t.Logf("bolock2 id %x", block2.Blockid)
	confirmStatus = ledger.ConfirmBlock(block2, false)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail 2", confirmStatus.Error)
	}
	//伪造一个新的txid
	t1.Txid = append(t1.Txid, []byte("a")...)
	t2.Txid = append(t2.Txid, []byte("b")...)

	block3, err := ledger.FormatBlock([]*pb.Transaction{t1, t2},
		[]byte("xchain-Miner-222223"),
		ecdsaPk,
		2234567899,
		0,
		0,
		block.Blockid, big.NewInt(0),
	)
	t.Logf("bolock3 id %x", block3.Blockid)
	confirmStatus = ledger.ConfirmBlock(block3, false)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail 3")
	}

	block4, err := ledger.FormatBlock([]*pb.Transaction{t1, t2},
		[]byte("xchain-Miner-222224"),
		ecdsaPk,
		2234567999,
		0,
		0,
		block3.Blockid, big.NewInt(0),
	)
	t.Logf("bolock4 id %x", block4.Blockid)
	ibErr := ledger.SavePendingBlock(block4)
	if ibErr != nil {
		t.Fatal("save pending block fail", ibErr)
	}
	ibBlock, ibLookErr := ledger.GetPendingBlock(block4.Blockid)
	if ibBlock == nil || ibLookErr != nil {
		t.Fatal("fail to get pending block", ibLookErr)
	} else {
		t.Log("pending block got", ibBlock)
	}
	confirmStatus = ledger.ConfirmBlock(block4, false)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail 4")
	}
	_, ibLookErr = ledger.GetPendingBlock(block4.Blockid)
	if ibLookErr != nil && ibLookErr != ErrBlockNotExist {
		t.Fatal("pending block is expected to be deleted", ibLookErr)
	}
	dumpLayer, dumpErr := ledger.Dump()
	if dumpErr != nil {
		t.Fatal("dump ledger fail")
	} else {
		for height, blocks := range dumpLayer {
			t.Log("Height", height, "blocks", blocks)
		}
	}

	gensisBlock := ledger.GetGenesisBlock()
	if gensisBlock != nil {
		t.Log("gensisBlock ", gensisBlock)
	} else {
		t.Fatal("gensis_block is expected to be not nil")
	}

	slideWindow := ledger.GetIrreversibleSlideWindow()
	maxBlockSize := ledger.GetMaxBlockSize()
	gasPrice := ledger.GetGasPrice()
	noFee := ledger.GetNoFee()
	newAccountAmount := ledger.GetNewAccountResourceAmount()
	t.Log("slideWindow:", slideWindow, " maxBlockSize:", maxBlockSize, " gasPrice", gasPrice,
		" noFee", noFee, " newAccountAmount", newAccountAmount)
	reservedContracts, err := ledger.GetReservedContracts()
	if reservedContracts != nil {
		t.Log("reservedContract ", reservedContracts)
	}
	if err != nil {
		t.Fatal(err)
	}
	fobidContracts, err := ledger.GetForbiddenContract()
	if fobidContracts != nil {
		t.Log("fobidContracts ", fobidContracts)
	}
	if err != nil {
		t.Fatal(err)
	}
	groupChainContracts, err := ledger.GetGroupChainContract()
	if groupChainContracts != nil {
		t.Log("groupChainContracts ", groupChainContracts)
	}
	if err != nil {
		t.Fatal(err)
	}
	curBlockid := block4.Blockid
	destBlockid := block2.Blockid

	undoBlocks, todoBlocks, findErr := ledger.FindUndoAndTodoBlocks(curBlockid, destBlockid)
	if findErr != nil {
		t.Fatal("fail to to find common parent of two blocks", "destBlockid",
			fmt.Sprintf("%x", destBlockid), "latestBlockid", fmt.Sprintf("%x", curBlockid))
	} else {
		t.Log("print undo block")
		for _, undoBlk := range undoBlocks {
			t.Log(undoBlk.Blockid)
		}
		t.Log("print todo block")
		for _, todoBlk := range todoBlocks {
			t.Log(todoBlk.Blockid)
		}
	}
	// test for IsTxInTrunk
	// t1 is in block3 and block3 is in branch
	if isOnChain := ledger.IsTxInTrunk(t1.Txid); !isOnChain {
		t.Fatal("expect true, got ", isOnChain)
	}
	// test for QueryBlockHeader
	blkHeader, err := ledger.QueryBlockHeader(block4.Blockid)
	if err != nil {
		t.Fatal("Query Block error")
	} else {
		t.Log("blkHeader ", blkHeader)
	}
	// test for ExistBlock
	if exist := ledger.ExistBlock(block3.Blockid); !exist {
		t.Fatal("expect block3 exist, got ", exist)
	}

	ledger.Close()
}

func TestTruncate(t *testing.T) {
	ledger, err := openLedger()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ledger.meta)

	t1 := &pb.Transaction{}
	t2 := &pb.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &protos.TxOutput{Amount: []byte("888"), ToAddr: []byte(BobAddress)})
	t1.Coinbase = true
	t1.Desc = []byte(`{"maxblocksize" : "128"}`)
	t1.Txid, _ = txhash.MakeTransactionID(t1)
	t2.TxInputs = append(t2.TxInputs, &protos.TxInput{RefTxid: t1.Txid, RefOffset: 0, FromAddr: []byte(AliceAddress)})
	t2.Txid, _ = txhash.MakeTransactionID(t2)
	ecdsaPk, pkErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	t.Logf("pkSk: %v", ecdsaPk)
	if pkErr != nil {
		t.Fatal("fail to generate publice/private key")
	}
	block1, err := ledger.FormatRootBlock([]*pb.Transaction{t1})
	if err != nil {
		t.Fatalf("format block fail, %v", err)
	}
	t.Logf("block1 id %x", block1.Blockid)
	confirmStatus := ledger.ConfirmBlock(block1, true)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail")
	}

	t1 = &pb.Transaction{}
	t2 = &pb.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &protos.TxOutput{Amount: []byte("666"), ToAddr: []byte(BobAddress)})
	t1.Txid, _ = txhash.MakeTransactionID(t1)
	t2.TxInputs = append(t2.TxInputs, &protos.TxInput{RefTxid: t1.Txid, RefOffset: 0, FromAddr: []byte(AliceAddress)})
	t2.Txid, _ = txhash.MakeTransactionID(t2)
	//block2
	block2, err := ledger.FormatBlock([]*pb.Transaction{t1, t2},
		[]byte("xchain-Miner-222222"),
		ecdsaPk,
		223456789,
		0,
		0,
		block1.Blockid, big.NewInt(0),
	)
	t.Logf("bolock2 id %x", block2.Blockid)
	confirmStatus = ledger.ConfirmBlock(block2, false)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail 2")
	}

	//block2 <- block3
	block3, err := ledger.FormatBlock([]*pb.Transaction{&pb.Transaction{Txid: []byte("dummy1")}},
		[]byte("xchain-Miner-333333"),
		ecdsaPk,
		223456790,
		0,
		0,
		block2.Blockid, big.NewInt(0),
	)
	confirmStatus = ledger.ConfirmBlock(block3, false)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail 2")
	}

	//block2 <- block4
	block4, err := ledger.FormatBlock([]*pb.Transaction{&pb.Transaction{Txid: []byte("dummy2")}},
		[]byte("xchain-Miner-444444"),
		ecdsaPk,
		223456791,
		0,
		0,
		block2.Blockid, big.NewInt(0),
	)
	confirmStatus = ledger.ConfirmBlock(block4, false)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail 2")
	}

	layers, _ := ledger.Dump()
	t.Log("Before truncate", layers)
	if len(layers) != 3 {
		t.Fatal("layers unexpected", len(layers))
	}
	err = ledger.Truncate(block1.Blockid)
	if err != nil {
		t.Fatalf("Trucate error")
	}
	layers, _ = ledger.Dump()
	if len(layers) != 1 {
		t.Fatal("layers unexpected", len(layers))
	}
	t.Log("After truncate", layers)

	metaBuf, _ := ledger.metaTable.Get([]byte(""))
	_ = proto.Unmarshal(metaBuf, ledger.meta)
	t.Log(ledger.meta)

	ledger.Close()
}
