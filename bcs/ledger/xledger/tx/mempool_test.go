package tx

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/mock"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/protos"
)

var sum = 100

func TestMy(t *testing.T) {
	run(nil, t)
}

// 打包50010个交易，耗时200ms左右
func BenchmarkMempoolGetBatch(b *testing.B) {
	run(b, nil)
}

func printMempool(m *Mempool) {
	fmt.Println("MEMPOOL")
	fmt.Println("MEMPOOL unconfirmed len:", len(m.unconfirmed))
	fmt.Println("MEMPOOL confirmed len:", len(m.confirmed))
	fmt.Println("MEMPOOL orphans len:", len(m.orphans))
	fmt.Println("MEMPOOL bucketKeys len:", len(m.bucketKeyNodes))
	// for k, v := range m.bucketKeyNodes {
	// 	fmt.Println(k)
	// 	fmt.Println(len(v))
	// }
	// fmt.Println("m.confirmed::::::")
	// for _, v := range m.confirmed {
	// 	fmt.Println(v)
	// }
	// fmt.Println("m.unconfirmed::::::")
	// for _, v := range m.unconfirmed {
	// 	fmt.Println(v)
	// }
	// fmt.Println("m.orphans::::::")
	// for _, v := range m.orphans {
	// 	fmt.Println(v)
	// }
}

func run(b *testing.B, t *testing.T) {
	if b != nil {
		sum = b.N
	}
	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	l, _ := logs.NewLogger("1111", "test")
	m := NewMempool(nil, l)
	setup(m)
	// printMempool(m)
	// return
	// now := time.Now()

	if b != nil {
		b.ResetTimer()
	}

	result := batchTx(m)
	printMempool(m)
	// fmt.Println("确认一笔交易")
	// e := m.ConfirmTx(result[80]) //
	// if e != nil {
	// 	panic(e)
	// }
	// fmt.Println("confirm tx:", string(result[40].Txid))

	deleteID := string(result[80].Txid) //"8001"
	fmt.Println("delete tx:", deleteID)
	m.DeleteTxAndChildren(deleteID)
	// m.ConfirmeTx(result[800])
	// if e != nil {
	// 	panic(e)
	// }
	printMempool(m)
	// _, ok := m.unconfirmed[deleteID]
	// fmt.Println("删除的交易在 未确认交易表吗？", ok)
	// fmt.Println("确认一笔交易OK")
	fmt.Println("再次打包")
	batchTx(m)
}

// var result []*pb.Transaction

// func ff(tx *pb.Transaction) bool {
// 	result = append(result, tx)
// 	return true
// }

func batchTx(m *Mempool) []*pb.Transaction {
	now := time.Now()
	var rrr []*pb.Transaction
	m.Range(func(tx *pb.Transaction) bool {
		rrr = append(rrr, tx)
		return true
	})
	end := time.Now()

	fmt.Println("耗时：", end.Sub(now))
	fmt.Println("打包交易量：", len(rrr))
	txids := make([]string, 0, len(rrr))
	for _, v := range rrr {
		txids = append(txids, string(v.Txid))
	}
	// fmt.Println(txids)
	return rrr
}

func setup(m *Mempool) {
	isTest = true
	type dbtxs struct {
		Txid string
	}

	txs := []string{
		"root0",
		"root1",
		"root2",
		"root3",
		"root4",
		"root5",
		"root6",
		"root7",
		"root8",
		"root9",
	}

	for _, t := range txs {
		id := []byte(t)
		tx0 := &pb.Transaction{
			Txid: id,
			TxInputs: []*protos.TxInput{
				{
					RefTxid: []byte("nil"),
				},
			},
			TxOutputs: []*protos.TxOutput{
				{
					Amount: []byte("1"),
				},
			},
			TxInputsExt: []*protos.TxInputExt{
				{
					RefTxid: []byte("nil"),
				},
			},
			TxOutputsExt: []*protos.TxOutputExt{
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
			},
		}
		dbTxs[string(id)] = tx0
	}

	// begin := time.Now()
	for k, t := range txs {
		fatherID := []byte(t)
		id := strconv.Itoa(k)
		tx := &pb.Transaction{
			Txid: []byte(id),
			TxInputs: []*protos.TxInput{
				{
					RefTxid: fatherID,
				},
			},
			TxOutputs: []*protos.TxOutput{
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
			},
			TxInputsExt: []*protos.TxInputExt{
				{
					RefTxid: fatherID,
				},
			},
			TxOutputsExt: []*protos.TxOutputExt{
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
			},
		}
		m.PutTx(tx)
	}

	for ii := 1; ii <= sum; ii++ {
		for i := 0; i < 10; i++ {
			if ii == 90 && i == 9 {
				continue
			}

			id := strconv.Itoa(ii*100 + i)
			tx1 := &pb.Transaction{
				Txid: []byte(id),
				TxInputs: []*protos.TxInput{
					{
						RefTxid:   []byte(strconv.Itoa(0 + (ii-1)*100)),
						RefOffset: int32(i), //int32(i)
					},
					{
						RefTxid:   []byte(strconv.Itoa(1 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(2 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(3 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(4 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(5 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(6 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(7 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(8 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(9 + (ii-1)*100)),
						RefOffset: int32(i),
					},
				},
				TxOutputs: []*protos.TxOutput{
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
				},
				TxInputsExt: []*protos.TxInputExt{
					{
						RefTxid:   []byte(strconv.Itoa(0 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(1 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(2 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(3 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(4 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(5 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(6 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(7 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(8 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(9 + (ii-1)*100)),
						RefOffset: int32(i),
					},
				},
				TxOutputsExt: []*protos.TxOutputExt{
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
				},
			}
			// aaa := time.Now()
			m.PutTx(tx1)
			// bbb := time.Now()
			// fmt.Println("插入单笔交易耗时：", bbb.Sub(aaa))
		}
	}
	// end := time.Now()
	// fmt.Println("插入交易耗时：", end.Sub(begin))
}

func TestTime(t *testing.T) {
	recvTimestamp := time.Now().UnixNano()
	fmt.Println("recvTimestamp:", recvTimestamp)
	tt := time.Unix(0, recvTimestamp)
	time.Sleep(time.Second * 5)
	fmt.Println(time.Since(tt))
	if time.Since(tt) > time.Second*2 {
		fmt.Println("aaa")
	}
}
