package tdpos

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"

	"github.com/xuperchain/xupercore/kernel/contract"
)

// 本文件实现tdpos的原Run方法，现全部移至三代合约
// tdpos原有的9个db存储现转化为一个bucket，4个key的链上存储形式
// contractBucket = "$tdpos"/ "$xpos"
// 1. 候选人提名相关key = "nominate"
//                value = <${candi_addr}, <${from_addr}, ${ballot_count}>>
// 2. 投票候选人相关key = "vote_${candi_addr}"
//                value = <${from_addr}, ${ballot_count}>
// 3. 撤销动作相关  key = "revoke_${candi_addr}"
//                value = <${from_addr}, <(${TYPE_VOTE/TYPE_NOMINATE}, ${ballot_count})>>
// 以上所有的数据读通过快照读取, 快照读取的是当前区块的前三个区块的值
// 以上所有数据都更新到各自的链上存储中，直接走三代合约写入，去除原Finalize的最后写入更新机制
// 由于三代合约读写集限制，不能针对同一个ExeInput触发并行操作，后到的tx将会出现读写集错误，即针对同一个大key的操作同一个区块只能顺序执行
// 撤销走的是proposal合约，但目前看来proposal没有指明height

// runNominateCandidate 执行提名候选人
func (tp *tdposConsensus) runNominateCandidate(contractCtx contract.KContext) (*contract.Response, error) {
	// 1.1 核查nominate合约参数有效性
	candidateName, height, err := tp.checkArgs(contractCtx.Args())
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	amountBytes := contractCtx.Args()["amount"]
	amountStr := string(amountBytes)
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if amount <= 0 || err != nil {
		return common.NewContractErrResponse(common.StatusErr, amountErr.Error()), amountErr
	}
	// 1.2 是否按照要求多签
	if ok := tp.isAuthAddress(candidateName, contractCtx.Initiator(), contractCtx.AuthRequire()); !ok {
		return common.NewContractErrResponse(common.StatusErr, authErr.Error()), authErr
	}
	// 1.3 调用冻结接口
	tokenArgs := map[string][]byte{
		"from":      []byte(contractCtx.Initiator()),
		"amount":    []byte(fmt.Sprintf("%d", amount)),
		"lock_type": []byte(utils.GovernTokenTypeTDPOS),
	}
	_, err = contractCtx.Call("xkernel", utils.GovernTokenKernelContract, "Lock", tokenArgs)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}

	// 2. 读取提名候选人key
	nKey := fmt.Sprintf("%s_%d_%s", tp.status.Name, tp.status.Version, nominateKey)
	res, err := tp.election.getSnapshotKey(height, tp.election.bindContractBucket, []byte(nKey))
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, "Internal error."), err
	}
	nominateValue := NewNominateValue()
	if res != nil { // 非首次初始化
		if err := json.Unmarshal(res, &nominateValue); err != nil {
			tp.log.Error("tdpos::runNominateCandidate::load read set err.")
			return common.NewContractErrResponse(common.StatusErr, "Internal error."), err
		}
	}
	if _, ok := nominateValue[candidateName]; ok { // 已经提过名
		return common.NewContractErrResponse(common.StatusErr, repeatNominateErr.Error()), repeatNominateErr
	}
	record := make(map[string]int64)
	record[contractCtx.Initiator()] = amount
	nominateValue[candidateName] = record

	// 3. 候选人改写
	returnBytes, err := json.Marshal(nominateValue)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	if err := contractCtx.Put(tp.election.bindContractBucket, []byte(nKey), returnBytes); err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)
	return common.NewContractOKResponse([]byte("ok")), nil
}

// runRevokeCandidate 执行候选人撤销,仅支持自我撤销
// 重构后的候选人撤销
// Args: candidate::候选人钱包地址
func (tp *tdposConsensus) runRevokeCandidate(contractCtx contract.KContext) (*contract.Response, error) {
	// 核查撤销nominate合约参数有效性
	candidateName, height, err := tp.checkArgs(contractCtx.Args())
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	// 1. 提名候选人改写
	nKey := fmt.Sprintf("%s_%d_%s", tp.status.Name, tp.status.Version, nominateKey)
	res, err := tp.election.getSnapshotKey(height, tp.election.bindContractBucket, []byte(nKey))
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, "Internal error."), err
	}
	nominateValue := NewNominateValue()
	if res != nil {
		if err := json.Unmarshal(res, &nominateValue); err != nil {
			tp.log.Error("tdpos::runRevokeCandidate::load read set err.")
			return common.NewContractErrResponse(common.StatusErr, notFoundErr.Error()), err
		}
	}
	// 1.1 查看是否有历史投票
	v, ok := nominateValue[candidateName]
	if !ok {
		return common.NewContractErrResponse(common.StatusErr, emptyNominateKey.Error()), emptyNominateKey
	}
	ballot, ok := v[contractCtx.Initiator()]
	if !ok {
		return common.NewContractErrResponse(common.StatusErr, notFoundErr.Error()), notFoundErr
	}
	// 1.2 查询到amount之后，再调用解冻接口，Args: FromAddr, amount
	tokenArgs := map[string][]byte{
		"from":      []byte(contractCtx.Initiator()),
		"amount":    []byte(fmt.Sprintf("%d", ballot)),
		"lock_type": []byte(utils.GovernTokenTypeTDPOS),
	}
	_, err = contractCtx.Call("xkernel", utils.GovernTokenKernelContract, "UnLock", tokenArgs)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}

	// 2. 读取撤销记录
	rKey := fmt.Sprintf("%s_%d_%s", tp.status.Name, tp.status.Version, revokeKey)
	res, err = tp.election.getSnapshotKey(height, tp.election.bindContractBucket, []byte(rKey))
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, "Internal error."), err
	}
	revokeValue := NewRevokeValue()
	if res != nil {
		if err := json.Unmarshal(res, &revokeValue); err != nil {
			tp.log.Error("tdpos::runRevokeCandidate::load revoke read set err.")
			return common.NewContractErrResponse(common.StatusErr, notFoundErr.Error()), err
		}
	}

	// 3. 更改撤销记录
	if _, ok := revokeValue[contractCtx.Initiator()]; !ok {
		revokeValue[contractCtx.Initiator()] = make([]revokeItem, 0)
	}
	revokeValue[contractCtx.Initiator()] = append(revokeValue[contractCtx.Initiator()], revokeItem{
		RevokeType:    NOMINATETYPE,
		Ballot:        ballot,
		TargetAddress: candidateName,
	})
	revokeBytes, err := json.Marshal(revokeValue)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}

	// 4. 删除候选人记录
	delete(nominateValue, candidateName)
	nominateBytes, err := json.Marshal(nominateValue)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	if err := contractCtx.Put(tp.election.bindContractBucket, []byte(rKey), revokeBytes); err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	if err := contractCtx.Put(tp.election.bindContractBucket, []byte(nKey), nominateBytes); err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)
	return common.NewContractOKResponse([]byte("ok")), nil
}

// runVote 执行投票
// Args: candidate::候选人钱包地址
//       amount::投票者票数
func (tp *tdposConsensus) runVote(contractCtx contract.KContext) (*contract.Response, error) {
	// 1.1 验证合约参数是否正确
	candidateName, height, err := tp.checkArgs(contractCtx.Args())
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	amountBytes := contractCtx.Args()["amount"]
	amountStr := string(amountBytes)
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if amount <= 0 || err != nil {
		return common.NewContractErrResponse(common.StatusErr, amountErr.Error()), amountErr
	}
	// 1.2 调用冻结接口
	tokenArgs := map[string][]byte{
		"from":      []byte(contractCtx.Initiator()),
		"amount":    []byte(fmt.Sprintf("%d", amount)),
		"lock_type": []byte(utils.GovernTokenTypeTDPOS),
	}
	_, err = contractCtx.Call("xkernel", utils.GovernTokenKernelContract, "Lock", tokenArgs)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	// 1.3 检查vote的地址是否在候选人池中，快照读取候选人池，vote相关参数一定是会在nominate列表中显示
	res, err := tp.election.getSnapshotKey(height, tp.election.bindContractBucket, []byte(fmt.Sprintf("%s_%d_%s", tp.status.Name, tp.status.Version, nominateKey)))
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, "Internal error."), err
	}
	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		tp.log.Error("tdpos::runVote::load nominates read set err.")
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	if _, ok := nominateValue[candidateName]; !ok {
		return common.NewContractErrResponse(common.StatusErr, voteNominateErr.Error()), voteNominateErr
	}

	// 2. 读取投票存储
	voteKey := fmt.Sprintf("%s_%d_%s%s", tp.status.Name, tp.status.Version, voteKeyPrefix, candidateName)
	res, err = tp.election.getSnapshotKey(height, tp.election.bindContractBucket, []byte(voteKey))
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, "tdpos::Vote::get key err."), err
	}
	voteValue := NewvoteValue()
	if res != nil {
		if err := json.Unmarshal(res, &voteValue); err != nil {
			tp.log.Error("tdpos::runVote::load vote read set err.")
			return common.NewContractErrResponse(common.StatusErr, err.Error()), err
		}
	}

	// 3. 改写vote数据
	if _, ok := voteValue[contractCtx.Initiator()]; !ok {
		voteValue[contractCtx.Initiator()] = 0
	}
	voteValue[contractCtx.Initiator()] += amount
	voteBytes, err := json.Marshal(voteValue)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	if err := contractCtx.Put(tp.election.bindContractBucket, []byte(voteKey), voteBytes); err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)
	return common.NewContractOKResponse([]byte("ok")), nil
}

// runRevokeVote 执行选票撤销
// 重构后的候选人撤销
// Args: candidate::候选人钱包地址
//       amount: 投票数
func (tp *tdposConsensus) runRevokeVote(contractCtx contract.KContext) (*contract.Response, error) {
	// 1.1 验证合约参数
	candidateName, height, err := tp.checkArgs(contractCtx.Args())
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	amountBytes := contractCtx.Args()["amount"]
	amountStr := string(amountBytes)
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if amount <= 0 || err != nil {
		return common.NewContractErrResponse(common.StatusErr, amountErr.Error()), amountErr
	}
	// 1.2 调用解冻接口，Args: FromAddr, amount
	tokenArgs := map[string][]byte{
		"from":      []byte(contractCtx.Initiator()),
		"amount":    []byte(fmt.Sprintf("%d", amount)),
		"lock_type": []byte(utils.GovernTokenTypeTDPOS),
	}
	_, err = contractCtx.Call("xkernel", utils.GovernTokenKernelContract, "UnLock", tokenArgs)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	// 1.3 检查是否在vote池子里面，读取vote存储
	voteKey := fmt.Sprintf("%s_%d_%s%s", tp.status.Name, tp.status.Version, voteKeyPrefix, candidateName)
	res, err := tp.election.getSnapshotKey(height, tp.election.bindContractBucket, []byte(voteKey))
	if err != nil {
		tp.log.Error("tdpos::runRevokeVote::load vote read set err when get key.")
		return common.NewContractErrResponse(common.StatusErr, "Internal error."), err
	}
	voteValue := NewvoteValue()
	if err := json.Unmarshal(res, &voteValue); err != nil {
		tp.log.Error("tdpos::runRevokeVote::load vote read set err.")
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	v, ok := voteValue[contractCtx.Initiator()]
	if !ok {
		return common.NewContractErrResponse(common.StatusErr, notFoundErr.Error()), notFoundErr
	}
	if v < amount {
		return common.NewContractErrResponse(common.StatusErr, "Your vote amount is less than have."), emptyNominateKey
	}

	// 2. 读取撤销记录，后续改写用
	rKey := fmt.Sprintf("%s_%d_%s", tp.status.Name, tp.status.Version, revokeKey)
	res, err = tp.election.getSnapshotKey(height, tp.election.bindContractBucket, []byte(rKey))
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, "Internal error."), err
	}
	revokeValue := NewRevokeValue()
	if res != nil {
		if err := json.Unmarshal(res, &revokeValue); err != nil {
			tp.log.Error("tdpos::runRevokeCandidate::load revoke read set err.")
			return common.NewContractErrResponse(common.StatusErr, notFoundErr.Error()), err
		}
	}

	// 3. 改写撤销存储，撤销表中新增操作
	if _, ok := revokeValue[contractCtx.Initiator()]; !ok {
		revokeValue[contractCtx.Initiator()] = make([]revokeItem, 0)
	}
	revokeValue[contractCtx.Initiator()] = append(revokeValue[contractCtx.Initiator()], revokeItem{
		RevokeType:    VOTETYPE,
		Ballot:        amount,
		TargetAddress: candidateName,
	})
	revokeBytes, err := json.Marshal(revokeValue)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}

	// 4. 改写vote数据，注意，vote即使变成null也并不影响其在候选人池中，无需重写候选人池
	voteValue[contractCtx.Initiator()] -= amount
	voteBytes, err := json.Marshal(voteValue)
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	if err := contractCtx.Put(tp.election.bindContractBucket, []byte(rKey), revokeBytes); err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	if err := contractCtx.Put(tp.election.bindContractBucket, []byte(voteKey), voteBytes); err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)
	return common.NewContractOKResponse([]byte("ok")), nil
}

func (tp *tdposConsensus) checkArgs(txArgs map[string][]byte) (string, int64, error) {
	candidateBytes := txArgs["candidate"]
	candidateName := string(candidateBytes)
	if candidateName == "" {
		return "", 0, nominateAddrErr
	}
	heightBytes := txArgs["height"]
	heightStr := string(heightBytes)
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		return "", 0, notFoundErr
	}
	if height <= tp.status.StartHeight || height > tp.election.ledger.GetTipBlock().GetHeight() {
		return "", 0, errors.New("Input height invalid. Pls wait seconds.")
	}
	return candidateName, height, nil
}

type nominateValue map[string]map[string]int64

func NewNominateValue() nominateValue {
	return make(map[string]map[string]int64)
}

type voteValue map[string]int64

func NewvoteValue() voteValue {
	return make(map[string]int64)
}

type revokeValue map[string][]revokeItem

type revokeItem struct {
	RevokeType    string
	Ballot        int64
	TargetAddress string
}

func NewRevokeValue() revokeValue {
	return make(map[string][]revokeItem)
}

func (tp *tdposConsensus) isAuthAddress(candidate string, initiator string, authRequire []string) bool {
	if strings.HasSuffix(initiator, candidate) {
		return true
	}
	for _, value := range authRequire {
		if strings.HasSuffix(value, candidate) {
			return true
		}
	}
	return false
}
