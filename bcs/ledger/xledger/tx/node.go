package tx

import (
	"errors"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
)

// Node tx wrapper.
type Node struct {
	txid string // txID string

	tx *pb.Transaction

	// 读写的 txInputsExt 和 readonlyInputs 互斥，其中的 node 不重复.
	// 如果在 txInputsExt 和 readonlyInputs 的同一个位置都存在交易，那么就是这两个交易对同一个 key 只读和读写了.
	readonlyOutputs []map[string]*Node // 只读子交易，数组的index为当前交易输出的index，此处数组的长度与txOutputsExt长度一致。
	readonlyInputs  []*Node

	bucketKeyToNode         map[string]*Node            // txid 为空时，构造 mock node，写了某个 bucket+key 的节点，bucket+key => *Node。
	bucketKeyToReadonlyNode map[string]map[string]*Node // txid 为空时，构造 mock node，只读某个 bucket+key 的节点列表，bucket+key => [txid => *Node]。

	txInputs     []*Node
	txInputsExt  []*Node
	txOutputs    []*Node
	txOutputsExt []*Node
}

// NewNode new node.
func NewNode(txid string, tx *pb.Transaction) *Node {
	return &Node{
		txid:                    txid,
		tx:                      tx,
		readonlyOutputs:         make([]map[string]*Node, len(tx.GetTxOutputsExt())),
		readonlyInputs:          make([]*Node, len(tx.GetTxInputsExt())),
		bucketKeyToNode:         make(map[string]*Node),
		bucketKeyToReadonlyNode: make(map[string]map[string]*Node),
		txInputs:                make([]*Node, len(tx.GetTxInputs())),
		txInputsExt:             make([]*Node, len(tx.GetTxInputsExt())),
		txOutputs:               make([]*Node, len(tx.GetTxOutputs())),
		txOutputsExt:            make([]*Node, len(tx.GetTxOutputsExt())),
	}
}

// 已经去重。
func (n *Node) getAllChildren() []*Node {
	if n == nil {
		return nil
	}

	result := make([]*Node, 0, len(n.txOutputs)+len(n.txOutputsExt))
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

	for _, vv := range n.readonlyOutputs {
		for _, v := range vv {
			if v != nil && !nodesDuplicate[v.txid] {
				result = append(result, v)
				nodesDuplicate[v.txid] = true
			}
		}

	}
	return result
}

// 如果此交易包含写某个key，那么找到所有读这个key的交易，目的是在打包交易时保证读交易先打包。
func (n *Node) getReadonlyBrotherNodes(confirmed map[*Node]bool) map[*Node]bool {
	result := map[*Node]bool{}
	for i, v := range n.txInputsExt {
		if v == nil {
			continue
		}
		if len(n.tx.GetTxInputsExt()) <= i {
			continue
		}
		ext := n.tx.GetTxInputsExt()[i]
		bk := ext.Bucket + string(ext.Key) // 此 bk 为写
		offset := ext.RefOffset
		if v.txid == "" {
			if rs, ok := v.bucketKeyToReadonlyNode[bk]; ok {
				for _, r := range rs {
					if !confirmed[r] {
						result[r] = true
					}
				}
			}
		} else {
			for _, r := range v.readonlyOutputs[offset] {
				if r != nil {
					if !confirmed[r] {
						result[r] = true
					}
				}
			}
		}

	}
	return result
}

// 如果此交易包含读key，那么找到所有写的交易，目的是在打包交易时保证读交易先打包。
func (n *Node) getWriteBrotherNodes(confirmed map[*Node]bool) map[*Node]bool {
	result := map[*Node]bool{}
	for i, v := range n.readonlyInputs {
		if v == nil {
			continue
		}
		if len(n.tx.GetTxInputsExt()) <= i {
			continue
		}
		ext := n.tx.GetTxInputsExt()[i]
		bk := ext.Bucket + string(ext.Key) // 此 bk 为读
		offset := ext.RefOffset
		if v.txid == "" {
			if w, ok := v.bucketKeyToNode[bk]; ok {
				if !confirmed[w] {
					result[w] = true
				}
			}
		} else {
			w := v.txOutputsExt[offset]
			if w != nil {
				if !confirmed[w] {
					result[w] = true
				}
			}
		}
	}
	return result
}

// 已经去重。
func (n *Node) getAllParent() []*Node {
	if n == nil {
		return nil
	}

	result := make([]*Node, 0, len(n.txInputs)+len(n.txInputsExt))
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
	if node.tx == nil {
		if readonly {
			a := offset + 1 - len(node.readonlyOutputs)
			if a > 0 {
				node.readonlyOutputs = append(node.readonlyOutputs, make([]map[string]*Node, a)...)
			}
		} else {
			indexa := offset + 1 - len(node.txOutputsExt)
			if indexa > 0 {
				if !readonly {
					node.txOutputsExt = append(node.txOutputsExt, make([]*Node, index)...)
				} else {

				}
			}
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
		if node.readonlyOutputs[offset] == nil {
			node.readonlyOutputs[offset] = map[string]*Node{n.txid: n}
		} else {
			node.readonlyOutputs[offset][n.txid] = n
		}

		n.readonlyInputs[index] = node
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
		offset := n.tx.TxInputs[i].RefOffset
		fn.txOutputs[offset] = nil // 转账交易不会有空的 refTxid，不需要判断是否越界。
		n.txInputs[i] = nil
	}

	inputKeys := make([]string, 0, len(n.tx.GetTxInputsExt()))
	for i, fn := range n.txInputsExt {
		bucket := n.tx.TxInputsExt[i].GetBucket()
		key := n.tx.TxInputsExt[i].GetKey()
		bk := bucket + string(key)
		inputKeys = append(inputKeys, bk)

		if fn == nil {
			continue
		}
		offset := int(n.tx.TxInputsExt[i].RefOffset)
		if len(fn.txOutputsExt) > offset { // 对于第一次写某个 key，父节点为 emptyTxIDNode，txOutputsExt 为空。
			fn.txOutputsExt[offset] = nil
		}

		if nn, ok := fn.bucketKeyToNode[bk]; ok {
			if nn.txid == n.txid {
				delete(fn.bucketKeyToNode, bk)
			}
		}

		n.txInputsExt[i] = nil
	}

	for i, fn := range n.readonlyInputs {
		if fn == nil {
			continue
		}
		offset := 0
		if len(n.tx.TxInputsExt) > 0 {
			offset = int(n.tx.TxInputsExt[i].RefOffset)
		}

		delete(fn.readonlyOutputs[offset], n.txid)

		if fn.txid == "" {
			for _, bk := range inputKeys {
				if m, ok := fn.bucketKeyToReadonlyNode[bk]; ok {
					if m == nil {
						continue
					}
					if _, ok := fn.bucketKeyToReadonlyNode[bk][n.txid]; ok {
						delete(fn.bucketKeyToReadonlyNode[bk], n.txid)
						if len(fn.bucketKeyToReadonlyNode[bk]) == 0 {
							delete(fn.bucketKeyToReadonlyNode, bk)
						}
					}
				}
			}
		}
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

	for _, n := range n.readonlyInputs {
		if n != nil {
			sum++
		}
	}

	return sum
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
