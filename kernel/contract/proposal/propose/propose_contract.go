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
	stopVoteHeight, err := json.Marshal(proposal.Args["stop_vote_height"])
	if err != nil {
		return nil, err
	}
	triggerArgs := make(map[string]interface{})
	triggerArgs["proposal_id"] = []byte(proposalID)
	trigger := &utils.TriggerDesc{
		Module: utils.ProposalKernelContract,
		Method: "CheckVoteResult",
		Args:   triggerArgs,
	}
	triggerBytes, err := json.Marshal(*trigger)
	if err != nil {
		return nil, err
	}
	timerArgs := make(map[string][]byte)
	timerArgs["block_height"] = stopVoteHeight
	timerArgs["trigger"] = triggerBytes
	timerArgs["proposal_id"] = []byte(proposalID)
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
	governTokenArgs["lock_type"] = []byte(utils.ProposalTypeOrdinary)
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

	// 获取提案
	proposalKey := utils.MakeProposalKey(string(proposalIDBuf))
	proposalBuf, err := ctx.Get(utils.GetProposalBucket(), []byte(proposalKey))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no proposal found")
	}
	proposal, err := t.parse(string(proposalBuf))
	if err != nil {
		return nil, fmt.Errorf("vote failed, parse proposal error")
	}

	// 比较投票状态
	if proposal.Status != utils.ProposalStatusVoting {
		return nil, fmt.Errorf("proposal status is %s,can not vote now", proposal.Status)
	}

	// 冻结一定数量的治理代币，根据提案类型冻结不同数量的代币
	from := ctx.Initiator() // 冻结账户地址
	lockAmount := amountBuf // 冻结数量
	governTokenArgs := make(map[string][]byte)
	governTokenArgs["from"] = []byte(from)
	governTokenArgs["amount"] = lockAmount
	governTokenArgs["lock_type"] = []byte(utils.ProposalTypeOrdinary)
	_, err = ctx.Call("xkernel", utils.GovernTokenKernelContract, "Lock", governTokenArgs)
	if err != nil {
		return nil, err
	}

	// 获取账户已有锁仓信息，并更新
	lockAmountCurrent := big.NewInt(0)
	lockAmountCurrent.SetString(string(lockAmount), 10)
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
	proposalBuf, err = t.unParse(proposal)
	if err != nil {
		return nil, fmt.Errorf("vote failed, unparse proposal error")
	}
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
	proposalKey := utils.MakeProposalKey(string(proposalIDBuf))
	proposalBuf, err := ctx.Get(utils.GetProposalBucket(), []byte(proposalKey))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no proposal found")
	}
	proposal, err := t.parse(string(proposalBuf))
	if err != nil {
		return nil, fmt.Errorf("vote failed, parse proposal error")
	}

	// 校验提案者身份
	if proposal.Proposer != ctx.Initiator() {
		return nil, fmt.Errorf("no authority to thaw: %s", ctx.Initiator())
	}

	// 比较投票数
	if proposal.VoteAmount.Cmp(big.NewInt(0)) == 1 {
		return nil, fmt.Errorf("some one has voted %s tickets, can not thaw now", proposal.VoteAmount.String())
	}

	// 更新proposal状态为撤销
	proposal.Status = utils.ProposalStatusCancelled
	proposalBuf, err = t.unParse(proposal)
	if err != nil {
		return nil, fmt.Errorf("vote failed, unparse proposal error")
	}
	err = ctx.Put(utils.GetProposalBucket(), []byte(proposalKey), proposalBuf)
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
	governTokenArgs["proposal_type"] = []byte(utils.ProposalTypeOrdinary)
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
	proposalKey := utils.MakeProposalKey(string(proposalIDBuf))
	proposalBuf, err := ctx.Get(utils.GetProposalBucket(), []byte(proposalKey))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no proposal found")
	}
	proposal, err := t.parse(string(proposalBuf))
	if err != nil {
		return nil, fmt.Errorf("vote failed, parse proposal error")
	}

	// 统计投票结果
	if proposal.VoteAmount.Cmp(big.NewInt(1000)) == -1 {
		proposal.Status = utils.ProposalStatusFailure
	} else {
		proposal.Status = utils.ProposalStatusSuccess
		// 增加定时任务
		timerArgs := make(map[string][]byte)
		triggerBytes, err := json.Marshal(*proposal.Trigger)
		if err != nil {
			return nil, err
		}
		timerArgs["block_height"] = []byte(strconv.FormatInt(proposal.Trigger.Height, 10))
		timerArgs["trigger"] = triggerBytes
		timerArgs["proposal_id"] = proposalIDBuf
		_, err = ctx.Call("xkernel", utils.TimerTaskKernelContract, "Add", timerArgs)
		if err != nil {
			return nil, err
		}
	}

	// 解锁提案提交时和投票锁定的治理代币
	startKey := utils.MakeProposalLockPrefix(string(proposalIDBuf))
	prefix := utils.MakeProposalLockPrefixSeparator(string(proposalIDBuf))
	endKey := utils.PrefixRange([]byte(prefix))
	iter, err := ctx.Select(utils.GetProposalBucket(), []byte(startKey), endKey)
	defer iter.Close()
	if err != nil {
		return nil, fmt.Errorf("CheckVoteResult failed, generate proposal lock key iterator error")
	}
	for iter.Next() {
		// 解锁锁仓
		account := iter.Key()[(len(prefix) + len(utils.GetProposalBucket())):]
		unLockAmount := iter.Value()

		// 撤销治理token的锁定
		governTokenArgs := make(map[string][]byte)
		governTokenArgs["from"] = account
		governTokenArgs["amount"] = unLockAmount
		governTokenArgs["lock_type"] = []byte(utils.ProposalTypeOrdinary)
		_, err = ctx.Call("xkernel", utils.GovernTokenKernelContract, "UnLock", governTokenArgs)
		if err != nil {
			continue
		}
	}

	// 保存提案
	proposalBuf, err = t.unParse(proposal)
	if err != nil {
		return nil, fmt.Errorf("check vote result failed, unparse proposal error")
	}
	err = ctx.Put(utils.GetProposalBucket(), []byte(proposalKey), proposalBuf)
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
