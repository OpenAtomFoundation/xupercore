package tdpos

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/xuperchain/xupercore/kernel/contract"
)

// 本文件实现tdpos的原Run方法，现全部移至三代合约
// tdpos原有的9个db存储现转化为一个bucket，4个key的链上存储形式
// contractBucket = "tdpos"
// 1. 候选人提名相关key = "nominate"
//                value = <${candi_addr}, <${from_addr}, ${ballot_count}>>
// 2. 投票候选人相关key = "vote_${candi_addr}"
//                value = <${from_addr}, ${ballot_count}>
// 3. 撤销动作相关  key = "revoke_${candi_addr}"
//                value = <${from_addr}, <(${TYPE_VOTE/TYPE_NOMINATE}, ${ballot_count})>>
// 4. 钱包p2p映射  key = "urlmap"
//                value = <${candi_addr}, ${neturl}>
//
// 以上所有的数据读通过快照读取, 快照读取的是当前区块的前三个区块的值
// 以上所有数据都更新到各自的链上存储中，直接走三代合约写入，去除原Finalize的最后写入更新机制
// 由于三代合约读写集限制，不能针对同一个ExeInput触发并行操作，后到的tx将会出现读写集错误，即针对同一个大key的操作同一个区块只能顺序执行
// 撤销走的是proposal合约，但目前看来proposal没有指明height

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
	revokeType string
	ballot     int64
	timestamp  int64
}

type netURLMap map[string]string

func NewNetURLMap() netURLMap {
	return make(map[string]string)
}

func NewRevokeValue() revokeValue {
	return make(map[string][]revokeItem)
}

func NewContractErrResponse(msg string) *contract.Response {
	return &contract.Response{
		Status:  StatusErr,
		Message: msg,
	}
}

func NewContractOKResponse(msg string) *contract.Response {
	return &contract.Response{
		Status:  StatusOK,
		Message: msg,
	}
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

// checkNominateParam 查看候选人合约参数是否合法
// Args: candidate::候选人钱包地址
//       neturl::候选人url
//       amount::投票者票数
func (tp *tdposConsensus) checkNominateParam(args map[string][]byte) (map[string]string, error) {
	candidateBytes := args["candidate"]
	candidateName := string(candidateBytes)
	if candidateName == "" {
		return nil, nominateAddrErr
	}
	candidateUrlBytes := args["neturl"]
	candidateUrl := string(candidateUrlBytes)
	if candidateUrl == "" {
		return nil, nominateUrlErr
	}
	amountBytes := args["amount"]
	amount := string(amountBytes)
	if amount == "" {
		return nil, amountErr
	}
	return map[string]string{
		"candidate": candidateName,
		"neturl":    candidateUrl,
		"amount":    amount,
	}, nil
}

// runNominateCandidate 执行提名候选人
// TODO: 抵押十万分之一？
func (tp *tdposConsensus) runNominateCandidate(contractCtx contract.KContext) (*contract.Response, error) {
	// 核查nominate合约参数有效性
	txArgs := contractCtx.Args()
	args, err := tp.checkNominateParam(txArgs)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	amount, err := strconv.ParseInt(args["amount"], 10, 64)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	// 是否按照要求多签
	if ok := tp.isAuthAddress(args["candidate"], contractCtx.Initiator(), contractCtx.AuthRequire()); !ok {
		return NewContractErrResponse(authErr.Error()), authErr
	}
	// TODO: 调用冻结接口，Args: FromAddr, amount

	// 读取提名候选人key，改写；读取钱包p2p映射，改写
	// 提名候选人改写
	tipHeight := tp.election.ledger.GetTipBlock().GetHeight()
	if tipHeight < 3 {
		tp.log.Debug("tdpos::getSnapshotKey::TipHeight < 3, use init parameters.")
		return NewContractErrResponse("Cannot nominate candidators when block height < 3."), tooLowHeight
	}
	res, err := tp.election.getSnapshotKey(tipHeight, contractBucket, []byte(nominateKey))
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		tp.log.Error("tdpos::runNominateCandidate::load read set err.")
		return NewContractErrResponse("Internal error."), err
	}
	res, err = tp.election.getSnapshotKey(tipHeight, contractBucket, []byte(urlmapKey))
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	netURLValue := NewNetURLMap()
	if err := json.Unmarshal(res, &netURLValue); err != nil {
		tp.log.Error("tdpos::runNominateCandidate::load neturl read set err.")
		return NewContractErrResponse("Internal error."), err
	}
	// 已经提过名
	if _, ok := nominateValue[args["candidate"]]; ok {
		return NewContractErrResponse(repeatNominateErr.Error()), repeatNominateErr
	}
	record := make(map[string]int64)
	record[contractCtx.Initiator()] = amount
	nominateValue[args["candidate"]] = record

	// 候选人改写
	returnBytes, err := json.Marshal(nominateValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(nominateKey), returnBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	// urlmap改写
	netURLValue[args["candidate"]] = args["neturl"]
	urlBytes, err := json.Marshal(netURLValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(urlmapKey), urlBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	return NewContractOKResponse("ok"), nil
}

// runRevokeCandidate 执行候选人撤销,仅支持自我撤销
// 重构后的候选人撤销
// Args: candidate::候选人钱包地址
func (tp *tdposConsensus) runRevokeCandidate(contractCtx contract.KContext) (*contract.Response, error) {
	// 核查撤销nominate合约参数有效性
	txArgs := contractCtx.Args()
	candidateBytes := txArgs["candidate"]
	candidateName := string(candidateBytes)
	if candidateName == "" {
		return NewContractErrResponse(nominateAddrErr.Error()), nominateAddrErr
	}
	// TODO: 调用解冻接口，Args: FromAddr, amount

	// 读取提名候选人key，改写；读取钱包p2p映射，改写
	// 提名候选人改写
	tipHeight := tp.election.ledger.GetTipBlock().GetHeight()
	if tipHeight < 3 {
		tp.log.Debug("tdpos::getSnapshotKey::TipHeight < 3, use init parameters.")
		return NewContractErrResponse("Cannot revoke candidators when block height < 3."), tooLowHeight
	}
	res, err := tp.election.getSnapshotKey(tipHeight, contractBucket, []byte(nominateKey))
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		tp.log.Error("tdpos::runRevokeCandidate::load read set err.")
		return NewContractErrResponse("Internal error."), err
	}
	// 查看是否有历史投票
	v, ok := nominateValue[candidateName]
	if !ok {
		return NewContractErrResponse(emptyNominateKey.Error()), emptyNominateKey
	}
	ballot, ok := v[contractCtx.Initiator()]
	if !ok {
		return NewContractErrResponse(notFoundErr.Error()), notFoundErr
	}
	// 读取撤销记录
	revokeKey := revokeKeyPrefix + candidateName
	res, err = tp.election.getSnapshotKey(tipHeight, contractBucket, []byte(revokeKey))
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	revokeValue := NewRevokeValue()
	if err := json.Unmarshal(res, &revokeValue); err != nil {
		tp.log.Error("tdpos::runRevokeCandidate::load revoke read set err.")
		return NewContractErrResponse("Internal error."), err
	}
	// 改写数据前操作
	// 1. 更改撤销记录
	if _, ok := revokeValue[contractCtx.Initiator()]; !ok {
		var itemSli []revokeItem
		revokeValue[contractCtx.Initiator()] = itemSli
	}
	revokeValue[contractCtx.Initiator()] = append(revokeValue[contractCtx.Initiator()], revokeItem{
		revokeType: NOMINATETYPE,
		ballot:     ballot,
		timestamp:  time.Now().UnixNano(),
	})
	revokeBytes, err := json.Marshal(revokeValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	// 2. 删除候选人记录
	delete(nominateValue, candidateName)
	nominateBytes, err := json.Marshal(nominateValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(nominateKey), nominateBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(revokeKey), revokeBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	return NewContractOKResponse("ok"), nil
}

// runVote 执行投票
// Args: candidate::候选人钱包地址
//       amount::投票者票数
func (tp *tdposConsensus) runVote(contractCtx contract.KContext) (*contract.Response, error) {
	txArgs := contractCtx.Args()
	candidateBytes := txArgs["candidate"]
	candidateName := string(candidateBytes)
	if candidateName == "" {
		return NewContractErrResponse(nominateAddrErr.Error()), nominateAddrErr
	}
	amountBytes := txArgs["amount"]
	amountStr := string(amountBytes)
	if amountStr == "" {
		return NewContractErrResponse(amountErr.Error()), amountErr
	}
	var amountBig big.Int
	amount := amountBig.SetBytes(amountBytes).Int64()
	// TODO: 调用冻结接口，Args: FromAddr, amount

	// 读取候选人投票key，改写
	// 首先检查vote的地址是否在候选人池中，快照读取候选人池
	tipHeight := tp.election.ledger.GetTipBlock().GetHeight()
	if tipHeight < 3 {
		tp.log.Debug("tdpos::getSnapshotKey::TipHeight < 3, use init parameters.")
		return NewContractErrResponse("Cannot vote candidators when block height < 3."), tooLowHeight
	}
	res, err := tp.election.getSnapshotKey(tipHeight, contractBucket, []byte(nominateKey))
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		tp.log.Error("tdpos::runVote::load nominates read set err.")
		return NewContractErrResponse(err.Error()), err
	}
	if _, ok := nominateValue[candidateName]; !ok {
		return NewContractErrResponse(voteNominateErr.Error()), voteNominateErr
	}
	// 读取投票存储
	voteKey := voteKeyPrefix + candidateName
	res, err = tp.election.getSnapshotKey(tipHeight, contractBucket, []byte(voteKey))
	if err != nil {
		tp.log.Error("tdpos::runVote::load vote read set err when get key.")
		return NewContractErrResponse("Internal error."), err
	}
	voteValue := NewvoteValue()
	if err := json.Unmarshal(res, &voteValue); err != nil {
		tp.log.Error("tdpos::runVote::load vote read set err.")
		return NewContractErrResponse(err.Error()), err
	}
	// 改写vote数据
	if _, ok := voteValue[contractCtx.Initiator()]; !ok {
		voteValue[contractCtx.Initiator()] = 0
	}
	voteValue[contractCtx.Initiator()] += amount
	// 改写前操作
	voteBytes, err := json.Marshal(voteValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(voteKey), voteBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	return NewContractOKResponse("ok"), nil
}

// runRevokeVote 执行选票撤销
// 重构后的候选人撤销
// Args: candidate::候选人钱包地址
//       amount: 投票数
func (tp *tdposConsensus) runRevokeVote(contractCtx contract.KContext) (*contract.Response, error) {
	txArgs := contractCtx.Args()
	candidateBytes := txArgs["candidate"]
	candidateName := string(candidateBytes)
	if candidateName == "" {
		return NewContractErrResponse(nominateAddrErr.Error()), nominateAddrErr
	}
	amountBytes := txArgs["amount"]
	amountStr := string(amountBytes)
	if amountStr == "" {
		return NewContractErrResponse(amountErr.Error()), amountErr
	}
	var amountBig big.Int
	amount := amountBig.SetBytes(amountBytes).Int64()
	// TODO: 调用解冻接口，Args: FromAddr, amount

	// 首先检查是否在vote池子里面，读取候选人存储
	tipHeight := tp.election.ledger.GetTipBlock().GetHeight()
	if tipHeight < 3 {
		tp.log.Debug("tdpos::getSnapshotKey::TipHeight < 3, use init parameters.")
		return NewContractErrResponse("Cannot revoke vote when block height < 3."), tooLowHeight
	}
	voteKey := voteKeyPrefix + candidateName
	res, err := tp.election.getSnapshotKey(tipHeight, contractBucket, []byte(voteKey))
	if err != nil {
		tp.log.Error("tdpos::runRevokeVote::load vote read set err when get key.")
		return NewContractErrResponse("Internal error."), err
	}
	voteValue := NewvoteValue()
	if err := json.Unmarshal(res, &voteValue); err != nil {
		tp.log.Error("tdpos::runRevokeVote::load vote read set err.")
		return NewContractErrResponse(err.Error()), err
	}
	v, ok := voteValue[candidateName]
	if !ok {
		return NewContractErrResponse(emptyNominateKey.Error()), emptyNominateKey
	}
	if v < amount {
		return NewContractErrResponse("Your vote amount is less than have."), emptyNominateKey
	}
	// 改写数据，注意，vote即使变成null也并不影响其在候选人池中，无需重写候选人池
	voteValue[candidateName] -= amount
	voteBytes, err := json.Marshal(voteValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(voteKey), voteBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	return NewContractOKResponse("ok"), nil
}
