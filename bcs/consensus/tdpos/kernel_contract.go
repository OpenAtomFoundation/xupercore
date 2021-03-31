package tdpos

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"

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
// 4. term存储相关，仅在vote操作后上报 key = "term"
//                               value = <${term}, ${Validators}, ${height}> (仅记录变化值)
// 以上所有的数据读通过快照读取, 快照读取的是当前区块的前三个区块的值
// 以上所有数据都更新到各自的链上存储中，直接走三代合约写入，去除原Finalize的最后写入更新机制
// 由于三代合约读写集限制，不能针对同一个ExeInput触发并行操作，后到的tx将会出现读写集错误，即针对同一个大key的操作同一个区块只能顺序执行
// 撤销走的是proposal合约，但目前看来proposal没有指明height

// runNominateCandidate 执行提名候选人
func (tp *tdposConsensus) runNominateCandidate(contractCtx contract.KContext) (*contract.Response, error) {
	// 核查nominate合约参数有效性
	txArgs := contractCtx.Args()
	candidateBytes := txArgs["candidate"]
	candidateName := string(candidateBytes)
	if candidateName == "" {
		return NewContractErrResponse(nominateAddrErr.Error()), nominateAddrErr
	}
	amountBytes := txArgs["amount"]
	amountStr := string(amountBytes)
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if amount <= 0 || err != nil {
		return NewContractErrResponse(amountErr.Error()), amountErr
	}
	// 是否按照要求多签
	if ok := tp.isAuthAddress(candidateName, contractCtx.Initiator(), contractCtx.AuthRequire()); !ok {
		return NewContractErrResponse(authErr.Error()), authErr
	}

	// 调用冻结接口
	tokenArgs := map[string][]byte{
		"from":      []byte(contractCtx.Initiator()),
		"amount":    []byte(fmt.Sprintf("%d", amount)),
		"lock_type": []byte(utils.GovernTokenTypeTDPOS),
	}
	_, err = contractCtx.Call("xkernel", utils.GovernTokenKernelContract, "Lock", tokenArgs)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}

	// 1. 读取提名候选人key，改写
	tipHeight := tp.election.ledger.GetTipBlock().GetHeight()
	if tipHeight <= 3 {
		return NewContractErrResponse("Cannot nominate candidators when block height <= 3."), tooLowHeight
	}
	res, err := tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(nominateKey))
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	nominateValue := NewNominateValue()
	if res != nil { // 非首次初始化
		if err := json.Unmarshal(res, &nominateValue); err != nil {
			tp.log.Error("tdpos::runNominateCandidate::load read set err.")
			return NewContractErrResponse("Internal error."), err
		}
	}
	if _, ok := nominateValue[candidateName]; ok { // 已经提过名
		return NewContractErrResponse(repeatNominateErr.Error()), repeatNominateErr
	}
	record := make(map[string]int64)
	record[contractCtx.Initiator()] = amount
	nominateValue[candidateName] = record

	// 2. 候选人改写
	returnBytes, err := json.Marshal(nominateValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(nominateKey), returnBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)
	return NewContractOKResponse([]byte("ok")), nil
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

	// 1. 提名候选人改写
	tipHeight := tp.election.ledger.GetTipBlock().GetHeight()
	if tipHeight <= 3 {
		tp.log.Debug("tdpos::getSnapshotKey::TipHeight <= 3, use init parameters.")
		return NewContractErrResponse("Cannot revoke candidators when block height < 3."), tooLowHeight
	}
	res, err := tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(nominateKey))
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	nominateValue := NewNominateValue()
	if res != nil {
		if err := json.Unmarshal(res, &nominateValue); err != nil {
			tp.log.Error("tdpos::runRevokeCandidate::load read set err.")
			return NewContractErrResponse(notFoundErr.Error()), err
		}
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

	// 查询到amount之后，再调用解冻接口，Args: FromAddr, amount
	tokenArgs := map[string][]byte{
		"from":      []byte(contractCtx.Initiator()),
		"amount":    []byte(fmt.Sprintf("%d", ballot)),
		"lock_type": []byte(utils.GovernTokenTypeTDPOS),
	}
	_, err = contractCtx.Call("xkernel", utils.GovernTokenKernelContract, "UnLock", tokenArgs)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}

	// 2. 读取撤销记录
	res, err = tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(revokeKey))
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	revokeValue := NewRevokeValue()
	if res != nil {
		if err := json.Unmarshal(res, &revokeValue); err != nil {
			tp.log.Error("tdpos::runRevokeCandidate::load revoke read set err.")
			return NewContractErrResponse(notFoundErr.Error()), err
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
		return NewContractErrResponse(err.Error()), err
	}

	// 4. 删除候选人记录
	delete(nominateValue, candidateName)
	nominateBytes, err := json.Marshal(nominateValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(revokeKey), revokeBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(nominateKey), nominateBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)
	return NewContractOKResponse([]byte("ok")), nil
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
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if amount <= 0 || err != nil {
		return NewContractErrResponse(amountErr.Error()), amountErr
	}

	// 调用冻结接口
	tokenArgs := map[string][]byte{
		"from":      []byte(contractCtx.Initiator()),
		"amount":    []byte(fmt.Sprintf("%d", amount)),
		"lock_type": []byte(utils.GovernTokenTypeTDPOS),
	}
	_, err = contractCtx.Call("xkernel", utils.GovernTokenKernelContract, "Lock", tokenArgs)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}

	// 1. 检查vote的地址是否在候选人池中，快照读取候选人池，vote相关参数一定是会在nominate列表中显示
	tipHeight := tp.election.ledger.GetTipBlock().GetHeight()
	if tipHeight <= 3 {
		tp.log.Debug("tdpos::getSnapshotKey::TipHeight <= 3, use init parameters.")
		return NewContractErrResponse("Cannot vote candidators when block height < 3."), tooLowHeight
	}
	res, err := tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(nominateKey))
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

	// 2. 读取投票存储
	voteKey := voteKeyPrefix + candidateName
	res, err = tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(voteKey))
	if err != nil {
		return NewContractErrResponse("tdpos::Vote::get key err."), err
	}
	voteValue := NewvoteValue()
	if res != nil {
		if err := json.Unmarshal(res, &voteValue); err != nil {
			tp.log.Error("tdpos::runVote::load vote read set err.")
			return NewContractErrResponse(err.Error()), err
		}
	}

	// 3. 改写vote数据
	if _, ok := voteValue[contractCtx.Initiator()]; !ok {
		voteValue[contractCtx.Initiator()] = 0
	}
	voteValue[contractCtx.Initiator()] += amount
	voteBytes, err := json.Marshal(voteValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(voteKey), voteBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}

	// 4. 顺带更新term索引
	if err := tp.refreshTerm(tipHeight, contractCtx); err != nil {
		return NewContractErrResponse(err.Error()), err
	}

	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)
	return NewContractOKResponse([]byte("ok")), nil
}

// refreshTerm 被动触发更新term列表，当且仅当当前候选人集合和
func (tp *tdposConsensus) refreshTerm(tipHeight int64, contractCtx contract.KContext) error {
	// 此处不并需要tipheight-3
	res, err := tp.election.getSnapshotKey(tipHeight, contractBucket, []byte(termKey))
	if err != nil {
		return err
	}
	item := termItem{
		Validators: tp.election.validators,
		Term:       tp.election.curTerm,
		Height:     tipHeight,
	}
	termValue := NewTermValue()
	// 初始化则直接push值
	if res == nil {
		termValue = append(termValue, item)
		termBytes, err := json.Marshal(termValue)
		if err != nil {
			return err
		}
		if err := contractCtx.Put(contractBucket, []byte(termKey), termBytes); err != nil {
			return err
		}
		return nil
	}
	if err := json.Unmarshal(res, &termValue); err != nil {
		return err
	}
	// 仅与最新的值对比，若有变化则插入新值
	tail := termValue[len(termValue)-1]
	if !common.AddressEqual(tail.Validators, tp.election.validators) {
		termValue = append(termValue, item)
	}
	termBytes, err := json.Marshal(termValue)
	if err != nil {
		return err
	}
	if err := contractCtx.Put(contractBucket, []byte(termKey), termBytes); err != nil {
		return err
	}
	return nil
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
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if amount <= 0 || err != nil {
		return NewContractErrResponse(amountErr.Error()), amountErr
	}

	// 调用解冻接口，Args: FromAddr, amount
	tokenArgs := map[string][]byte{
		"from":      []byte(contractCtx.Initiator()),
		"amount":    []byte(fmt.Sprintf("%d", amount)),
		"lock_type": []byte(utils.GovernTokenTypeTDPOS),
	}
	_, err = contractCtx.Call("xkernel", utils.GovernTokenKernelContract, "UnLock", tokenArgs)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}

	// 1. 检查是否在vote池子里面，读取vote存储
	tipHeight := tp.election.ledger.GetTipBlock().GetHeight()
	if tipHeight <= 3 {
		tp.log.Debug("tdpos::getSnapshotKey::TipHeight <= 3, use init parameters.")
		return NewContractErrResponse("Cannot revoke vote when block height < 3."), tooLowHeight
	}
	voteKey := voteKeyPrefix + candidateName
	res, err := tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(voteKey))
	if err != nil {
		tp.log.Error("tdpos::runRevokeVote::load vote read set err when get key.")
		return NewContractErrResponse("Internal error."), err
	}
	voteValue := NewvoteValue()
	if err := json.Unmarshal(res, &voteValue); err != nil {
		tp.log.Error("tdpos::runRevokeVote::load vote read set err.")
		return NewContractErrResponse(err.Error()), err
	}
	v, ok := voteValue[contractCtx.Initiator()]
	if !ok {
		return NewContractErrResponse(notFoundErr.Error()), notFoundErr
	}
	if v < amount {
		return NewContractErrResponse("Your vote amount is less than have."), emptyNominateKey
	}

	// 2. 读取撤销记录，后续改写用
	res, err = tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(revokeKey))
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	revokeValue := NewRevokeValue()
	if res != nil {
		if err := json.Unmarshal(res, &revokeValue); err != nil {
			tp.log.Error("tdpos::runRevokeCandidate::load revoke read set err.")
			return NewContractErrResponse(notFoundErr.Error()), err
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
		return NewContractErrResponse(err.Error()), err
	}

	// 4. 改写vote数据，注意，vote即使变成null也并不影响其在候选人池中，无需重写候选人池
	voteValue[contractCtx.Initiator()] -= amount
	voteBytes, err := json.Marshal(voteValue)
	if err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(revokeKey), revokeBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(voteKey), voteBytes); err != nil {
		return NewContractErrResponse(err.Error()), err
	}
	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)
	return NewContractOKResponse([]byte("ok")), nil
}

// runGetTdposInfos 读接口
func (tp *tdposConsensus) runGetTdposInfos(contractCtx contract.KContext) (*contract.Response, error) {
	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)

	initValue := `{
		"validators": ` + fmt.Sprintf("%s", tp.election.initValidators) + ` 
	}`
	tipHeight := tp.election.ledger.GetTipBlock().GetHeight()
	if tipHeight <= 3 {
		return NewContractOKResponse([]byte(initValue)), nil
	}

	// nominate信息
	res, err := tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(nominateKey))
	if res == nil {
		return NewContractOKResponse([]byte(initValue)), nil
	}
	if err != nil {
		return NewContractErrResponse("Internal error."), err
	}
	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		return NewContractErrResponse("tdpos::GetTdposInfos::load nominates read set err."), err
	}
	tp.log.Debug("tdpos::GetTdposInfos", "nominateValue", nominateValue)

	// vote信息
	voteMap := make(map[string]voteValue)
	for candidate, _ := range nominateValue {
		voteKey := voteKeyPrefix + candidate // 读取投票存储
		res, err = tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(voteKey))
		if err != nil {
			tp.log.Error("tdpos::GetTdposInfos::load vote read set err when get key.", "key", voteKey)
			continue
		}
		voteValue := NewvoteValue()
		if res == nil {
			continue
		}
		if err := json.Unmarshal(res, &voteValue); err != nil {
			tp.log.Error("tdpos::GetTdposInfos::load vote read set err.", "res", res, "err", err)
			continue
		}
		voteMap[candidate] = voteValue
	}
	tp.log.Debug("tdpos::GetTdposInfos", "voteMap", voteMap)

	// revoke信息
	res, err = tp.election.getSnapshotKey(tipHeight-3, contractBucket, []byte(revokeKey))
	if err != nil {
		tp.log.Error("tdpos::GetTdposInfos::load revoke read set err when get key.", "key", revokeKey)
		return NewContractErrResponse("load revoke mem error."), err
	}
	revokeValue := NewRevokeValue()
	if res != nil {
		if err := json.Unmarshal(res, &revokeValue); err != nil {
			tp.log.Error("tdpos::GetTdposInfos::load revoke read set err.", "res", res, "err", err)
			return NewContractErrResponse("load revoke mem error."), err
		}
	}
	tp.log.Debug("tdpos::GetTdposInfos", "revokeValue", revokeValue)

	r := `{
		"validators": ` + fmt.Sprintf("%s", tp.election.validators) + `,
		"nominate":` + fmt.Sprintf("%v", nominateValue) + `,
		"vote":` + fmt.Sprintf("%v", voteMap) + `,
		"revoke":` + fmt.Sprintf("%v", revokeValue) + `
	}`
	return NewContractOKResponse([]byte(r)), nil
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

type termValue []*termItem

type termItem struct {
	Validators []string
	Term       int64
	Height     int64
}

func NewTermValue() []termItem {
	return make([]termItem, 0)
}

func NewContractErrResponse(msg string) *contract.Response {
	return &contract.Response{
		Status:  StatusErr,
		Message: msg,
	}
}

func NewContractOKResponse(json []byte) *contract.Response {
	return &contract.Response{
		Status:  StatusOK,
		Message: "success",
		Body:    json,
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
