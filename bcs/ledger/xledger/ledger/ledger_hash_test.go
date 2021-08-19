package ledger

import (
	"bytes"
	"math/rand"
	"testing"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/lib/crypto/hash"
)

func TestMerkleHash(t *testing.T) {
	left := []byte("hello")
	right := []byte("world")
	result := make([]byte, 32)
	result = merkleDoubleSha256(left, right, result)

	result1 := hash.DoubleSha256(append(left, right...))
	if !bytes.Equal(result, result1) {
		t.Fatal("not equal")
	}
}

func BenchmarkNormalMerkle(b *testing.B) {
	var txs []*pb.Transaction
	for i := 0; i < 10000; i++ {
		buf := make([]byte, 32)
		rand.Read(buf)
		txs = append(txs, &pb.Transaction{
			Txid: buf,
		})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MakeMerkleTree(txs)
	}
}
