package propose

import (
	"encoding/json"
	"fmt"
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

	// 增加定时任务
	proposal, err := t.parse(string(proposalBuf))
	if err != nil {
		return nil, err
	}

	fmt.Println("proposalAndTriger,", *proposal.Trigger)
	timerArgs := make(map[string][]byte)
	triggerBytes, err := json.Marshal(*proposal.Trigger)
	if err != nil {
		return nil, err
	}
	timerArgs["block_height"] = []byte(strconv.FormatInt(proposal.Trigger.Height, 10))
	timerArgs["trigger"] = triggerBytes
	timerArgs["proposal_id"] = []byte(proposalID)
	_, err = ctx.Call("xkernel", "$timer_task", "Add", timerArgs)
	if err != nil {
		return nil, err
	}

	// 冻结一定数量的治理代币，根据提案类型冻结不同数量的代币
	from := ctx.Initiator()                     // 冻结账户地址
	amount := "1000"                            // 冻结数量
	proposalType := utils.ProposalTypeConsensus // 提案类型
	period := "10000"                           // 冻结区块高度
	governTokenArgs := make(map[string][]byte)
	governTokenArgs["from"] = []byte(from)
	governTokenArgs["amount"] = []byte(amount)
	governTokenArgs["period"] = []byte(period)
	governTokenArgs["lock_id"] = []byte(proposalID)
	governTokenArgs["lock_type"] = []byte(proposalType)
	_, err = ctx.Call("xkernel", "$govern_token", "Lock", governTokenArgs)
	if err != nil {
		return nil, err
	}

	// 保存proposal id
	err = ctx.Put(utils.GetProposalBucket(), utils.GetProposalIDKey(), []byte(proposalID))
	if err != nil {
		return nil, err
	}

	// 设置初始投票数
	voteKey := utils.MakeProposalVoteKey(proposalID)
	err = ctx.Put(utils.GetProposalBucket(), []byte(voteKey), []byte("0"))
	if err != nil {
		return nil, err
	}

	// 保存proposal
	key := utils.MakeProposalKey(proposalID)
	err = ctx.Put(utils.GetProposalBucket(), []byte(key), proposalBuf)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: 0,
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
	if proposal.Args["stop_vote_height"] == nil {
		return nil, fmt.Errorf("%s not found in contract args", "stop_vote_height")
	}
	stopVoteHeight := proposal.Args["stop_vote_height"].(int64)

	// 比较投票截止时
	ledgerHeight := int64(0)
	if ledgerHeight > stopVoteHeight {
		return nil, fmt.Errorf("this propposal is expired for voting, %d > %d", ledgerHeight, stopVoteHeight)
	}

	// 冻结一定数量的治理代币，根据提案类型冻结不同数量的代币
	from := ctx.Initiator()                     // 冻结账户地址
	lockAmount := amountBuf                     // 冻结数量
	proposalType := utils.ProposalTypeConsensus // 提案类型
	period := "10000"                           // 冻结区块高度
	governTokenArgs := make(map[string][]byte)
	governTokenArgs["from"] = []byte(from)
	governTokenArgs["amount"] = lockAmount
	governTokenArgs["period"] = []byte(period)
	governTokenArgs["lock_id"] = proposalIDBuf
	governTokenArgs["lock_type"] = []byte(proposalType)
	_, err = ctx.Call("xkernel", "$govern_token", "Lock", governTokenArgs)
	if err != nil {
		return nil, err
	}

	// 获取并更新提案投票数
	voteKey := utils.MakeProposalVoteKey(string(proposalIDBuf))
	voteBuf, err := ctx.Get(utils.GetProposalBucket(), []byte(voteKey))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no vote result found")
	}
	voteTmp, _ := strconv.Atoi(string(voteBuf))
	voteUint64 := uint64(voteTmp)
	amountTmp, _ := strconv.Atoi(string(amountBuf))
	amountUint64 := uint64(amountTmp)
	voteCurrent := voteUint64 + amountUint64
	voteCurrentStr := strconv.FormatUint(voteCurrent, 10)
	err = ctx.Put(utils.GetProposalBucket(), []byte(voteKey), []byte(voteCurrentStr))
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: 0,
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
	_, err := ctx.Get(utils.GetProposalBucket(), []byte(proposalKey))
	if err != nil {
		return nil, fmt.Errorf("vote failed, no proposal found")
	}
	//proposal, err := t.parse(string(proposalBuf))
	//if err != nil {
	//	return nil, fmt.Errorf("vote failed, parse proposal error")
	//}

	// 撤销治理token的锁定
	from := ctx.Initiator() // 冻结账户地址
	//proposalType := ProposalTypeConsensus // 提案类型
	governTokenArgs := make(map[string][]byte)
	governTokenArgs["from"] = []byte(from)
	governTokenArgs["proposal_id"] = proposalIDBuf
	//governTokenArgs["proposal_type"] = []byte(proposalType)
	_, err = ctx.Call("xkernel", "$govern_token", "UnLock", governTokenArgs)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: 0,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  utils.StatusOK,
		Message: "success",
		Body:    nil,
	}, nil
}

func (t *KernMethod) IsVoteOK(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	proposalIDBuf := args["proposal_id"]
	if proposalIDBuf == nil {
		return nil, fmt.Errorf("query proposal status failed, proposal_id is nil")
	}

	voteKey := utils.MakeProposalVoteKey(string(proposalIDBuf))
	voteBuf, err := ctx.Get(utils.GetProposalBucket(), []byte(voteKey))
	if err != nil {
		return nil, fmt.Errorf("query proposal status failed, no vote result found")
	}

	voteTmp, _ := strconv.Atoi(string(voteBuf))
	voteUint64 := uint64(voteTmp)
	if voteUint64 < 10000 {
		return nil, fmt.Errorf("proposal not ok")
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
		return "1", nil
	} else {
		// 找到了，自增1
		proposalIDInt64, _ := strconv.Atoi(string(latestProposalID))
		proposalIDUint64 := uint64(proposalIDInt64)
		return strconv.FormatUint(proposalIDUint64+1, 10), nil
	}
}

// IsPropose return true if tx has Propose method
func (t *KernMethod) parse(proposalStr string) (*utils.Proposal, error) {
	proposal, err := utils.Parse(proposalStr)
	if err != nil {
		return nil, err
	}

	return proposal, nil
}
