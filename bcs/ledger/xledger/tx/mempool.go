package tx

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/gammazero/deque"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/lib/logs"
)

const (
	defaultMempoolUnconfirmedLen = 5000                             // 默认未确认交易表大小为5000。
	defaultMempoolConfirmedLen   = defaultMempoolUnconfirmedLen / 2 // 默认确认交易表大小为2500。
	defaultMempoolOrphansLen     = defaultMempoolUnconfirmedLen / 5 // 默认孤儿交易表大小为1000。

	defaultMaxtxLimit = 100000 // 默认 mempool 中最多10w个未确认交易。

	stoneNodeID = "stoneNodeID"
)

var (
	// ErrTxExist tx already in mempool when put tx.
	ErrTxExist = errors.New("tx already in mempool")
)

// Mempool tx mempool.
type Mempool struct {
	log logs.Logger

	txLimit int

	Tx *Tx
	// 所有的交易都在下面的三个集合中。三个集合中的元素不会重复。
	confirmed   map[string]*Node // txID => *Node，所有的未确认交易树的 root，也就是确认交易。
	unconfirmed map[string]*Node // txID => *Node，所有未确认交易的集合。
	orphans     map[string]*Node // txID => *Node，所有的孤儿交易。

	bucketKeyNodes map[string]map[string]*Node // 所有引用了某个 key 的交易作为一个键值对，无论只读或者读写。

	emptyTxIDNode *Node
	stoneNode     *Node // 所有的子节点都是存在交易，即所有的 input 和 output 都是空，意味着这些交易是从石头里蹦出来的（emmm... 应该能说得过去）。

	m *sync.Mutex
}

// Debug Debug mempool.
func (m *Mempool) Debug() {
	m.m.Lock()
	defer m.m.Unlock()
	m.debug()
}

func (m *Mempool) debug() {
	if m.emptyTxIDNode != nil {
		m.log.Error("Mempool Debug", "confirmedLen", len(m.confirmed), "unconfirmedLen", len(m.unconfirmed), "orphanLen", len(m.orphans), "bucketKeyNodesLen", len(m.bucketKeyNodes),
			"len(m.emptyTxIDNode.readonlyOutputs)", len(m.emptyTxIDNode.readonlyOutputs), "len(m.emptyTxIDNode.bucketKeyToNode)", len(m.emptyTxIDNode.bucketKeyToNode),
			"len(m.emptyTxIDNode.bucketKeyToReadonlyNode)", len(m.emptyTxIDNode.bucketKeyToReadonlyNode),
			"len(m.emptyTxIDNode.txOutputsExt)", len(m.emptyTxIDNode.txOutputsExt))
	} else {
		m.log.Error("Mempool Debug", "confirmedLen", len(m.confirmed), "unconfirmedLen",
			len(m.unconfirmed), "orphanLen", len(m.orphans), "bucketKeyNodesLen", len(m.bucketKeyNodes))
	}
}

// NewMempool new mempool.
func NewMempool(tx *Tx, log logs.Logger, txLimit int) *Mempool {
	if txLimit <= 0 {
		txLimit = defaultMaxtxLimit
	}
	m := &Mempool{
		log:            log,
		txLimit:        txLimit,
		Tx:             tx,
		confirmed:      make(map[string]*Node, defaultMempoolConfirmedLen),
		unconfirmed:    make(map[string]*Node, defaultMempoolUnconfirmedLen),
		orphans:        make(map[string]*Node, defaultMempoolOrphansLen),
		bucketKeyNodes: make(map[string]map[string]*Node, defaultMempoolUnconfirmedLen),
		m:              &sync.Mutex{},
	}

	// go m.gc() // 目前此版本不会有孤儿交易进入 mempool。
	return m
}

// HasTx has tx in mempool.
func (m *Mempool) HasTx(txid string) bool {
	m.m.Lock()
	defer m.m.Unlock()
	if _, ok := m.unconfirmed[txid]; ok {
		return true
	}
	if _, ok := m.confirmed[txid]; ok {
		return true
	}
	if n, ok := m.orphans[txid]; ok {
		if n.tx != nil {
			return true
		}
	}
	return false
}

// Range 按照拓扑排序遍历节点交易。
func (m *Mempool) Range(f func(tx *pb.Transaction) bool) {
	if f == nil {
		return
	}

	m.m.Lock()
	defer func() {
		if err := recover(); err != nil {
			m.log.Error("Mempool Range panic", "error", err)
		}
		m.m.Unlock()
	}()

	m.log.Debug("Mempool Range", "confirmed", len(m.confirmed), "unconfirmed", len(m.unconfirmed), "orphans", len(m.orphans), "bucketKeyNodes", len(m.bucketKeyNodes))
	var q deque.Deque
	nodeInputSumMap := make(map[*Node]int, len(m.confirmed))
	confirmed := make(map[*Node]bool, len(m.confirmed))
	for _, n := range m.confirmed { // 先把 confirmed 中的交易放入要遍历的列表。
		q.PushBack(n)
		confirmed[n] = true
	}

	// key为只读交易，value为读写交易，所有key。
	readToWriteNodes := make(map[*Node]map[*Node]bool, 0)

	// 写交易为key，所有只读交易为value，针对同一个bucket+key。
	writeToReadonlyNodes := make(map[*Node]map[*Node]bool, 0)

	// 写交易为key，所有遍历过的只读交易为value，针对同一个bucket+key。
	writeToRangedReadNodes := make(map[*Node]map[*Node]bool, 0)

	// 如果某个写交易可以被打包，但是同个key的读交易还未打包，那么写交易放到此map，等到读交易全部打包完成再打包写交易
	waitRangeNodes := make(map[*Node]bool, 0)

	// 记录处理过读写关系的节点。
	processedRWRefNodes := make(map[*Node]bool, 0)

	for q.Len() > 0 {
		node := q.PopFront().(*Node)

		for _, children := range node.readonlyOutputs {
			for _, n := range children {
				if n == nil {
					continue
				}
				if !processedRWRefNodes[n] {
					m.processRWRefNodes(n, readToWriteNodes, writeToReadonlyNodes, confirmed)
					processedRWRefNodes[n] = true
				}
				if m.isNextNode(n, false, nodeInputSumMap) {
					if !m.processReadAndWriteNodes(n, readToWriteNodes, writeToReadonlyNodes, writeToRangedReadNodes, waitRangeNodes, f, &q) {
						return
					}
				}
			}
		}

		// 只读交易，引用的key的version为空，此时 node 为 emptyTxIDNode。
		for _, nodeMap := range node.bucketKeyToReadonlyNode {
			for _, n := range nodeMap {
				if n == nil {
					continue
				}
				if !processedRWRefNodes[n] {
					m.processRWRefNodes(n, readToWriteNodes, writeToReadonlyNodes, confirmed)
					processedRWRefNodes[n] = true
				}
				if m.isNextNode(n, false, nodeInputSumMap) {
					if !m.processReadAndWriteNodes(n, readToWriteNodes, writeToReadonlyNodes, writeToRangedReadNodes, waitRangeNodes, f, &q) {
						return
					}
				}
			}
		}

		// utxo 子交易
		for _, n := range node.txOutputs {
			if n == nil {
				continue
			}
			if !processedRWRefNodes[n] {
				m.processRWRefNodes(n, readToWriteNodes, writeToReadonlyNodes, confirmed)
				processedRWRefNodes[n] = true
			}
			if m.isNextNode(n, false, nodeInputSumMap) {
				if !m.processReadAndWriteNodes(n, readToWriteNodes, writeToReadonlyNodes, writeToRangedReadNodes, waitRangeNodes, f, &q) {
					return
				}
			}
		}

		// 写子交易，当前n为写交易
		for _, n := range node.txOutputsExt {
			if n == nil {
				continue
			}
			if !processedRWRefNodes[n] {
				m.processRWRefNodes(n, readToWriteNodes, writeToReadonlyNodes, confirmed)
				processedRWRefNodes[n] = true
			}
			if m.isNextNode(n, false, nodeInputSumMap) {
				if !m.processReadAndWriteNodes(n, readToWriteNodes, writeToReadonlyNodes, writeToRangedReadNodes, waitRangeNodes, f, &q) {
					return
				}
			}
		}

		// 此时 node 为 emptyTxIDNode，此时n为写交易，refTxid为空。
		for _, n := range node.bucketKeyToNode { // 读写，此处要判断只读交易是否都已经打包
			if n == nil {
				continue
			}
			if !processedRWRefNodes[n] {
				m.processRWRefNodes(n, readToWriteNodes, writeToReadonlyNodes, confirmed)
				processedRWRefNodes[n] = true
			}
			if m.isNextNode(n, false, nodeInputSumMap) {
				if !m.processReadAndWriteNodes(n, readToWriteNodes, writeToReadonlyNodes, writeToRangedReadNodes, waitRangeNodes, f, &q) {
					return
				}
			}
		}
	}
}

func (m *Mempool) processRWRefNodes(n *Node, readToWriteNodes, writeToReadonlyNodes map[*Node]map[*Node]bool, confirmed map[*Node]bool) {
	ws := n.getWriteBrotherNodes(confirmed)
	if _, ok := readToWriteNodes[n]; !ok {
		readToWriteNodes[n] = ws
	} else {
		for w := range ws {
			readToWriteNodes[n][w] = true
		}
	}
	rs := n.getReadonlyBrotherNodes(confirmed)
	if _, ok := writeToReadonlyNodes[n]; !ok {
		writeToReadonlyNodes[n] = rs
	} else {
		for r := range rs {
			writeToReadonlyNodes[n][r] = true
		}
	}
}

// 当 n 的入度为0时，判断相关联的只读、读写交易。
func (m *Mempool) processReadAndWriteNodes(n *Node, readToWriteNodes, writeToReadonlyNodes, writeToRangedReadNodes map[*Node]map[*Node]bool,
	waitRangeNodes map[*Node]bool, f func(tx *pb.Transaction) bool, q *deque.Deque) bool {
	ranged := false
	if rs, ok := writeToReadonlyNodes[n]; ok { // 此时n作为一个写交易，存在需要先打包的读交易。
		if len(rs) <= len(writeToRangedReadNodes[n]) { // 已经打包的依赖只读交易如果大于等于依赖的所有的只读交易，说明此交易可打包。
			if !f(n.tx) {
				return false
			}
			ranged = true
			q.PushBack(n)
			delete(waitRangeNodes, n)
		} else {
			waitRangeNodes[n] = true // 此时n作为写交易，有依赖的只读交易还未打包。
		}
	} else { // 没有依赖的只读交易，可以直接打包此交易。
		if !f(n.tx) {
			return false
		}
		ranged = true
		q.PushBack(n)
		delete(waitRangeNodes, n)
	}

	if ws, ok := readToWriteNodes[n]; ok { // 此时n作为一个只读交易，有依赖于n的读写交易。
		for w := range ws { // 遍历所有依赖n的读写交易，检查是否有可以打包的读写交易。
			if ranged { // 更新读写交易的依赖的已经打包的只读交易，
				if _, ok := writeToRangedReadNodes[w]; ok {
					writeToRangedReadNodes[w][n] = true
				} else {
					writeToRangedReadNodes[w] = map[*Node]bool{n: true}
				}
			}
			if rs, ok := writeToReadonlyNodes[w]; ok {
				// 如果写交易入度为0，判断依赖的只读交易是否已经全部打包。
				if waitRangeNodes[w] && len(rs) <= len(writeToRangedReadNodes[w]) {
					// 递归处理
					if !m.processReadAndWriteNodes(w, readToWriteNodes, writeToRangedReadNodes, writeToRangedReadNodes, waitRangeNodes, f, q) {
						return false
					}
				}
			}
		}
	}
	return true
}

// GetTxCounnt get 获取未确认交易与孤儿交易总数
func (m *Mempool) GetTxCounnt() int {
	m.m.Lock()
	defer m.m.Unlock()
	return len(m.unconfirmed) + len(m.orphans)
}

// Full 交易池满了返回 true
func (m *Mempool) Full() bool {
	m.m.Lock()
	defer m.m.Unlock()
	return len(m.unconfirmed) >= m.txLimit
}

// PutTx put tx. TODO：后续判断新增的交易是否会导致循环依赖。
func (m *Mempool) PutTx(tx *pb.Transaction) error {
	if tx == nil {
		return errors.New("can not put nil tx into mempool")
	}
	m.m.Lock()
	defer m.m.Unlock()

	if len(m.unconfirmed) >= m.txLimit {
		return errors.New("The tx mempool is full")
	}

	m.log.Debug("Mempool PutTx", "txid", tx.HexTxid())

	// tx 可能是确认交易、未确认交易以及孤儿交易，检查双花。
	txid := string(tx.Txid)
	if _, ok := m.confirmed[txid]; ok {
		m.log.Warn("tx already in mempool confirmd", "txid:", tx.HexTxid())
		return ErrTxExist
	}
	if _, ok := m.unconfirmed[txid]; ok {
		m.log.Warn("tx already in mempool unconfirmd", "txid:", tx.HexTxid())
		return ErrTxExist
	}

	if n, ok := m.orphans[txid]; ok {
		if n.tx != nil {
			m.log.Warn("tx already in mempool orphans", "txid:", tx.HexTxid())
			return ErrTxExist
		}
	}

	return m.putTx(tx, false)
}

// FindConflictByTx 找到所有与 tx 冲突的交易。返回数组中，前面是子交易，后面是父交易。
// 保证事物原子性，此接口不删除交易，只返回交易列表，如果需要删除需要调用删除交易相关接口。
// 1.区块内只读交易不在mempool，mempool中有写交易无论在不在区块内，找到所有mempool中的写交易，上层进行undo；
// 2.区块内写交易无论在不在mempool，mempool中只读交易（不在区块内），找到所有mempool中的只读交易，上层进行undo；
// 3.区块内写交易无论在不在mempool，mempool中写交易，mempool 中的写交易为无效交易，上层进行undo。
func (m *Mempool) FindConflictByTx(tx *pb.Transaction, txidInBlock map[string]bool, ranged map[*Node]bool) []*pb.Transaction {
	m.m.Lock()
	defer m.m.Unlock()

	m.log.Debug("Mempool FindConflictByTx", "txid", tx.HexTxid())

	conflictTxs := make([]*pb.Transaction, 0, 10)
	if n, ok := m.unconfirmed[string(tx.Txid)]; ok {
		ranged[n] = true
	}
	for _, txInput := range tx.TxInputs {
		// 根据 utxo 找到冲突的所有交易以及子交易。
		utxoConflictTxs := m.findByUtxo(string(txInput.RefTxid), int(txInput.RefOffset), ranged)
		conflictTxs = append(conflictTxs, utxoConflictTxs...)
	}

	// 根据 tx 找到所有 key 版本冲突的交易以及子交易。
	readonlyKeyVersion, writeKeyVersion := getTxUsedKeyVersion(tx) // 找到当前交易所有引用的 key 的 verison。
	if _, ok := m.unconfirmed[string(tx.Txid)]; !ok {
		// 1.区块内只读交易不在mempool，mempool中有写交易无论在不在区块内，找到所有mempool中的写交易，上层进行undo；
		for refTxid, offsetMap := range readonlyKeyVersion {
			refNode, ok := m.unconfirmed[refTxid]
			if !ok {
				refNode, ok = m.confirmed[refTxid]
				if !ok {
					continue
				}
			}
			for offset, bk := range offsetMap {
				if refTxid == "" {
					wn, ok := refNode.bucketKeyToNode[bk]
					if !ok {
						continue
					}
					if wn != nil && !ranged[wn] {
						conflictTxs = append(conflictTxs, m.findChildrenFromNode(wn, ranged)...)
					}
				} else {
					wn := refNode.txOutputsExt[offset]
					if wn != nil && !ranged[wn] {
						conflictTxs = append(conflictTxs, m.findChildrenFromNode(wn, ranged)...)
					}
				}
			}

		}
	}

	for refTxid, offsetMap := range writeKeyVersion {
		refNode, ok := m.unconfirmed[refTxid]
		if !ok {
			refNode, ok = m.confirmed[refTxid]
			if !ok {
				continue
			}
		}
		for offset, bk := range offsetMap {
			if refTxid == "" {
				rns, ok := refNode.bucketKeyToReadonlyNode[bk]
				if !ok {
					continue
				}
				for _, r := range rns {
					if r != nil && !txidInBlock[r.txid] && !ranged[r] {
						//2.区块内写交易无论在不在mempool，mempool中只读交易（不在区块内），找到所有mempool中的只读交易，上层进行undo；
						conflictTxs = append(conflictTxs, m.findChildrenFromNode(r, ranged)...)
					}
				}
				wn, ok := refNode.bucketKeyToNode[bk]
				if !ok {
					continue
				}
				if wn != nil && wn.txid != string(tx.Txid) && !ranged[wn] {
					// 3.区块内写交易无论在不在mempool，mempool中写交易，mempool 中的写交易为无效交易，上层进行undo。
					conflictTxs = append(conflictTxs, m.findChildrenFromNode(wn, ranged)...)
				}

			} else {
				rns := refNode.readonlyOutputs[offset]
				for _, r := range rns {
					if r != nil && !txidInBlock[r.txid] && !ranged[r] {
						//2.区块内写交易无论在不在mempool，mempool中只读交易（不在区块内），找到所有mempool中的只读交易，上层进行undo；
						conflictTxs = append(conflictTxs, m.findChildrenFromNode(r, ranged)...)
					}
				}
				wn := refNode.txOutputsExt[offset]
				if wn != nil && wn.txid != string(tx.Txid) && !ranged[wn] {
					// 3.区块内写交易无论在不在mempool，mempool中写交易，mempool 中的写交易为无效交易，上层进行undo。
					conflictTxs = append(conflictTxs, m.findChildrenFromNode(wn, ranged)...)
				}
			}
		}
	}

	return conflictTxs
}

func (m *Mempool) doDelNode(node *Node) {
	node.breakOutputs() // 断开 node 与所有父节点的关系。
	m.deleteBucketKey(node)
	delete(m.confirmed, node.txid)
	delete(m.unconfirmed, node.txid)
	delete(m.orphans, node.txid)
}

func (m *Mempool) dfs(node *Node, ranged map[*Node]bool, f func(n *Node)) {
	if ranged[node] {
		return
	}
	for _, v := range node.txOutputs {
		if v != nil && !ranged[node] {
			m.dfs(v, ranged, f)
		}
	}

	for _, v := range node.txOutputsExt {
		if v != nil && !ranged[node] {
			m.dfs(v, ranged, f)
		}
	}

	for _, v := range node.readonlyOutputs {
		for _, n := range v {
			if n != nil && !ranged[node] {
				m.dfs(n, ranged, f)
			}
		}
	}

	ranged[node] = true
	f(node)
}

func (m *Mempool) findChildrenFromNode(node *Node, ranged map[*Node]bool) []*pb.Transaction {
	foundTxs := make([]*pb.Transaction, 0, 10)
	f := func(n *Node) {
		foundTxs = append(foundTxs, n.tx)
	}
	m.dfs(node, ranged, f)
	return foundTxs
}

// GetTx 从 mempool 中查询一笔交易，先查未确认交易表，然后是孤儿交易表。
func (m *Mempool) GetTx(txid string) (*pb.Transaction, bool) {
	m.m.Lock()
	defer m.m.Unlock()

	if n := m.unconfirmed[txid]; n != nil {
		return n.tx, true
	}

	if n := m.orphans[txid]; n != nil {
		return n.tx, true
	}
	return nil, false
}

// findByUtxo delete txs by utxo(addr & txid & offset) 暂时 addr 没用到，根据 txid 和 offset 就可以锁定一个 utxo。
func (m *Mempool) findByUtxo(txid string, offset int, ranged map[*Node]bool) []*pb.Transaction {
	node := m.getNode(txid)
	if node == nil {
		return nil
	}

	if offset >= len(node.txOutputs) {
		return nil
	}
	n := node.txOutputs[offset]
	if n == nil {
		return nil
	}

	result := make([]*pb.Transaction, 0, 100)
	children := m.findChildrenFromNode(n, ranged)
	result = append(result, children...)
	return result
}

func getTxUsedKeyVersion(tx *pb.Transaction) (map[string]map[int]string, map[string]map[int]string) {
	readonlyKeyVersion := make(map[string]map[int]string, len(tx.GetTxOutputsExt()))
	writeKeyVersion := make(map[string]map[int]string, len(tx.GetTxOutputsExt()))

	outKeys := make(map[string]struct{})
	for _, output := range tx.GetTxOutputsExt() {
		outKeys[output.GetBucket()+string(output.GetKey())] = struct{}{}
	}

	for _, input := range tx.GetTxInputsExt() {
		bk := input.GetBucket() + string(input.GetKey())
		if _, ok := outKeys[bk]; ok {
			if tmp, ok := writeKeyVersion[string(input.GetRefTxid())]; ok {
				tmp[int(input.GetRefOffset())] = bk
			} else {
				writeKeyVersion[string(input.GetRefTxid())] = map[int]string{int(input.GetRefOffset()): bk}
			}

		} else {
			if tmp, ok := readonlyKeyVersion[string(input.GetRefTxid())]; ok {
				tmp[int(input.GetRefOffset())] = bk
			} else {
				readonlyKeyVersion[string(input.GetRefTxid())] = map[int]string{int(input.GetRefOffset()): bk}
			}
		}
	}

	return readonlyKeyVersion, writeKeyVersion
}

func (m *Mempool) inUnconfirmedOrOrphans(txid string) bool {
	if _, ok := m.unconfirmed[txid]; ok {
		return true
	}

	if n, ok := m.orphans[txid]; ok {
		if n.tx != nil {
			return true
		}
		return false
	}
	return false
}

func (m *Mempool) getNode(txid string) *Node {
	if n, ok := m.confirmed[txid]; ok {
		return n
	} else if n, ok := m.unconfirmed[txid]; ok {
		return n
	} else if n, ok := m.orphans[txid]; ok {
		return n
	}
	return nil
}

// BatchDeleteTx 从 mempool 删除所有 txs。
func (m *Mempool) BatchDeleteTx(txs []*pb.Transaction) {
	m.m.Lock()
	defer m.m.Unlock()

	m.log.Debug("Mempool BatchDeletx", "txsLen", len(txs))
	for _, tx := range txs {
		m.deleteTx(string(tx.Txid))
	}
}

// DeleteTxAndChildren delete tx from mempool. 返回交易是从子交易到父交易顺序。
func (m *Mempool) DeleteTxAndChildren(txid string) []*pb.Transaction { // DeletTeTxAndChildren
	m.m.Lock()
	defer m.m.Unlock()

	m.log.Debug("Mempool DeleteTxAndChildren", "txid", hex.EncodeToString([]byte(txid)))

	return m.deleteTx(txid)
}

func (m *Mempool) deleteTx(txid string) []*pb.Transaction {
	var (
		node *Node
		ok   bool
	)
	if node, ok = m.unconfirmed[txid]; ok {
		delete(m.unconfirmed, txid)
	} else if node, ok = m.orphans[txid]; ok {
		delete(m.orphans, txid)
	} else if node, ok = m.confirmed[txid]; ok {
		delete(m.confirmed, txid)
	} else {
		return nil
	}

	if node != nil {
		m.deleteBucketKey(node)
		node.breakOutputs()
		return m.deleteChildrenFromNode(node)
	}
	return nil
}

// BatchConfirmTx 批量确认交易
func (m *Mempool) BatchConfirmTx(txs []*pb.Transaction) {
	m.m.Lock()
	defer m.m.Unlock()
	for _, tx := range txs {
		txid := string(tx.GetTxid())
		if _, ok := m.confirmed[txid]; ok {
			// 已经在确认交易表
			continue
		}

		if n, ok := m.unconfirmed[txid]; ok {
			m.moveToConfirmed(n)
		} else if n, ok := m.orphans[txid]; ok {
			if n.tx != nil {
				m.moveToConfirmed(n)
			}
		}
	}

	m.cleanConfirmedTxs()
}

// BatchConfirmTxID 批量确认交易ID
func (m *Mempool) BatchConfirmTxID(txids []string) {
	m.m.Lock()
	defer m.m.Unlock()
	for _, txid := range txids {
		if _, ok := m.confirmed[txid]; ok {
			// 已经在确认交易表
			continue
		}

		if n, ok := m.unconfirmed[txid]; ok {
			m.moveToConfirmed(n)
		} else if n, ok := m.orphans[txid]; ok {
			if n.tx != nil {
				m.moveToConfirmed(n)
			}
		}
	}

	m.cleanConfirmedTxs()
}

// ConfirmTxID txid
func (m *Mempool) ConfirmTxID(txid string) {
	m.m.Lock()
	defer m.m.Unlock()

	m.log.Debug("Mempool ConfirmTxID", "txid", hex.EncodeToString([]byte(txid)))

	if _, ok := m.confirmed[txid]; ok {
		// 已经在确认交易表
		return
	}

	if n, ok := m.unconfirmed[txid]; ok {
		m.moveToConfirmed(n)
	} else if n, ok := m.orphans[txid]; ok {
		if n.tx != nil {
			m.moveToConfirmed(n)
		}
	}

	m.cleanConfirmedTxs()
}

// ConfirmTx confirm tx.
// 将 tx 从未确认交易表放入确认交易表，或者删除。
func (m *Mempool) ConfirmTx(tx *pb.Transaction) error {
	m.m.Lock()
	defer m.m.Unlock()

	m.log.Debug("Mempool ConfirmTx", "txid", tx.HexTxid())

	id := string(tx.Txid)
	if _, ok := m.confirmed[id]; ok {
		// 已经在确认交易表
		return nil
	}

	if n, ok := m.unconfirmed[id]; ok {
		m.moveToConfirmed(n)
	} else if n, ok := m.orphans[id]; ok {
		// n 可能是 mock
		if n.tx == nil {
			m.putTx(tx, true)
		}
		m.moveToConfirmed(n)
	} else {
		// mempool 中所有交易与此交易没有联系，但是可能有冲突交易。
		return m.processConflict(tx)
	}

	m.cleanConfirmedTxs()
	return nil
}

// RetrieveTx tx.
// 将交易恢复到 mempool。与mempool中交易冲突时，保留此交易。
// 此次版本暂时不用此接口。
// func (m *Mempool) RetrieveTx(tx *pb.Transaction) error {
// 	if tx == nil {
// 		return errors.New("tx is nil")
// 	}
// 	m.m.RLock()
// 	defer m.m.RUnlock()

// 	m.log.Debug("Mempool RetrieveTx", "txid", tx.HexTxid())

// 	// tx 可能是确认交易、未确认交易以及孤儿交易，检查双花。
// 	txid := string(tx.Txid)
// 	if _, ok := m.confirmed[txid]; ok {
// 		return nil
// 	}
// 	if _, ok := m.unconfirmed[txid]; ok {
// 		return nil
// 	}

// 	if n, ok := m.orphans[txid]; ok {
// 		if n.tx != nil {
// 			return nil
// 		}
// 	}

// 	return m.putTx(tx, true)
// }

// 暂定每隔十分钟处理一次孤儿交易
// func (m *Mempool) gc() { // todo
// 	ticker := time.NewTicker(time.Minute * 10)
// 	for range ticker.C {
// 		m.gcOrphans()
// 	}
// }

func (m *Mempool) gcOrphans() {
	m.m.Lock()
	defer m.m.Unlock()
	for _, v := range m.orphans {
		if v.tx == nil {
			continue
		}
		recvTimestamp := v.tx.GetTimestamp() // unix nano
		t := time.Unix(0, recvTimestamp)
		if time.Since(t) > time.Second*600 {
			m.deleteTx(v.txid)
		}
	}
}

func (m *Mempool) isNextNode(node *Node, readonly bool, inputSumMap map[*Node]int) bool {
	if node == nil {
		return false
	}

	var inputSum int
	if sum, ok := inputSumMap[node]; ok {
		inputSum = sum - 1
	} else {
		inputSum = node.getInputSum() - 1
	}
	inputSumMap[node] = inputSum // 即使只有一个依赖交易，那么子交易也需要加入到 inputSumMap 中，用来循环依赖判断。

	switch inputSum {
	case 0: // 入度为0，说明所有依赖交易都已经遍历过。
		return true
	case -1: // 入度为-1，说明存在循环引用。
		panic("tx circular dependence in mempool")
	default:
		return false
	}
}

// putTx 添加交易核心逻辑。
func (m *Mempool) putTx(tx *pb.Transaction, retrieve bool) error {
	var node *Node
	if n, ok := m.orphans[string(tx.Txid)]; ok {
		node = n
		if node.tx == nil {
			node.tx = tx
			node.readonlyInputs = make([]*Node, len(tx.GetTxInputsExt()))
			if len(node.bucketKeyToNode) == 0 {
				node.bucketKeyToNode = make(map[string]*Node)
			}
			if len(node.bucketKeyToReadonlyNode) == 0 {
				node.bucketKeyToReadonlyNode = make(map[string]map[string]*Node)
			}

			node.txInputs = make([]*Node, len(tx.GetTxInputs()))
			node.txInputsExt = make([]*Node, len(tx.GetTxInputsExt()))
		}
	} else {
		node = NewNode(string(tx.Txid), tx)
	}

	// 存证交易。
	if len(tx.GetTxInputs()) == 0 && len(tx.GetTxInputsExt()) == 0 {
		m.processEvidenceNode(node)
	}

	var (
		isOrphan bool
		err      error
	)
	// 更新节点的所有父关系。
	isOrphan, err = m.processNodeInputs(node, retrieve)
	if err != nil {
		return err
	}

	if isOrphan {
		m.orphans[node.txid] = node
	} else {
		m.unconfirmed[node.txid] = node
		if _, ok := m.orphans[node.txid]; ok {
			// 如果是 mock orphan，则删除掉。
			delete(m.orphans, node.txid)
		}
	}

	// 更新节点的所有子关系。
	m.processNodeOutputs(node, isOrphan)

	m.putBucketKey(node)
	return nil
}

func (m *Mempool) deleteBucketKey(node *Node) {
	if node.tx == nil {
		return
	}

	for _, input := range node.tx.GetTxInputsExt() {
		key := input.GetBucket() + string(input.GetKey())
		if nodes, ok := m.bucketKeyNodes[key]; ok {
			delete(nodes, node.txid)
			if len(nodes) == 0 {
				delete(m.bucketKeyNodes, key)
			}
		}
	}

	for _, output := range node.tx.GetTxOutputsExt() {
		key := output.GetBucket() + string(output.GetKey())
		if nodes, ok := m.bucketKeyNodes[key]; ok {
			delete(nodes, node.txid)
			if len(nodes) == 0 {
				delete(m.bucketKeyNodes, key)
			}
		}
	}
}

func (m *Mempool) putBucketKey(node *Node) {
	if node.tx == nil {
		return
	}

	for _, input := range node.tx.GetTxInputsExt() {
		key := input.GetBucket() + string(input.GetKey())
		if nodes, ok := m.bucketKeyNodes[key]; ok {
			nodes[node.txid] = node
		} else {
			m.bucketKeyNodes[key] = map[string]*Node{node.txid: node}
		}
	}

	for _, output := range node.tx.GetTxOutputsExt() {
		key := output.GetBucket() + string(output.GetKey())
		if nodes, ok := m.bucketKeyNodes[key]; ok {
			nodes[node.txid] = node
		} else {
			m.bucketKeyNodes[key] = map[string]*Node{node.txid: node}
		}
	}
}

// 处理存证交易（没有任何输入和输出）。
func (m *Mempool) processEvidenceNode(node *Node) {
	if m.stoneNode == nil {
		m.stoneNode = NewNode(stoneNodeID, nil)
		m.stoneNode.readonlyOutputs = append(m.stoneNode.readonlyOutputs, map[string]*Node{node.txid: node})
	}
	m.confirmed[m.stoneNode.txid] = m.stoneNode
	m.stoneNode.readonlyOutputs[0][node.txid] = node
	node.readonlyInputs = append(node.readonlyInputs, m.stoneNode)

	m.unconfirmed[node.txid] = node
}

func (m *Mempool) processNodeInputs(node *Node, retrieve bool) (bool, error) {
	var (
		err              error
		txInputOrphan    bool
		txInputExtOrphan bool
	)

	txInputOrphan, err = m.processTxInputs(node, retrieve)
	if err != nil {
		return false, err
	}
	txInputExtOrphan, err = m.processTxInputsExt(node, retrieve)
	if err != nil {
		return false, err
	}

	return txInputOrphan || txInputExtOrphan, nil
}

func (m *Mempool) processNodeOutputs(node *Node, isOrphan bool) {
	// 如果 node 为 mock orphan，发现孤儿交易引用的 offset 在父交易中不存在，那么此孤儿交易为无效交易，此无效交易的所有子交易也是无效交易
	node.txOutputs = m.pruneSlice(node.txOutputs, len(node.tx.GetTxOutputs()))
	node.txOutputsExt = m.pruneSlice(node.txOutputsExt, len(node.tx.GetTxOutputsExt()))
	m.pruneReadonlyOutputs(node)
	if isOrphan {
		return
	}
	m.checkAndMoveOrphan(node)
}

func (m *Mempool) pruneReadonlyOutputs(node *Node) {
	index := len(node.readonlyOutputs) - len(node.tx.GetTxOutputsExt())
	maxLen := len(node.tx.GetTxOutputsExt())
	if index > 0 { // 说明有孤儿交易依赖于无效的引用。
		for _, txidMap := range node.readonlyOutputs[maxLen:] {
			for txid := range txidMap {
				m.deleteTx(txid)
			}
		}
		node.readonlyOutputs = node.readonlyOutputs[:maxLen]
	}

	if index < 0 {
		node.readonlyOutputs = append(node.readonlyOutputs, make([]map[string]*Node, maxLen-len(node.readonlyOutputs))...)
	}
}

// 遍历子节点，如果是孤儿交易，遍历孤儿交易的所有父节点，如果所有父节点都在确认表或者未确认表时，此交易加入未确认表，否则此交易还是孤儿交易。
func (m *Mempool) checkAndMoveOrphan(node *Node) {
	orphans := make([]*Node, 0, len(node.txOutputs)+len(node.txOutputsExt))
	om := make(map[*Node]bool, len(node.txOutputs)+len(node.txOutputsExt))
	for _, n := range node.txOutputs {
		if n == nil || om[n] {
			continue
		}
		om[n] = true
		if _, ok := m.orphans[n.txid]; ok {
			orphans = append(orphans, n)
		}
	}

	for _, n := range node.txOutputsExt {
		if n == nil || om[n] {
			continue
		}
		om[n] = true
		if _, ok := m.orphans[n.txid]; ok {
			orphans = append(orphans, n)
		}
	}

	for _, n := range node.readonlyOutputs {
		if n == nil {
			continue
		}
		for id, v := range n {
			if v == nil || om[v] {
				continue
			}
			om[v] = true
			if _, ok := m.orphans[id]; ok {
				orphans = append(orphans, v)
			}
		}
	}

	m.processOrphansToUnconfirmed(orphans)
}

// orphans 这些孤儿节点的父节点中，有一个父节点加入到了未确认交易表或者确认交易表，所以遍历所有子交易看看是否也可以加入未确认交易表。
func (m *Mempool) processOrphansToUnconfirmed(orphans []*Node) {
	if len(orphans) == 0 {
		return
	}

	var q deque.Deque
	om := make(map[*Node]bool, len(orphans))
	for _, n := range orphans {
		q.PushBack(n)
		om[n] = true
	}

	for q.Len() > 0 {
		n := q.PopFront().(*Node)
		delete(om, n)
		allFatherFound := true
		childrenMap := make(map[*Node]bool, 100)
		for _, v := range n.txInputs {
			if v == nil || childrenMap[v] {
				continue
			}
			childrenMap[v] = true
			if ok := m.inConfirmedOrUnconfirmed(v.txid); !ok {
				allFatherFound = false
				break
			}
		}

		if allFatherFound {
			for _, v := range n.txInputsExt {
				if v == nil || childrenMap[v] {
					continue
				}
				childrenMap[v] = true
				if ok := m.inConfirmedOrUnconfirmed(v.txid); !ok {
					allFatherFound = false
					break
				}
			}
		}

		if allFatherFound {
			for _, v := range n.readonlyInputs {
				if v == nil || childrenMap[v] {
					continue
				}
				childrenMap[v] = true
				if ok := m.inConfirmedOrUnconfirmed(v.txid); !ok {
					allFatherFound = false
					break
				}
			}
		}

		if allFatherFound {
			delete(m.orphans, n.txid)
			m.unconfirmed[n.txid] = n
			for _, cn := range n.getAllChildren() {
				if _, ok := m.orphans[cn.txid]; ok {
					parent := cn.getAllParent()
					shouldPush := true
					for _, pn := range parent {
						if !om[pn] && !m.inConfirmedOrUnconfirmed(pn.txid) {
							shouldPush = false
							break
						}
					}
					hexid := hex.EncodeToString([]byte(cn.txid))
					if shouldPush {
						if _, ok := om[cn]; ok {
							m.log.Info("Mempool processOrphansToUnconfirmed push back", "in om txid donot push", hexid)
						} else {
							m.log.Info("Mempool processOrphansToUnconfirmed push back", "txid", hexid)
							om[cn] = true
							q.PushBack(cn) // 放入队列的前提是这个节点的所有父交易都是unconfirm或者在队列中
						}
					}
				}
			}
		}
	}
}

func (m *Mempool) inConfirmedOrUnconfirmed(id string) bool {
	_, ok := m.confirmed[id]
	if ok {
		return true
	} else if _, ok = m.unconfirmed[id]; ok {
		return true
	} else {
		return false
	}
}

// 将 res 根据 maxLen 进行裁剪，同时删除掉无效的交易。
func (m *Mempool) pruneSlice(res []*Node, maxLen int) []*Node {
	index := len(res) - maxLen
	if index > 0 { // 说明有孤儿交易依赖于无效的引用。
		for _, n := range res[maxLen:] {
			if n == nil {
				continue
			}
			m.deleteTx(n.txid)
		}
		res = res[:maxLen]
		return res
	}

	if index < 0 {
		res = append(res, make([]*Node, maxLen-len(res))...)
		return res
	}
	return res
}

func (m *Mempool) deleteChildrenFromNode(node *Node) []*pb.Transaction {
	deletedTxs := make([]*pb.Transaction, 0, 10)
	ranged := make(map[*Node]bool, 10)
	f := func(n *Node) {
		deletedTxs = append(deletedTxs, n.tx)
		m.doDelNode(n)
	}
	m.dfs(node, ranged, f)
	return deletedTxs
}

func (m *Mempool) inMempool(txid string) bool {
	if _, ok := m.unconfirmed[txid]; ok {
		return true
	}
	if _, ok := m.confirmed[txid]; ok {
		return true
	}
	if _, ok := m.orphans[txid]; ok {
		return true
	}
	return false
}

// 更新 node 的 TxInputs 字段。
func (m *Mempool) processTxInputs(node *Node, retrieve bool) (bool, error) {
	isOrphan := false
	tx := node.tx
	for i, input := range tx.TxInputs {
		id := string(input.RefTxid)
		if n, ok := m.confirmed[id]; ok {
			if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), n, retrieve); err != nil {
				return false, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else if n, ok := m.unconfirmed[id]; ok {
			if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), n, retrieve); err != nil {
				return false, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else if n, ok := m.orphans[id]; ok {
			isOrphan = true
			if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), n, retrieve); err != nil {
				return false, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else {
			if dbTx, _ := m.queryTxFromDB(id); dbTx != nil {
				n := NewNode(string(dbTx.Txid), dbTx)
				if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), n, retrieve); err != nil {
					return false, err
				} else if forDeleteNode != nil {
					m.deleteTx(forDeleteNode.txid)
				}
				m.confirmed[string(dbTx.Txid)] = n

			} else {
				// 孤儿交易
				orphanNode := NewNode(id, nil)
				if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), orphanNode, retrieve); err != nil {
					return false, err
				} else if forDeleteNode != nil {
					m.deleteTx(forDeleteNode.txid)
				}
				m.orphans[id] = orphanNode
				isOrphan = true
			}
		}
	}

	return isOrphan, nil
}

// txid 为空的 node
func (m *Mempool) processEmptyRefTxID(node *Node, index int) error {
	bucket := node.tx.TxInputsExt[index].GetBucket()
	key := node.tx.TxInputsExt[index].GetKey()
	bk := bucket + string(key)
	offset := node.tx.TxInputsExt[index].GetRefOffset()
	if m.emptyTxIDNode == nil {
		m.emptyTxIDNode = NewNode("", nil)
		m.emptyTxIDNode.bucketKeyToReadonlyNode = map[string]map[string]*Node{}
		m.emptyTxIDNode.readonlyOutputs = append(m.emptyTxIDNode.readonlyOutputs, make([]map[string]*Node, offset+1)...)
	}

	m.confirmed[""] = m.emptyTxIDNode
	if node.isReadonlyKey(index) { // 只读的key，此时index一定为0。
		node.readonlyInputs[index] = m.emptyTxIDNode

		if m.emptyTxIDNode.bucketKeyToReadonlyNode[bk] == nil {
			m.emptyTxIDNode.bucketKeyToReadonlyNode[bk] = make(map[string]*Node, 1)
		}
		m.emptyTxIDNode.bucketKeyToReadonlyNode[bk][node.txid] = node
	} else { // 修改了这个key，也就是 bucketKeyToNode 中全是修改了这个key的node。
		if _, ok := m.emptyTxIDNode.bucketKeyToNode[bk]; ok {
			return errors.New("bucket and key invalid:" + bucket + "_" + string(key))
		}
		m.emptyTxIDNode.bucketKeyToNode[bk] = node
		node.txInputsExt[index] = m.emptyTxIDNode
	}
	return nil
}

func (m *Mempool) processTxInputsExt(node *Node, retrieve bool) (bool, error) {
	isOrphan := false
	tx := node.tx
	for index, input := range tx.TxInputsExt {
		if len(input.GetRefTxid()) == 0 {
			m.processEmptyRefTxID(node, index)
			continue
		}

		id := string(input.RefTxid)
		if n, ok := m.confirmed[id]; ok {
			offset := int(input.RefOffset)
			if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
				return isOrphan, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else if n, ok := m.unconfirmed[id]; ok {
			offset := int(input.RefOffset)
			if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
				return isOrphan, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else if n, ok := m.orphans[id]; ok {
			isOrphan = true
			offset := int(input.RefOffset)
			if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
				return isOrphan, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else {
			if dbTx, _ := m.queryTxFromDB(id); dbTx != nil {
				n := NewNode(string(dbTx.GetTxid()), dbTx)
				offset := int(input.RefOffset)
				if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
					return isOrphan, err
				} else if forDeleteNode != nil {
					m.deleteTx(forDeleteNode.txid)
				}
				m.confirmed[id] = n
			} else {
				// 孤儿交易
				orphanNode := NewNode(id, nil)
				offset := int(input.RefOffset)
				if forDeleteNode, err := node.updateInputExt(index, offset, orphanNode, retrieve); err != nil {
					return isOrphan, err
				} else if forDeleteNode != nil {
					m.deleteTx(forDeleteNode.txid)
				}
				m.orphans[id] = orphanNode
				isOrphan = true
			}
		}
	}

	return isOrphan, nil
}

var (
	isTest bool
	dbTxs  = make(map[string]*pb.Transaction, 10) // for test
)

func (m *Mempool) queryTxFromDB(txid string) (*pb.Transaction, error) {
	if !isTest { // 单测使用。
		return m.Tx.ledger.QueryTransaction([]byte(txid))
	}
	tx, _ := dbTxs[txid]
	return tx, nil
}

// 在 ConfirmTx 时，如果当前交易不在 mempool 中，那么删除掉所有与此交易有冲突的交易。
func (m *Mempool) processConflict(tx *pb.Transaction) error {
	for _, input := range tx.GetTxInputs() {
		id := string(input.GetRefTxid())
		offset := int(input.GetRefOffset())

		m.updateNodeTxInput(tx, id, offset)
	}

	for i, input := range tx.GetTxInputsExt() {
		id := string(input.GetRefTxid())
		offset := int(input.GetRefOffset())

		node := NewNode(string(tx.GetTxid()), tx)

		if !node.isReadonlyKey(i) {
			m.updateNodeTxInputExt(tx, id, offset)
		}
	}
	return nil
}

func (m *Mempool) updateNodeTxInputExt(tx *pb.Transaction, refTxid string, offset int) {
	if n, ok := m.unconfirmed[refTxid]; ok {
		if conflictNode := n.txOutputsExt[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
		}
	} else if n, ok := m.orphans[refTxid]; ok {
		if conflictNode := n.txOutputsExt[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
		}
	}
}

func (m *Mempool) updateNodeTxInput(tx *pb.Transaction, refTxid string, offset int) {
	if n, ok := m.unconfirmed[refTxid]; ok {
		if conflictNode := n.txOutputs[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
		}
	} else if n, ok := m.orphans[refTxid]; ok {
		if conflictNode := n.txOutputs[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
		}
	}
}

func (m *Mempool) moveToConfirmed(node *Node) {
	var q deque.Deque
	q.PushBack(node)
	for q.Len() > 0 {
		n := q.PopFront().(*Node)
		for _, v := range n.getAllParent() {
			if _, ok := m.confirmed[v.txid]; ok {
				continue
			}
			q.PushBack(v)
		}

		n.breakOutputs() // 断绝父子关系
		m.confirmed[n.txid] = n

		delete(m.unconfirmed, n.txid)
		delete(m.orphans, n.txid)

		// 遍历所有子交易，判断是否需要将孤儿交易移动到未确认交易表
		m.checkAndMoveOrphan(n)
		m.deleteBucketKey(n)
	}
}

// 确认交易表中，如果有出度为0的交易，删除此交易。
func (m *Mempool) cleanConfirmedTxs() {
	for id, node := range m.confirmed {
		if id == "" || id == stoneNodeID {
			continue
		}
		if len(node.bucketKeyToNode) != 0 {
			continue
		}
		if len(node.bucketKeyToReadonlyNode) != 0 {
			continue
		}

		hasChild := false
		for _, n := range node.txOutputs {
			if n != nil {
				hasChild = true
				break
			}
		}
		if hasChild {
			continue
		}

		for _, n := range node.txOutputsExt {
			if n != nil {
				hasChild = true
				break
			}
		}
		if hasChild {
			continue
		}

		for _, n := range node.readonlyOutputs {
			if len(n) > 0 {
				hasChild = true
				break
			}
		}
		if hasChild {
			continue
		}

		delete(m.confirmed, id)
	}
}
