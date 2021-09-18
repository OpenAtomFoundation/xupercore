package tx

import (
	"errors"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
)

// Node tx wrapper.
type Node struct {
	txid string // txID string

	tx *pb.Transaction

	readonlyInputs  map[string]*Node // 当前交易对某些key为只读，将只读父交易加入此列表，rang 时使用。
	readonlyOutputs map[string]*Node

	bucketKeyToNode map[string]*Node // txid 为空时，构造 mock node，所有子节点。key 为 bucket+key。

	txInputs     []*Node
	txInputsExt  []*Node
	txOutputs    []*Node
	txOutputsExt []*Node
}

// NewNode new node.
func NewNode(txid string, tx *pb.Transaction) *Node {
	return &Node{
		txid:            txid,
		tx:              tx,
		readonlyInputs:  make(map[string]*Node),
		readonlyOutputs: make(map[string]*Node),
		bucketKeyToNode: make(map[string]*Node),
		txInputs:        make([]*Node, len(tx.GetTxInputs())),
		txInputsExt:     make([]*Node, len(tx.GetTxInputsExt())),
		txOutputs:       make([]*Node, len(tx.GetTxOutputs())),
		txOutputsExt:    make([]*Node, len(tx.GetTxOutputsExt())),
	}
}

// 已经去重。
func (n *Node) getAllChildren() []*Node {
	if n == nil {
		return nil
	}

	result := make([]*Node, 0, len(n.txOutputs)+len(n.txOutputsExt)+len(n.readonlyOutputs))
	nodesDuplicate := make(map[string]bool, cap(result))

	for _, v := range n.txOutputs {
		if v != nil && !nodesDuplicate[v.txid] {
			result = append(result, v)
			nodesDuplicate[v.txid] = true
		}
	}

	for _, v := range n.txOutputsExt {
		if v != nil && !nodesDuplicate[v.txid] {
			result = append(result, v)
			nodesDuplicate[v.txid] = true
		}
	}

	for _, v := range n.readonlyOutputs {
		if v != nil && !nodesDuplicate[v.txid] {
			result = append(result, v)
			nodesDuplicate[v.txid] = true
		}
	}
	return result
}

// 已经去重。
func (n *Node) getAllParent() []*Node {
	if n == nil {
		return nil
	}

	result := make([]*Node, 0, len(n.txInputs)+len(n.txInputsExt)+len(n.readonlyInputs))
	nodesDuplicate := make(map[string]bool, cap(result))

	for _, v := range n.txInputs {
		if v != nil && !nodesDuplicate[v.txid] {
			result = append(result, v)
			nodesDuplicate[v.txid] = true
		}
	}

	for _, v := range n.txInputsExt {
		if v != nil && !nodesDuplicate[v.txid] {
			result = append(result, v)
			nodesDuplicate[v.txid] = true
		}
	}

	for _, v := range n.readonlyInputs {
		if v != nil && !nodesDuplicate[v.txid] {
			result = append(result, v)
			nodesDuplicate[v.txid] = true
		}
	}

	return result
}

// 更新节点的父关系。
func (n *Node) updateInput(index, offset int, node *Node, retrieve bool) (*Node, error) {
	if offset < 0 || (node.tx != nil && offset >= len(node.txOutputs)) {
		return nil, errors.New("invalid tx")
	}
	if node.tx == nil {
		// mock node. 处理 txOutputs 字段。
		index := offset - len(node.txOutputs) + 1
		if index > 0 {
			node.txOutputs = append(node.txOutputs, make([]*Node, index)...)
		}
	}

	var forDeleted *Node
	on := node.txOutputs[offset]
	if on != nil {
		if !retrieve {
			return nil, errors.New("double spent in mempool")
		}
		forDeleted = on
	}
	n.txInputs[index] = node
	node.txOutputs[offset] = n

	return forDeleted, nil
}

// 此处没有检查 index 是否越界，调用前需要保证安全。
func (n *Node) isReadonlyKey(index int) bool {
	bucket := n.tx.TxInputsExt[index].GetBucket()
	key := n.tx.TxInputsExt[index].GetKey()
	for _, ext := range n.tx.GetTxOutputsExt() {
		if ext.GetBucket() == bucket && string(ext.GetKey()) == string(key) && len(ext.GetValue()) > 0 {
			return false
		}
	}
	return true
}

func (n *Node) updateInputExt(index, offset int, node *Node, retrieve bool) (*Node, error) {
	if offset < 0 || (node.tx != nil && offset >= len(node.txOutputsExt)) {
		return nil, errors.New("invalid tx")
	}

	readonly := n.isReadonlyKey(index)
	if node.tx == nil && !readonly {
		index := offset - len(node.txOutputsExt) + 1
		if index > 0 {
			node.txOutputsExt = append(node.txOutputsExt, make([]*Node, index)...)
		}
	}

	var forDeleted *Node
	if !readonly {
		on := node.txOutputsExt[offset]
		if on != nil {
			if !retrieve {
				return nil, errors.New("double spent in mempool")
			}
			forDeleted = on
		}
	}

	if readonly {
		node.readonlyOutputs[n.txid] = n
		n.readonlyInputs[node.txid] = node
	} else {
		node.txOutputsExt[offset] = n
		n.txInputsExt[index] = node
	}

	return forDeleted, nil
}

// 删除 n 和其所有父节点的关系。断绝父子关系。
func (n *Node) breakOutputs() {
	for i, fn := range n.txInputs {
		if fn == nil {
			continue
		}
		for ii, v := range fn.txOutputs {
			if v != nil && v.txid == n.txid {
				fn.txOutputs[ii] = nil
			}
		}
		n.txInputs[i] = nil
	}

	for i, fn := range n.txInputsExt {
		if fn == nil {
			continue
		}
		for i, v := range fn.txOutputsExt {
			if v != nil && v.txid == n.txid {
				fn.txOutputsExt[i] = nil
			}
		}

		bucket := n.tx.TxInputsExt[i].GetBucket()
		key := n.tx.TxInputsExt[i].GetKey()
		bk := bucket + string(key)
		if nn, ok := fn.bucketKeyToNode[bk]; ok {
			if nn.txid == n.txid {
				delete(fn.bucketKeyToNode, bk)
			}
		}

		n.txInputsExt[i] = nil
	}

	for k, fn := range n.readonlyInputs {
		if fn == nil {
			continue
		}

		delete(fn.readonlyOutputs, n.txid)
		delete(n.readonlyInputs, k)
	}
}

func (n *Node) getInputSum() int {
	sum := 0
	for _, n := range n.txInputs {
		if n != nil {
			sum++
		}
	}

	for _, n := range n.txInputsExt {
		if n != nil {
			sum++
		}
	}

	return sum + len(n.readonlyInputs)
}

func (n *Node) removeAllInputs() {
	for i, fn := range n.txInputs {
		if fn == nil {
			continue
		}
		n.txInputs[i] = nil

		for ii, cn := range fn.txOutputs {
			if fn == nil {
				continue
			}

			if cn.txid == n.txid {
				fn.txOutputs[ii] = nil
			}
		}
	}
}
