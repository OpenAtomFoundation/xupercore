package tx

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/lib/logs"
)

const (
	defaultMempoolUnconfirmedLen = 5000                             // 默认未确认交易表大小为5000。
	defaultMempoolConfirmedLen   = defaultMempoolUnconfirmedLen / 2 // 默认确认交易表大小为2500。
	defaultMempoolOrphansLen     = defaultMempoolUnconfirmedLen / 5 // 默认孤儿交易表大小为1000。
)

var (
	emptyTxIDNode *Node
	stoneNode     *Node // 所有的子节点都是存在交易，即所有的 input 和 output 都是空，意味着这些交易是从石头里蹦出来的（emmm... 应该能说得过去）。

	stoneNodeID string = "stoneNodeID" // 暂定
)

// Mempool tx mempool.
type Mempool struct {
	log logs.Logger

	Tx *Tx
	// 所有的交易都在下面的三个集合中。三个集合中的元素不会重复。
	confirmed   map[string]*Node // txID => *Node，所有的未确认交易树的 root，也就是确认交易。
	unconfirmed map[string]*Node // txID => *Node，所有未确认交易的集合。
	orphans     map[string]*Node // txID => *Node，所有的孤儿交易。

	bucketKeyNodes map[string]map[string]*Node // 所有引用了某个 key 的交易作为一个键值对，无论只读或者读写。

	m *sync.RWMutex
}

// NewMempool new mempool.
func NewMempool(tx *Tx, log logs.Logger) *Mempool {
	m := &Mempool{
		log:            log,
		Tx:             tx,
		confirmed:      make(map[string]*Node, defaultMempoolConfirmedLen),
		unconfirmed:    make(map[string]*Node, defaultMempoolUnconfirmedLen),
		orphans:        make(map[string]*Node, defaultMempoolOrphansLen),
		bucketKeyNodes: make(map[string]map[string]*Node, defaultMempoolUnconfirmedLen),
		m:              &sync.RWMutex{},
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

// Range 按照拓扑排序遍历节点交易.
func (m *Mempool) Range(f func(tx *pb.Transaction) bool) {
	m.m.Lock()
	defer m.m.Unlock()

	m.log.Debug("Mempool Range", "confirmed", len(m.confirmed), "unconfirmed", len(m.unconfirmed), "orphans", len(m.orphans), "bucketKeyNodes", len(m.bucketKeyNodes))

	nodeInputSumMap := make(map[*Node]int, len(m.confirmed))

	forRangeNode := make([]*Node, 0, len(m.confirmed))
	for _, n := range m.confirmed { // 先把 confirmed 中的交易放入要遍历的列表。
		forRangeNode = append(forRangeNode, n)
	}

	for len(forRangeNode) > 0 {
		tmpNodes := make([]*Node, 0, len(m.confirmed))
		for _, root := range forRangeNode {
			for _, n := range root.txOutputs {
				if m.isNextNode(n, false, nodeInputSumMap) {
					if f != nil && !f(n.tx) {
						return
					}
					tmpNodes = append(tmpNodes, n)
				}
			}

			for _, n := range root.txOutputsExt {
				if m.isNextNode(n, false, nodeInputSumMap) {
					if f != nil && !f(n.tx) {
						return
					}
					tmpNodes = append(tmpNodes, n)
				}
			}

			for _, n := range root.readonlyOutputs {
				if m.isNextNode(n, true, nodeInputSumMap) {
					if f != nil && !f(n.tx) {
						return
					}
					tmpNodes = append(tmpNodes, n)
				}
			}

			for _, n := range root.bucketKeyToNode {
				if m.isNextNode(n, false, nodeInputSumMap) {
					if f != nil && !f(n.tx) {
						return
					}
					tmpNodes = append(tmpNodes, n)
				}
			}
		}
		forRangeNode = tmpNodes
	}
}

// GetTxCounnt get 获取未确认交易与孤儿交易总数
func (m *Mempool) GetTxCounnt() int {
	return len(m.unconfirmed) + len(m.orphans)
}

// PutTx put tx.
func (m *Mempool) PutTx(tx *pb.Transaction) error {
	if tx == nil {
		return errors.New("can not put nil tx into mempool")
	}
	m.m.Lock()
	defer m.m.Unlock()

	m.log.Debug("Mempool PutTx", "txid", tx.HexTxid())

	// tx 可能是确认交易、未确认交易以及孤儿交易，检查双花。
	txid := string(tx.Txid)
	if _, ok := m.confirmed[txid]; ok {
		return errors.New("tx already in mempool confirmd, txid:" + tx.HexTxid())
	}
	if _, ok := m.unconfirmed[txid]; ok {
		return errors.New("tx already in mempool unconfirmd, txid:" + tx.HexTxid())
	}

	if n, ok := m.orphans[txid]; ok {
		if n.tx != nil {
			return errors.New("tx already in mempool orphans, txid:" + tx.HexTxid())
		}
	}

	return m.putTx(tx, false)
}

// DeleteConflictByTx 删除所有与 tx 冲突的交易以及子交易。返回所有删除的交易。
func (m *Mempool) DeleteConflictByTx(tx *pb.Transaction) []*pb.Transaction {
	if m.HasTx(string(tx.GetTxid())) {
		// 如果 mempool 中有此交易，说明没有冲突交易，在 PutTx 时会保证冲突。
		return nil
	}
	m.m.Lock()
	defer m.m.Unlock()

	m.log.Debug("Mempool DeleteConflictByTx", "txid", tx.HexTxid())

	deletedTxs := make([]*pb.Transaction, 0, 0)
	for _, txInput := range tx.TxInputs {
		deletedTxs = append(deletedTxs, m.deleteByUtxo(string(txInput.RefTxid), int(txInput.RefOffset))...)
	}

	deletedTxs = append(deletedTxs, m.deleteBucketKeyByTx(tx)...)
	return deletedTxs
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

// deleteByUtxo delete txs by utxo(addr & txid & offset) 暂时 addr 没用到，根据 txid 和 offset 就可以锁定一个 utxo。
func (m *Mempool) deleteByUtxo(txid string, offset int) []*pb.Transaction {

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

	result = append(result, n.tx)
	result = append(result, m.deleteTx(n.txid)...)
	return result
}

func (m *Mempool) deleteBucketKeyByTx(tx *pb.Transaction) []*pb.Transaction {
	usedKeyVersion := getTxUsedKeyVersion(tx)
	result := make([]*pb.Transaction, 0, 0)
	for k := range usedKeyVersion {
		nodes, ok := m.bucketKeyNodes[k]
		if !ok {
			continue
		}

		for _, n := range nodes {
			result = append(result, m.deleteUsedKeyVersion(n, usedKeyVersion)...)
		}
	}
	return nil
}

func (m *Mempool) deleteUsedKeyVersion(node *Node, usedKeyVersion map[string]string) []*pb.Transaction {
	result := make([]*pb.Transaction, 0, 0)
	outKeys := make(map[string]struct{})
	tx := node.tx
	for _, output := range tx.GetTxOutputsExt() {
		outKeys[output.GetBucket()+string(output.GetKey())] = struct{}{}
	}

	for _, input := range tx.GetTxInputsExt() {
		bk := input.GetBucket() + string(input.GetKey())
		if _, ok := outKeys[bk]; ok { // 说明 bk 非只读。
			if v, ok := usedKeyVersion[bk]; ok { // 说明 bk 某个 version 已经被用掉了。
				if v == makeVersion(input.GetRefTxid(), input.GetRefOffset()) { // 说明 input 引用的 bk 的 version 已经被用掉了。
					result = append(result, node.tx)
					result = append(result, m.deleteTx(node.txid)...)
				}
			}
		}
	}

	return result
}

// 返回 key：bucket+key，value：version。
func getTxUsedKeyVersion(tx *pb.Transaction) map[string]string {
	keyVersion := make(map[string]string, len(tx.GetTxOutputsExt()))

	outKeys := make(map[string]struct{})
	for _, output := range tx.GetTxOutputsExt() {
		outKeys[output.GetBucket()+string(output.GetKey())] = struct{}{}
	}

	for _, input := range tx.GetTxInputsExt() {
		bk := input.GetBucket() + string(input.GetKey())
		if _, ok := outKeys[bk]; ok {
			keyVersion[bk] = makeVersion(input.GetRefTxid(), input.GetRefOffset())
		}
	}

	return keyVersion
}

func makeVersion(txid []byte, offset int32) string {
	return fmt.Sprintf("%x_%d", txid, offset)
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

// DeleteTxAndChildren delete tx from mempool.
func (m *Mempool) DeleteTxAndChildren(txid string) []*pb.Transaction { // DeletTeTxAndChildren
	m.m.Lock()
	defer m.m.Unlock()

	m.log.Debug("Mempool DeleteTxAndChildren", "txid", hex.EncodeToString([]byte(txid)))

	if _, ok := m.confirmed[txid]; ok {
		// TODO 是否删除确认交易表中的交易。不应该删除，confirmed 中应该是已经共识确认过的，回滚区块应该调用 retrieveTx 接口。
		// 本次先按照删除处理。
	}
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

// ConfirmTxID txid
func (m *Mempool) ConfirmTxID(txid string) {
	m.m.RLock()
	defer m.m.RUnlock()

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
}

// ConfirmTx confirm tx.
// 将 tx 从未确认交易表放入确认交易表，或者删除。
func (m *Mempool) ConfirmTx(tx *pb.Transaction) error {
	m.m.RLock()
	defer m.m.RUnlock()

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

func (m *Mempool) isNextNode(node *Node, readonly bool, inputSum map[*Node]int) bool {
	if node == nil {
		return false
	}

	if sum, ok := inputSum[node]; ok {
		if sum <= 1 { // 如果此时节点入度为1，说明只剩下当前父节点这一个依赖，返回 true。
			delete(inputSum, node)
			return true
		}
		inputSum[node] = sum - 1
	} else {
		sum := node.getInputSum() - 1
		if sum <= 0 { // 第一次遍历到此节点，且入度为1，返回true。
			return true
		}
		inputSum[node] = sum
	}

	return false
}

// putTx 添加交易核心逻辑。
func (m *Mempool) putTx(tx *pb.Transaction, retrieve bool) error {
	var node *Node
	if n, ok := m.orphans[string(tx.Txid)]; ok {
		node = n
	} else {
		node = NewNode(string(tx.Txid), tx)
	}

	// 存证交易。
	if len(tx.GetTxInputs()) == 0 && len(tx.GetTxInputs()) == 0 &&
		len(tx.GetTxInputs()) == 0 && len(tx.GetTxInputs()) == 0 {
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
	if stoneNode == nil {
		stoneNode = NewNode(stoneNodeID, nil)
	}
	m.confirmed[stoneNode.txid] = stoneNode
	stoneNode.readonlyOutputs[node.txid] = node
	node.readonlyInputs[stoneNode.txid] = stoneNode
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
	if isOrphan {
		return
	}
	m.checkAndMoveOrphan(node)
}

// 遍历子节点，如果是孤儿交易，遍历孤儿交易的所有父节点，如果所有父节点都在确认表或者未确认表时，此交易加入未确认表，否则此交易还是孤儿交易。
func (m *Mempool) checkAndMoveOrphan(node *Node) {

	orphans := make([]*Node, 0, len(node.txOutputs)+len(node.txOutputsExt))
	for _, n := range node.txOutputs {
		if n == nil {
			continue
		}
		if _, ok := m.orphans[n.txid]; ok {
			orphans = append(orphans, n)
		}
	}

	for _, n := range node.txOutputsExt {
		if n == nil {
			continue
		}
		if _, ok := m.orphans[n.txid]; ok {
			orphans = append(orphans, n)
		}
	}

	for _, n := range node.readonlyOutputs {
		if n == nil {
			continue
		}
		if _, ok := m.orphans[n.txid]; ok {
			orphans = append(orphans, n)
		}
	}

	m.processOrphansToUnconfirmed(orphans)
}

// orphans 这些孤儿节点的父节点中，有一个父节点加入到了未确认交易表或者确认交易表，所以遍历所有子交易看看是否也可以加入未确认交易表。
func (m *Mempool) processOrphansToUnconfirmed(orphans []*Node) {
	if len(orphans) == 0 {
		return
	}
	nodes := orphans

	for len(nodes) > 0 {

		tmp := make([]*Node, 0)

		for _, n := range nodes {
			allFatherFound := true

			for _, v := range n.txInputs {
				if ok := m.inConfirmedOrUnconfirmed(v.txid); !ok {
					allFatherFound = false
					break
				}
			}

			if allFatherFound {
				for _, v := range n.txInputsExt {
					if ok := m.inConfirmedOrUnconfirmed(v.txid); !ok {
						allFatherFound = false
						break
					}
				}
			}

			if allFatherFound {
				for _, v := range n.readonlyInputs {
					if ok := m.inConfirmedOrUnconfirmed(v.txid); !ok {
						allFatherFound = false
						break
					}
				}
			}

			if allFatherFound {
				delete(m.orphans, n.txid)
				m.unconfirmed[n.txid] = n
				tmp = append(tmp, n.getAllChildren()...)
			} else {
				tmp = append(tmp, n.getAllParent()...)
			}
		}

		nodes = tmp
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

func (m *Mempool) pruneSlice(res []*Node, maxLen int) []*Node {
	tmp := len(res) - maxLen
	if tmp > 0 {
		for _, n := range res[tmp:] {
			m.deleteTx(n.txid)
		}

		res = res[:tmp]
	}
	return res
}

// 删除 node 的所有子交易，先从 orphans 中查找。
func (m *Mempool) deleteChildrenFromNode(node *Node) []*pb.Transaction {
	deletedTxs := make([]*pb.Transaction, 0, 10)
	children := node.getAllChildren()

	for len(children) > 0 {
		tmp := make(map[string]*Node, 0)
		for _, n := range children {
			if _, ok := m.orphans[n.txid]; ok {
				delete(m.orphans, n.txid)
			} else if _, ok := m.unconfirmed[n.txid]; ok {
				delete(m.unconfirmed, n.txid)
			} else {
				continue // 按道理不应出现此情况。
			}

			deletedTxs = append(deletedTxs, n.tx)
			for _, v := range n.getAllChildren() {
				if m.inMempool(v.txid) {
					tmp[v.txid] = v
				}
			}

			n.breakOutputs()
			m.deleteBucketKey(n)
		}
		next := make([]*Node, 0, len(tmp))
		for _, v := range tmp {
			next = append(next, v)
		}
		children = next
	}

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
	if emptyTxIDNode == nil {
		emptyTxIDNode = NewNode("", nil)
	}

	m.confirmed[""] = emptyTxIDNode
	if node.isReadonlyKey(index) {
		emptyTxIDNode.readonlyOutputs[node.txid] = node
		node.readonlyInputs[emptyTxIDNode.txid] = emptyTxIDNode
	} else {
		if _, ok := emptyTxIDNode.bucketKeyToNode[bk]; ok {
			return errors.New("bucket and key invalid:" + bucket + "_" + string(key))
		}
		emptyTxIDNode.bucketKeyToNode[bk] = node
		node.txInputsExt[index] = emptyTxIDNode
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

		} else if n, ok1 := m.unconfirmed[id]; ok1 {
			offset := int(input.RefOffset)
			if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
				return isOrphan, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else if n, ok2 := m.orphans[id]; ok2 {
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
				if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
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

func (m *Mempool) processConflict(tx *pb.Transaction) error {
	for _, input := range tx.GetTxInputs() {
		id := string(input.GetRefTxid())
		offset := int(input.GetRefOffset())

		m.updateNode(tx, id, offset)
	}

	for i, input := range tx.GetTxInputsExt() {
		id := string(input.GetRefTxid())
		offset := int(input.GetRefOffset())

		tmpNode := &Node{
			txid: string(tx.GetTxid()),
			tx:   tx,
		}

		if !tmpNode.isReadonlyKey(i) {
			m.updateNode(tx, id, offset)
		}
	}
	return nil
}

func (m *Mempool) updateNode(tx *pb.Transaction, refTxid string, offset int) {
	if n, ok := m.unconfirmed[refTxid]; ok {
		if conflictNode := n.txOutputs[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
			m.putTx(tx, false)
		}
	} else if n, ok := m.orphans[refTxid]; ok {
		if conflictNode := n.txOutputs[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
			m.putTx(tx, false)
		}
	}
}

func (m *Mempool) moveToConfirmed(node *Node) {
	nodes := []*Node{node}

	for len(nodes) > 0 {
		tmp := make([]*Node, 0)
		for _, n := range nodes {
			tmp = append(tmp, n.getAllParent()...)

			n.breakOutputs() // 断绝父子关系
			m.confirmed[n.txid] = n

			delete(m.unconfirmed, n.txid)
			delete(m.orphans, n.txid)

			// 遍历所有子交易，判断是否需要将孤儿交易移动到未确认交易表
			m.checkAndMoveOrphan(n)
			m.deleteBucketKey(n)
		}
		nodes = tmp
	}

	m.cleanConfirmedTxs()
}

// 确认交易表中，如果有出度为0的交易，删除此交易。
func (m *Mempool) cleanConfirmedTxs() {
	for k, v := range m.confirmed {
		if k == "" || k == stoneNodeID {
			continue
		}
		if len(v.bucketKeyToNode) != 0 {
			continue
		}

		if len(v.readonlyOutputs) != 0 {
			continue
		}

		hasChild := false
		for _, vv := range v.txOutputs {
			if vv != nil {
				hasChild = true
				break
			}
		}
		if hasChild {
			continue
		}

		for _, vv := range v.txOutputsExt {
			if vv != nil {
				hasChild = true
				break
			}
		}
		if hasChild {
			continue
		}

		delete(m.confirmed, k)
	}
}
