package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/lib/crypto/hash"
	"github.com/xuperchain/xupercore/lib/utils"
)

func getLeafSize(txCount int) int {
	if txCount&(txCount-1) == 0 { // 刚好是2的次幂
		return txCount
	}
	exponent := uint(math.Log2(float64(txCount))) + 1
	return 1 << exponent // 2^exponent
}

func merkleDoubleSha256(left, right, result []byte) []byte {
	sum1 := sha256.New()
	sum1.Write(left)
	sum1.Write(right)
	result = sum1.Sum(result[:0])

	sum2 := sha256.New()
	sum2.Write(result)
	result = sum2.Sum(result[:0])
	return result
}

func arenaAllocator(elemSize, elemNumer int) func() []byte {
	idx := 0
	arena := make([]byte, elemNumer*elemSize)
	return func() []byte {
		if idx >= elemNumer {
			return nil
		}
		mem := arena[idx*elemSize : (idx+1)*elemSize]
		idx++
		return mem
	}
}

// MakeMerkleTree generate merkele-tree
func MakeMerkleTree(txList []*pb.Transaction) [][]byte {
	txCount := len(txList)
	if txCount == 0 {
		return nil
	}
	leafSize := getLeafSize(txCount) //需要补充为完全树
	treeSize := leafSize*2 - 1       //整个树的节点个数

	alloc := arenaAllocator(256/8, treeSize)

	tree := make([][]byte, treeSize)
	for i, tx := range txList {
		tree[i] = alloc()
		copy(tree[i], tx.Txid)
		// tree[i] = tx.Txid //用现有的txid填充部分叶子节点
	}
	noneLeafOffset := leafSize //非叶子节点的插入点
	for i := 0; i < treeSize-1; i += 2 {
		switch {
		case tree[i] == nil: //没有左孩子
			tree[noneLeafOffset] = nil
		case tree[i+1] == nil: //没有右孩子
			// concat := bytes.Join([][]byte{tree[i], tree[i]}, []byte{})
			// tree[noneLeafOffset] = hash.DoubleSha256(concat)
			tree[noneLeafOffset] = merkleDoubleSha256(tree[i], tree[i], alloc())
		default: //左右都有
			// concat := bytes.Join([][]byte{tree[i], tree[i+1]}, []byte{})
			// tree[noneLeafOffset] = hash.DoubleSha256(concat)
			tree[noneLeafOffset] = merkleDoubleSha256(tree[i], tree[i+1], alloc())
		}
		noneLeafOffset++
	}
	return tree
}

// // FastMakeMerkleTree generate merkele-tree
// func FastMakeMerkleTree(txList []*pb.Transaction) [][]byte {
// 	txCount := len(txList)
// 	if txCount == 0 {
// 		return nil
// 	}
// 	ch := make(chan func(), 8)
// 	for i := 0; i < 8; i++ {
// 		go func() {
// 			for f := range ch {
// 				f()
// 			}
// 		}()
// 	}

// 	leafSize := getLeafSize(txCount) //需要补充为完全树
// 	treeSize := leafSize*2 - 1       //整个树的节点个数
// 	tree := make([][]byte, treeSize)
// 	for i, tx := range txList {
// 		tree[i] = tx.Txid //用现有的txid填充部分叶子节点
// 	}
// 	head := tree
// 	for leafSize > 1 {
// 		children := head[:leafSize]
// 		parent := head[leafSize : leafSize+leafSize/2]
// 		fastMakeMerkleTree(children, parent, ch)
// 		head = head[leafSize:]
// 		leafSize /= 2
// 	}
// 	close(ch)
// 	return tree
// }

// func fastMakeMerkleTree(children [][]byte, parent [][]byte, ch chan func()) {
// 	wg := sync.WaitGroup{}
// 	j := 0
// 	for i := 0; i < len(children); i += 2 {
// 		idx := j
// 		switch {
// 		case children[i] == nil:
// 			parent[j] = nil
// 		case children[i+1] == nil:
// 			left, right := children[i], children[i]
// 			wg.Add(1)
// 			ch <- func() {
// 				parent[idx] = merkleDoubleSha256(left, right, nil)
// 				wg.Done()
// 			}
// 		default:
// 			left, right := children[i], children[i+1]
// 			wg.Add(1)
// 			ch <- func() {
// 				parent[idx] = merkleDoubleSha256(left, right, nil)
// 				wg.Done()
// 			}
// 		}
// 		j++
// 	}
// 	wg.Wait()
// }

// 序列化系统合约失败的Txs
func encodeFailedTxs(buf *bytes.Buffer, block *pb.InternalBlock) error {
	txids := []string{}
	for txid := range block.FailedTxs {
		txids = append(txids, txid)
	}
	sort.Strings(txids) //ascii increasing order
	for _, txid := range txids {
		txErr := block.FailedTxs[txid]
		err := binary.Write(buf, binary.LittleEndian, []byte(txErr))
		if err != nil {
			return err
		}
	}
	return nil
}

func encodeJustify(buf *bytes.Buffer, block *pb.InternalBlock) error {
	if block.Justify == nil {
		// no justify field
		return nil
	}
	err := binary.Write(buf, binary.LittleEndian, block.Justify.ProposalId)
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.LittleEndian, block.Justify.ProposalMsg)
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.LittleEndian, block.Justify.Type)
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.LittleEndian, block.Justify.ViewNumber)
	if err != nil {
		return err
	}
	if block.Justify.SignInfos != nil {
		for _, sign := range block.Justify.SignInfos.QCSignInfos {
			err = binary.Write(buf, binary.LittleEndian, []byte(sign.Address))
			if err != nil {
				return err
			}
			err = binary.Write(buf, binary.LittleEndian, []byte(sign.PublicKey))
			if err != nil {
				return err
			}
			err = binary.Write(buf, binary.LittleEndian, sign.Sign)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// VerifyMerkle
func VerifyMerkle(block *pb.InternalBlock) error {
	blockid := block.Blockid
	merkleTree := MakeMerkleTree(block.Transactions)
	if len(merkleTree) > 0 {
		merkleRoot := merkleTree[len(merkleTree)-1]
		if !(bytes.Equal(merkleRoot, block.MerkleRoot)) {
			return errors.New("merkle root is wrong, block id:" + utils.F(blockid) + ",block merkle root:" + utils.F(block.MerkleRoot) + ", make merkle root:" + utils.F(merkleRoot))
		}
		return nil
	} else {
		return errors.New("can not make merkle tree , block id:" + utils.F(blockid))
	}
}

// MakeBlockID generate BlockID
func MakeBlockID(block *pb.InternalBlock) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, block.Version)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, block.Nonce)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, block.TxCount)
	if err != nil {
		return nil, err
	}
	if block.Proposer != nil {
		err = binary.Write(buf, binary.LittleEndian, block.Proposer)
		if err != nil {
			return nil, err
		}
	}
	err = binary.Write(buf, binary.LittleEndian, block.Timestamp)
	if err != nil {
		return nil, err
	}
	if block.Pubkey != nil {
		err = binary.Write(buf, binary.LittleEndian, block.Pubkey)
		if err != nil {
			return nil, err
		}
	}
	err = binary.Write(buf, binary.LittleEndian, block.PreHash)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, block.MerkleRoot)
	if err != nil {
		return nil, err
	}
	err = encodeFailedTxs(buf, block)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, block.CurTerm)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, block.CurBlockNum)
	if err != nil {
		return nil, err
	}
	if block.TargetBits > 0 {
		err = binary.Write(buf, binary.LittleEndian, block.TargetBits)
		if err != nil {
			return nil, err
		}
	}
	err = encodeJustify(buf, block)
	if err != nil {
		return nil, fmt.Errorf("encodeJustify failed, err=%v", err)
	}
	return hash.DoubleSha256(buf.Bytes()), nil
}
