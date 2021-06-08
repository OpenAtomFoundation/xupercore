package propose

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
)

type KernMethod struct {
	BcName string
}

func NewKernContractMethod(bcName string) *KernMethod {
	t := &KernMethod{
		BcName: bcName,
	}
	return t
}

func (t *KernMethod) Propose(ctx contract.KContext) (*contract.Response, error) {

	// get proposal id
	proposalID, err := t.getNextProposalID(ctx)
	if err != nil {
		return nil, err
	}

	// 解析提案
	args := ctx.Args()
	proposalBuf := args["proposal"]

	// 增加提案投票统计的定时任务
	proposal, err := t.parse(string(proposalBuf))
	if err != nil {
		return nil, err
	}

	// 校验参数
	err = checkProposalArgs(proposal)
	if err != nil {
		return nil, err
	}

	stopVoteHeight := []byte(proposal.Args["stop_vote_height"].(string))
	timerArgs, err := t.makeTimerArgs(proposalID, stopVoteHeight, "CheckVoteResult")
	if err != nil {
		return nil, err
	}
	_, err = ctx.Call("xkernel", utils.TimerTaskKernelContract, "Add", timerArgs)
	if err != nil {
		return nil, err
	}

	// 冻结一定数量的治理代币，根据提案类型冻结不同数量的代币
	from := ctx.Initiator() // 冻结账户地址
	amount := "1000"        // 冻结数量
	governTokenArgs := make(map[string][]byte)
	governTokenArgs["from"] = []byte(from)
	governTokenArgs["amount"] = []byte(amount)
	governTokenArgs["lock_type"] = []byte(utils.GovernTokenTypeOrdinary)
	_, err = ctx.Call("xkernel", utils.GovernTokenKernelContract, "Lock", governTokenArgs)
	if err != nil {
		return nil, err
	}

	// 保存该提案的锁仓信息
	lockKey := utils.MakeProposalLockKey(proposalID, from)
	err = ctx.Put(utils.GetProposalBucket(), []byte(lockKey), []byte(amount))
	if err != nil {
		return nil, err
	}

	// 保存proposal id
	err = ctx.Put(utils.GetProposalBucket(), utils.GetProposalIDKey(), []byte(proposalID))
	if err != nil {
		return nil, err
	}

	// 设置初始投票数
	proposal.VoteAmount = big.NewInt(0)
	// 设置voting状态
	proposal.Status = utils.ProposalStatusVoting
	// 设置提案者
	proposal.Proposer = ctx.Initiator()

	proposalBuf, err = t.unParse(proposal)
	if err != nil {
		return nil, err
	}
	// 保存proposal
	proposalKey := utils.MakeProposalKey(proposalID)
	err = ctx.Put(utils.GetProposalBucket(), []byte(proposalKey), proposalBuf)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: 100,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    []byte(proposalID),
	}, nil
}

func (t *KernMethod) Vote(ctx contract.KContext) (*contract.Response, error) {

	args := ctx.Args()
	proposalIDBuf := args["proposal_id"]
	amountBuf := args["amount"]
	if proposalIDBuf == nil || amountBuf == nil {
		return nil, fmt.Errorf("vote failed, proposal_id or amount is nil")
	}

	// 校验数量
	lockAmount := big.NewInt(0)
	_, isAmount := lockAmount.SetString(string(amountBuf), 10)
	if !isAmount || lockAmount.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("vote failed, amount is not valid: %s", string(amountBuf))
	}

	// 获取提案
	proposal, err := t.getProposal(ctx, string(proposalIDBuf))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no proposal found, err: %v", err.Error())
	}

	// 比较投票状态
	if proposal.Status != utils.ProposalStatusVoting {
		return nil, fmt.Errorf("proposal status is %s,can not vote now", proposal.Status)
	}

	// 冻结一定数量的治理代币，根据提案类型冻结不同数量的代币
	from := ctx.Initiator() // 冻结账户地址
	governTokenArgs := make(map[string][]byte)
	governTokenArgs["from"] = []byte(from)
	governTokenArgs["amount"] = amountBuf
	governTokenArgs["lock_type"] = []byte(utils.GovernTokenTypeOrdinary)
	_, err = ctx.Call("xkernel", utils.GovernTokenKernelContract, "Lock", governTokenArgs)
	if err != nil {
		return nil, err
	}

	// 获取账户已有锁仓信息，并更新
	lockAmountCurrent := lockAmount
	lockKey := utils.MakeProposalLockKey(string(proposalIDBuf), from)
	lockAmountBuf, err := ctx.Get(utils.GetProposalBucket(), []byte(lockKey))
	if err == nil {
		lockAmountOld := big.NewInt(0)
		lockAmountOld.SetString(string(lockAmountBuf), 10)
		lockAmountCurrent = lockAmountCurrent.Add(lockAmountCurrent, lockAmountOld)
	}

	// 保存该提案的锁仓信息
	err = ctx.Put(utils.GetProposalBucket(), []byte(lockKey), []byte(lockAmountCurrent.String()))
	if err != nil {
		return nil, err
	}

	// 获取并更新提案投票数
	amount := big.NewInt(0)
	amount.SetString(string(amountBuf), 10)
	proposal.VoteAmount = proposal.VoteAmount.Add(proposal.VoteAmount, amount)
	err = t.updateProposal(ctx, string(proposalIDBuf), proposal)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: 100,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) Thaw(ctx contract.KContext) (*contract.Response, error) {

	args := ctx.Args()
	proposalIDBuf := args["proposal_id"]
	if proposalIDBuf == nil {
		return nil, fmt.Errorf("vote failed, proposal_id or amount is nil")
	}

	// 获取提案
	proposal, err := t.getProposal(ctx, string(proposalIDBuf))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no proposal found, err: %v", err.Error())
	}

	// 校验提案者身份
	if proposal.Proposer != ctx.Initiator() {
		return nil, fmt.Errorf("no authority to thaw: %s", ctx.Initiator())
	}

	// 比较投票数
	if proposal.VoteAmount.Cmp(big.NewInt(0)) == 1 {
		return nil, fmt.Errorf("some one has voted %s tickets, can not thaw now", proposal.VoteAmount.String())
	}

	// 比较投票状态
	if proposal.Status != utils.ProposalStatusVoting {
		return nil, fmt.Errorf("proposal status is %s, only a voting proposal could be thawed", proposal.Status)
	}

	// 更新proposal状态为撤销
	proposal.Status = utils.ProposalStatusCancelled
	err = t.updateProposal(ctx, string(proposalIDBuf), proposal)
	if err != nil {
		return nil, err
	}

	// 获取账户锁仓信息
	from := ctx.Initiator() // 冻结账户地址
	lockKey := utils.MakeProposalLockKey(string(proposalIDBuf), from)
	lockAmountBuf, err := ctx.Get(utils.GetProposalBucket(), []byte(lockKey))
	if err != nil {
		return nil, err
	}

	// 撤销治理token的锁定
	governTokenArgs := make(map[string][]byte)
	governTokenArgs["from"] = []byte(from)
	governTokenArgs["amount"] = lockAmountBuf
	governTokenArgs["lock_type"] = []byte(utils.GovernTokenTypeOrdinary)
	_, err = ctx.Call("xkernel", utils.GovernTokenKernelContract, "UnLock", governTokenArgs)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: 100,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) Query(ctx contract.KContext) (*contract.Response, error) {

	args := ctx.Args()
	proposalIDBuf := args["proposal_id"]
	if proposalIDBuf == nil {
		return nil, fmt.Errorf("query failed, proposal_id is nil")
	}

	// 获取提案
	proposal, err := t.getProposal(ctx, string(proposalIDBuf))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no proposal found, err: %v", err.Error())
	}

	proposalResBuf, err := json.Marshal(proposal)
	if err != nil {
		return nil, fmt.Errorf("query proposal failed, error:%s", err.Error())
	}

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    proposalResBuf,
	}, nil
}

type ProposalID struct {
	ProposalID string `json:"proposal_id"`
}

func (t *KernMethod) CheckVoteResult(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()

	// 调用权限校验
	if ctx.Caller() != utils.TimerTaskKernelContract {
		return nil, fmt.Errorf("caller %s no authority to CheckVoteResult", ctx.Caller())
	}

	proposalID := &ProposalID{}
	err := json.Unmarshal(args["args"], proposalID)
	if err != nil {
		return nil, fmt.Errorf("parse proposal id from args error")
	}

	proposalIDBuf, err := base64.StdEncoding.DecodeString(proposalID.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("parse proposal id error")
	}

	// 获取提案
	proposal, err := t.getProposal(ctx, string(proposalIDBuf))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no proposal found, err: %v", err.Error())
	}

	// 比较提案状态，只有voting状态的提案可以进行检票
	if proposal.Status != utils.ProposalStatusVoting {
		//return nil, fmt.Errorf("proposal status is %s, only a voting proposal could be checked", proposal.Status)

		// 返回nil，是个空交易
		return &contract.Response{
			Status:  utils.StatusException,
			Message: fmt.Sprintf("proposal status is %s, only a voting proposal could be checked", proposal.Status),
			Body:    nil,
		}, nil
	}

	// 获取治理代币总额，以及投票阈值
	totalSupplyRes, err := ctx.Call("xkernel", utils.GovernTokenKernelContract, "TotalSupply", nil)
	if err != nil {
		return nil, fmt.Errorf("CheckVoteResult failed, query govern token totalsupply error")
	}
	threadTickets := big.NewInt(0)
	threadTickets.SetString(string(totalSupplyRes.Body), 10)
	voteThread := big.NewInt(0)
	voteThread.SetString(proposal.Args["min_vote_percent"].(string), 10)
	threadTickets = threadTickets.Mul(threadTickets, voteThread).Div(threadTickets, big.NewInt(100))

	// 统计投票结果
	if proposal.VoteAmount.Cmp(threadTickets) == -1 {
		proposal.Status = utils.ProposalStatusRejected
	} else {
		proposal.Status = utils.ProposalStatusPassed
		// 增加定时任务，回调proposal.Trigger
		timerArgs, err := t.makeTimerArgs(string(proposalIDBuf), []byte(strconv.FormatInt(proposal.Trigger.Height, 10)), "Trigger")
		if err != nil {
			return nil, err
		}
		_, err = ctx.Call("xkernel", utils.TimerTaskKernelContract, "Add", timerArgs)
		if err != nil {
			return nil, err
		}
	}

	// 提案表决未通过，则解锁提案提交时和投票锁定的治理代币
	if proposal.Status == utils.ProposalStatusRejected {
		// 解锁提案提交时和投票锁定的治理代币
		if t.unlockGovernTokensForProposal(ctx, string(proposalIDBuf)) != nil {
			return nil, fmt.Errorf("proposal trigger failed, unlock govern token error")
		}
	}

	// 保存提案
	err = t.updateProposal(ctx, string(proposalIDBuf), proposal)
	if err != nil {
		return nil, err
	}

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) Trigger(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()

	// 调用权限校验
	if ctx.Caller() != utils.TimerTaskKernelContract {
		return nil, fmt.Errorf("caller %s no authority to Trigger", ctx.Caller())
	}

	proposalID := &ProposalID{}
	err := json.Unmarshal(args["args"], proposalID)
	if err != nil {
		return nil, fmt.Errorf("parse proposal id from args error")
	}

	proposalIDBuf, err := base64.StdEncoding.DecodeString(proposalID.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("parse proposal id error")
	}

	// 获取提案
	proposal, err := t.getProposal(ctx, string(proposalIDBuf))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no proposal found, err: %v", err.Error())
	}

	// 比较提案状态，只有passed状态的提案可以进行提案内容执行
	if proposal.Status != utils.ProposalStatusPassed {
		//return nil, fmt.Errorf("proposal status is %s, only a passed proposal could be triggered", proposal.Status)

		// 返回nil，是个空交易
		return &contract.Response{
			Status:  utils.StatusException,
			Message: fmt.Sprintf("proposal status is %s, only a passed proposal could be triggered", proposal.Status),
			Body:    nil,
		}, nil
	}

	// 执行提案trigger任务
	triggerTxArgs := make(map[string][]byte)
	triggerArgsBytes, _ := json.Marshal(proposal.Trigger.Args)
	triggerTxArgs["args"] = triggerArgsBytes
	triggerTxArgs["height"] = []byte(strconv.FormatInt(proposal.Trigger.Height, 10))
	_, err = ctx.Call(proposal.Trigger.Module, proposal.Trigger.Contract, proposal.Trigger.Method, triggerTxArgs)
	if err != nil {
		proposal.Status = utils.ProposalStatusCompletedAndFailure
	} else {
		proposal.Status = utils.ProposalStatusCompletedAndSuccess
	}

	// 解锁提案提交时和投票锁定的治理代币
	if t.unlockGovernTokensForProposal(ctx, string(proposalIDBuf)) != nil {
		return nil, fmt.Errorf("proposal trigger failed, unlock govern token error")
	}

	// 保存提案
	err = t.updateProposal(ctx, string(proposalIDBuf), proposal)
	if err != nil {
		return nil, err
	}

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) getNextProposalID(ctx contract.KContext) (string, error) {
	latestProposalID, err := ctx.Get(utils.GetProposalBucket(), utils.GetProposalIDKey())
	if err != nil {
		// 没找到，从1开始
		return big.NewInt(1).String(), nil
	} else {
		// 找到了，自增1
		proposalID := big.NewInt(0)
		proposalID.SetString(string(latestProposalID), 10)
		return proposalID.Add(proposalID, big.NewInt(1)).String(), nil
	}
}

func (t *KernMethod) unlockGovernTokensForProposal(ctx contract.KContext, proposalID string) error {
	startKey := utils.MakeProposalLockPrefix(proposalID)
	prefix := utils.MakeProposalLockPrefixSeparator(proposalID)
	endKey := utils.PrefixRange([]byte(prefix))
	iter, err := ctx.Select(utils.GetProposalBucket(), []byte(startKey), endKey)
	if err != nil {
		return fmt.Errorf("unlockGovernTokensForProposal failed, generate proposal lock key iterator error")
	}
	defer iter.Close()
	for iter.Next() {
		// 解锁锁仓
		account := iter.Key()[(len(startKey)):]
		unLockAmount := iter.Value()

		// 撤销治理token的锁定
		governTokenArgs := make(map[string][]byte)
		governTokenArgs["from"] = account
		governTokenArgs["amount"] = unLockAmount
		governTokenArgs["lock_type"] = []byte(utils.GovernTokenTypeOrdinary)
		_, err = ctx.Call("xkernel", utils.GovernTokenKernelContract, "UnLock", governTokenArgs)
		if err != nil {
			continue
		}
	}

	return nil
}

func (t *KernMethod) getProposal(ctx contract.KContext, proposalID string) (*utils.Proposal, error) {
	proposalKey := utils.MakeProposalKey(proposalID)
	proposalBuf, err := ctx.Get(utils.GetProposalBucket(), []byte(proposalKey))
	if err != nil {
		return nil, fmt.Errorf("get proposal failed, no proposal found")
	}
	proposal, err := t.parse(string(proposalBuf))
	if err != nil {
		return nil, fmt.Errorf("get proposal failed, parse proposal error")
	}

	return proposal, nil
}

func (t *KernMethod) updateProposal(ctx contract.KContext, proposalID string, proposal *utils.Proposal) error {
	proposalKey := utils.MakeProposalKey(proposalID)
	proposalBuf, err := t.unParse(proposal)
	if err != nil {
		return fmt.Errorf("update proposal failed, unparse proposal error")
	}
	err = ctx.Put(utils.GetProposalBucket(), []byte(proposalKey), proposalBuf)
	if err != nil {
		return fmt.Errorf("update proposal failed, save proposal error")
	}

	return nil
}

func (t *KernMethod) makeTimerArgs(proposalID string, triggerHeight []byte, method string) (map[string][]byte, error) {
	triggerArgs := make(map[string]interface{})
	triggerArgs["proposal_id"] = []byte(proposalID)
	trigger := &utils.TriggerDesc{
		Module:   "xkernel",
		Contract: utils.ProposalKernelContract,
		Method:   method,
		Args:     triggerArgs,
	}
	triggerBytes, err := json.Marshal(*trigger)
	if err != nil {
		return nil, fmt.Errorf("makeTimerArgs error: %v", err.Error())
	}
	timerArgs := make(map[string][]byte)
	timerArgs["block_height"] = triggerHeight
	timerArgs["trigger"] = triggerBytes

	return timerArgs, nil
}

func (t *KernMethod) parse(proposalStr string) (*utils.Proposal, error) {
	proposal, err := utils.Parse(proposalStr)
	if err != nil {
		return nil, err
	}

	return proposal, nil
}

func (t *KernMethod) unParse(proposal *utils.Proposal) ([]byte, error) {
	proposalBuf, err := utils.UnParse(proposal)
	if err != nil {
		return nil, err
	}

	return proposalBuf, nil
}

func checkProposalArgs(proposal *utils.Proposal) error {
	if proposal.Args["min_vote_percent"] == "" || proposal.Args["stop_vote_height"] == "" {
		return fmt.Errorf("no min_vote_percent or stop_vote_height found")
	}

	err := checkVoteThread(proposal.Args["min_vote_percent"].(string))
	if err != nil {
		return err
	}

	voteStopHeight, err := parseVoteStopHeight(proposal.Args["stop_vote_height"].(string))
	if err != nil {
		return err
	}

	// 判断 voteStopHeight 大于当前高度
	// todo

	// 判断 trigger.Height 大于 voteStopHeight
	if proposal.Trigger.Height != 0 {
		triggerHeight := big.NewInt(proposal.Trigger.Height)
		if triggerHeight.Cmp(voteStopHeight) != 1 {
			return fmt.Errorf("trigger_height must be bigger than stop_vote_height")
		}
	}

	return nil
}

func checkVoteThread(voteThreadStr string) error {
	voteThread := big.NewInt(0)
	_, ok := voteThread.SetString(voteThreadStr, 10)
	if !ok {
		return fmt.Errorf("min_vote_percent parse, %s", voteThreadStr)
	}
	if voteThread.Cmp(big.NewInt(100)) == 1 || voteThread.Cmp(big.NewInt(51)) == -1 {
		return fmt.Errorf("min_vote_percent err, %s", voteThread.String())
	}

	return nil
}

func parseVoteStopHeight(voteStopHeightStr string) (*big.Int, error) {
	voteStopHeight := big.NewInt(0)
	_, ok := voteStopHeight.SetString(voteStopHeightStr, 10)
	if !ok {
		return voteStopHeight, fmt.Errorf("vote_stop_height err, %s", voteStopHeightStr)
	}

	return voteStopHeight, nil
}
