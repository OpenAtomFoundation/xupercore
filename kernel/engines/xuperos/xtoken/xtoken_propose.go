package xtoken

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

// ---------- Propose and Vote ----------

func (x *Contract) Propose(ctx contract.KContext) (*contract.Response, error) {
	// 1、检查token是否存在
	args := ctx.Args()
	tokenName := string(args["tokenName"])
	topic := string(args["topic"])
	if len(topic) == 0 {
		return nil, errors.New("topic can not be empty")
	}
	data := string(args["data"])
	if len(data) == 0 {
		return nil, errors.New("data can not be empty")
	}
	token, err := x.getToken(ctx, tokenName)
	if err != nil {
		return nil, err
	}
	// 2、检查发起提案人账户余额
	bal, err := x.balanceOf(ctx, tokenName, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if bal.Cmp(token.GenesisProposal.ProposeMinToken) < 0 {
		return nil, errors.New("insufficient balance to initiate a proposal")
	}

	// 如果账户有冻结的金额，说明已有参与的进行中提案，包括自己发起的以及投票的。
	fronzen, err := x.getFrozenBalance(ctx, tokenName, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if fronzen.Cmp(big.NewInt(0)) > 0 {
		return nil, errors.New("there are already participating proposals")
	}

	// 3、更新提案数据
	pID, err := x.getLatestProposalID(ctx, tokenName, topic)
	if err != nil {
		return nil, err
	}
	if err := x.saveNewProposal(ctx, tokenName, topic, data, pID.Add(pID, big.NewInt(1))); err != nil {
		return nil, err
	}

	// 锁定发起人账户余额。
	err = x.lockProposerToken(ctx, tokenName, ctx.Initiator(), pID)
	if err != nil {
		return nil, err
	}

	type ProposalResult struct {
		ProposalID *big.Int
	}
	result := &ProposalResult{ProposalID: pID}
	value, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	err = x.addFee(ctx, Propose)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   value,
	}, nil
}

// 锁定提案者所有余额
func (x *Contract) lockProposerToken(ctx contract.KContext, token, address string, pid *big.Int) error {
	bal, err := x.balanceOf(ctx, token, address)
	if err != nil {
		return err
	}
	return x.setVotingProposalByProposer(ctx, token, address, pid, bal)
}

func (x *Contract) checkVoteOption(option string) (int, error) {
	opt, err := strconv.Atoi(option)
	if err != nil {
		return 0, errors.New("invalid vote option")
	}
	switch opt {
	case voteAgreeOption, voteOpposeOption, voteWaiveOption:
		return opt, nil
	}
	return 0, errors.New("invalid vote option")
}

func (x *Contract) Vote(ctx contract.KContext) (*contract.Response, error) {
	// 1、检查参数是否正确（token、proposal 是否存在且提案状态是否为 voting）
	// 检查投票topic下是否有已经投票的提案，如果有则不允许投票。
	args := ctx.Args()
	tokenName := string(args["tokenName"])
	topic := string(args["topic"])
	proposalID := string(args["proposalID"])
	value := string(args["value"])
	option := string(args["option"])
	opt, err := x.checkVoteOption(option)
	if err != nil {
		return nil, err
	}
	voteAmount, ok := big.NewInt(0).SetString(value, 10)
	if !ok {
		return nil, errors.New("invalid vote value")
	}
	if voteAmount.Cmp(big.NewInt(0)) <= 0 {
		return nil, errors.New("invalid vote value")
	}
	if len(tokenName) == 0 || len(topic) == 0 || len(proposalID) == 0 || len(value) == 0 {
		return nil, errors.New("invalid param")
	}
	pID, ok := big.NewInt(0).SetString(proposalID, 10)
	if !ok {
		return nil, errors.New("invalid proposalID")
	}

	p, err := x.getProposal(ctx, tokenName, topic, pID)
	if err != nil {
		return nil, err
	}
	if p.Status != ProposalVoting {
		return nil, errors.New("proposal closed")
	}

	// 2、检查发起者余额是否大于投票金额
	bal, err := x.balanceOf(ctx, tokenName, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if bal.Cmp(voteAmount) < 0 {
		return nil, errors.New("insufficient balance to vote")
	}

	// 如果账户有冻结的金额，说明已有参与的进行中提案，包括自己发起的以及投票的。
	fronzen, err := x.getFrozenBalance(ctx, tokenName, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if fronzen.Cmp(big.NewInt(0)) > 0 {
		return nil, errors.New("there are already participating proposals")
	}

	// 3、更新投票数据
	if err := x.saveVote(ctx, tokenName, topic, opt, pID, voteAmount); err != nil {
		return nil, err
	}

	// 4、更新投票人的所有voting状态的提案数据
	if err := x.addAddressVotingProposal(ctx, tokenName, ctx.Initiator(), topic, pID, voteAmount); err != nil {
		return nil, err
	}

	err = x.addFee(ctx, Vote)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (x *Contract) CheckVote(ctx contract.KContext) (*contract.Response, error) {
	ok, err := x.checkPermission(ctx, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("check permission failed")
	}

	args := ctx.Args()
	tokenName := string(args["tokenName"])
	topic := string(args["topic"])
	proposalID := string(args["proposalID"])

	// 检查参数
	if len(tokenName) == 0 || len(topic) == 0 || len(proposalID) == 0 {
		return nil, errors.New("invalid param")
	}
	pID, ok := big.NewInt(0).SetString(proposalID, 10)
	if !ok {
		return nil, errors.New("invalid proposalID")
	}

	proposal, err := x.getProposal(ctx, tokenName, topic, pID)
	if err != nil {
		return nil, err
	}
	if proposal.Status != ProposalVoting {
		return nil, errors.New("proposal closed")
	}

	// 获取所有此提案的投票数据，包括赞成、反对、弃权，
	agreeCount, err := x.getAgreeVoteAmount(ctx, tokenName, topic, pID, true)
	if err != nil {
		return nil, err
	}
	opposeCount, err := x.getOpposeVoteAmount(ctx, tokenName, topic, pID, true)
	if err != nil {
		return nil, err
	}
	waiveCount, err := x.getWaiveVoteAmount(ctx, tokenName, topic, pID, true)
	if err != nil {
		return nil, err
	}

	tmp := big.NewInt(0).Add(agreeCount, opposeCount)
	totalVote := tmp.Add(tmp, waiveCount)
	token, err := x.getToken(ctx, tokenName)
	if err != nil {
		return nil, err
	}

	// 判断所有票数总量是否大于 ProposalEffectiveAmount，否则标记提案失效
	if totalVote.Cmp(token.GenesisProposal.ProposalEffectiveAmount) < 0 {
		// 参与的总票数不足，当前提案更新为无效提案。
		proposal.Status = ProposalInvalid
	} else {
		if agreeCount.Cmp(opposeCount) > 0 {
			// 判断赞成票是否大于反对票，是则更新提案为成功，否则提案失败。
			proposal.Status = ProposalSuccess
		} else {
			proposal.Status = ProposalFailure
		}
	}

	result, err := x.setProposalResult(ctx, proposal, tokenName, agreeCount, opposeCount, waiveCount)
	if err != nil {
		return nil, err
	}

	value, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	err = x.addFee(ctx, CheckVote)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   value,
	}, nil
}

func (x *Contract) setProposalResult(ctx contract.KContext, proposal *Proposal, tokenName string, agreeCount, opposeCount, waiveCount *big.Int) (*CheckVoteResult, error) {
	if err := x.updateProposal(ctx, tokenName, proposal); err != nil {
		return nil, err
	}
	if proposal.Status == ProposalSuccess {
		// 提案成功则更新投票状态以及topic最新数据
		if err := x.setTopicData(ctx, tokenName, proposal.Topic, proposal.Data); err != nil {
			return nil, err
		}
	}

	// 更新proposer下的进行中的提案数据
	err := x.delVotingProposalByProposer(ctx, tokenName, proposal.Proposer, proposal.ID)
	if err != nil {
		return nil, err
	}

	return &CheckVoteResult{
		Status:      proposal.Status,
		AgreeCount:  agreeCount,
		OpposeCount: opposeCount,
		WaiveCount:  waiveCount,
	}, nil
}

func (x *Contract) QueryProposal(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	topic := string(ctx.Args()["topic"])
	if len(tokenName) == 0 || len(topic) == 0 {
		return nil, errors.New("tokenName and topic param can not be empty")
	}
	proposalIDvalue := string(ctx.Args()["proposalID"])
	if len(proposalIDvalue) == 0 {
		return nil, errors.New("proposalID can not be empty")
	}
	pID, ok := big.NewInt(0).SetString(proposalIDvalue, 10)
	if !ok {
		return nil, errors.New("invalid proposalID")
	}
	p, err := x.getProposal(ctx, tokenName, topic, pID)
	if err != nil {
		return nil, err
	}
	value, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	err = x.addFee(ctx, QueryProposal)
	if err != nil {
		return nil, err
	}

	return &contract.Response{
		Status: Success,
		Body:   value,
	}, nil
}

func (x *Contract) QueryProposalVotes(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	topic := string(ctx.Args()["topic"])
	if len(tokenName) == 0 || len(topic) == 0 {
		return nil, errors.New("tokenName and topic param can not be empty")
	}
	proposalIDvalue := string(ctx.Args()["proposalID"])
	if len(proposalIDvalue) == 0 {
		return nil, errors.New("proposalID can not be empty")
	}
	pID, ok := big.NewInt(0).SetString(proposalIDvalue, 10)
	if !ok {
		return nil, errors.New("invalid proposalID")
	}

	agreeCount, err := x.getAgreeVoteAmount(ctx, tokenName, topic, pID, false)
	if err != nil {
		return nil, err
	}
	opposeCount, err := x.getOpposeVoteAmount(ctx, tokenName, topic, pID, false)
	if err != nil {
		return nil, err
	}
	waiveCount, err := x.getWaiveVoteAmount(ctx, tokenName, topic, pID, false)
	if err != nil {
		return nil, err
	}

	result := &CheckVoteResult{
		AgreeCount:  agreeCount,
		OpposeCount: opposeCount,
		WaiveCount:  waiveCount,
	}
	err = x.addFee(ctx, QueryProposalVotes)
	if err != nil {
		return nil, err
	}

	value, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   value,
	}, nil
}

func (x *Contract) QueryTopic(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	topic := string(ctx.Args()["topic"])
	if len(tokenName) == 0 || len(topic) == 0 {
		return nil, errors.New("tokenName and topic param can not be empty")
	}
	key := []byte(KeyOfTopicData(tokenName, topic))
	value, err := ctx.Get(XTokenContract, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get address voting topic failed")
	}

	err = x.addFee(ctx, QueryTopic)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   value,
	}, nil
}

func (x *Contract) setTopicData(ctx contract.KContext, tokenName, topic, data string) error {
	key := []byte(KeyOfTopicData(tokenName, topic))
	return ctx.Put(XTokenContract, key, []byte(data))
}

func (x *Contract) getAddressVotingProposal(ctx contract.KContext, token, address string) (map[string]map[string]*big.Int, error) {
	key := []byte(KeyOfAddressVotingProposal(token, address))
	value, err := ctx.Get(XTokenContract, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get address voting topic failed")
	}
	proposalMap := make(map[string]map[string]*big.Int)
	if len(value) != 0 {
		if err := json.Unmarshal(value, &proposalMap); err != nil {
			return nil, errors.Wrap(err, "addAddressVotingProposal unmarshal topic map failed")
		}
	}
	return proposalMap, nil
}

// 用户的voting状态的提案列表添加一个新的提案ID，用户投票时调用此函数。
func (x *Contract) addAddressVotingProposal(ctx contract.KContext, token, address, topic string, pID, amount *big.Int) error {
	proposalMap, err := x.getAddressVotingProposal(ctx, token, address)
	if err != nil {
		return err
	}
	if pidMap, ok := proposalMap[topic]; ok {
		if pidMap == nil {
			pidMap = map[string]*big.Int{pID.String(): amount}
		} else {
			if _, ok := pidMap[pID.String()]; ok {
				return errors.New("address voting proposal ID already exist")
			}
			pidMap[pID.String()] = amount
		}
		proposalMap[topic] = pidMap
	} else {
		proposalMap[topic] = map[string]*big.Int{pID.String(): amount}
	}

	value, err := json.Marshal(proposalMap)
	if err != nil {
		return err
	}
	key := []byte(KeyOfAddressVotingProposal(token, address))
	err = ctx.Put(XTokenContract, key, value)
	if err != nil {
		return err
	}
	return nil
}

// 用户的voting状态的提案列表删除结束的提案
func (x *Contract) delAddressVotingProposal(ctx contract.KContext, token, address, topic string, pID *big.Int) error {
	proposalMap, err := x.getAddressVotingProposal(ctx, token, address)
	if err != nil {
		return err
	}
	if pidMap, ok := proposalMap[topic]; !ok {
		return nil
	} else {
		if pidMap == nil {
			// 不应有此错误，若有说明投票或者检票时数据处理有问题。
			return errors.New("address voting topic pidMap empty when delete proposal by ID")
		}
		if _, ok := pidMap[pID.String()]; !ok {
			// 不应有此错误，若有说明投票或者检票时数据处理有问题。
			return errors.New("address voting amount in pidMap empty when delete proposal by ID")
		}
		delete(pidMap, pID.String())
		if len(pidMap) == 0 {
			delete(proposalMap, topic)
		} else {
			proposalMap[topic] = pidMap
		}
	}

	value, err := json.Marshal(proposalMap)
	if err != nil {
		return err
	}
	key := []byte(KeyOfAddressVotingProposal(token, address))
	err = ctx.Put(XTokenContract, key, value)
	if err != nil {
		return err
	}
	return nil
}

func (x *Contract) getLatestProposalID(ctx contract.KContext, token, topic string) (*big.Int, error) {
	key := []byte(KeyOfLatestProposalID(token, topic))
	value, err := ctx.Get(XTokenContract, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get latest proposal ID failed")
	}
	if len(value) == 0 { // 如果没有数据，ID 从0开始。
		return big.NewInt(0), nil
	}
	latestID, ok := big.NewInt(0).SetString(string(value), 10)
	if !ok {
		return nil, errors.New("getLatestProposalID bigInt set string failed")
	}
	return latestID, nil
}

func (x *Contract) saveLatestProposalID(ctx contract.KContext, token, topic string, proposalID *big.Int) error {
	key := []byte(KeyOfLatestProposalID(token, topic))
	err := ctx.Put(XTokenContract, key, []byte(proposalID.String()))
	if err != nil {
		return err
	}
	return nil
}

func (x *Contract) saveNewProposal(ctx contract.KContext, token, topic, data string, proposalID *big.Int) error {
	p := &Proposal{
		Topic:    topic,
		ID:       proposalID,
		Data:     data,
		Proposer: ctx.Initiator(),
		Status:   ProposalVoting,
	}
	return x.saveProposal(ctx, token, p)
}

// 保存proposal以及更新最新的proposalID
func (x *Contract) saveProposal(ctx contract.KContext, token string, p *Proposal) error {
	value, err := json.Marshal(p)
	if err != nil {
		return errors.Wrap(err, "json marshal new proposal failed")
	}
	key := []byte(KeyOfProposalID(token, p.Topic, p.ID))
	err = ctx.Put(XTokenContract, key, value)
	if err != nil {
		return err
	}
	err = x.saveLatestProposalID(ctx, token, p.Topic, p.ID)
	if err != nil {
		return err
	}
	return nil
}

// 只保存proposal，不更新最新的proposalID
func (x *Contract) updateProposal(ctx contract.KContext, token string, p *Proposal) error {
	value, err := json.Marshal(p)
	if err != nil {
		return errors.Wrap(err, "json marshal new proposal failed")
	}
	key := []byte(KeyOfProposalID(token, p.Topic, p.ID))
	err = ctx.Put(XTokenContract, key, value)
	if err != nil {
		return err
	}
	return nil
}

func (x *Contract) getProposal(ctx contract.KContext, token, topic string, proposalID *big.Int) (*Proposal, error) {
	key := []byte(KeyOfProposalID(token, topic, proposalID))
	value, err := ctx.Get(XTokenContract, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get latest proposal ID failed")
	}
	if len(value) == 0 {
		return nil, errors.New("no proposal data found")
	}
	p := new(Proposal)
	if err := json.Unmarshal(value, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (x *Contract) getAgreeVoteAmount(ctx contract.KContext, token, topic string, proposalID *big.Int, delAddrVotiingProposal bool) (*big.Int, error) {
	return x.getVoteAmount(ctx, token, topic, proposalID, voteAgreeOption, delAddrVotiingProposal)
}

func (x *Contract) getOpposeVoteAmount(ctx contract.KContext, token, topic string, proposalID *big.Int, delAddrVotiingProposal bool) (*big.Int, error) {
	return x.getVoteAmount(ctx, token, topic, proposalID, voteOpposeOption, delAddrVotiingProposal)
}

func (x *Contract) getWaiveVoteAmount(ctx contract.KContext, token, topic string, proposalID *big.Int, delAddrVotiingProposal bool) (*big.Int, error) {
	return x.getVoteAmount(ctx, token, topic, proposalID, voteWaiveOption, delAddrVotiingProposal)
}

func (x *Contract) getVoteAmount(ctx contract.KContext, token, topic string, proposalID *big.Int, option int, delAddrVotiingProposal bool) (*big.Int, error) {
	start, _ := x.getVoteKeyPrefix(token, topic, option, proposalID)
	iter, err := ctx.Select(XTokenContract, []byte(start), []byte(start+"~"))
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	voteCount := big.NewInt(0)
	for iter.Next() {
		key := iter.Key()
		value := string(iter.Value())
		address := strings.TrimPrefix(string(key), start)

		voteAmount, ok := big.NewInt(0).SetString(value, 10)
		if !ok {
			// 此处不应有错误，如果有，说明投票时检查的有问题。
			return nil, errors.New("vote value invalid, address: " + address)
		}
		voteCount = voteCount.Add(voteCount, voteAmount)
		if delAddrVotiingProposal {
			// 检票时，此参数为true，查询投票时为false。
			err = x.delAddressVotingProposal(ctx, token, address, topic, proposalID)
			if err != nil {
				return nil, err
			}
		}
	}
	return voteCount, nil
}

func (x *Contract) saveVote(ctx contract.KContext, token, topic string, option int, proposalID, amount *big.Int) error {
	key, err := x.getVoteKey(token, topic, ctx.Initiator(), option, proposalID)
	if err != nil {
		return err
	}

	value, err := ctx.Get(XTokenContract, []byte(key))
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return errors.Wrap(err, "address vote failed")
	}
	if len(value) != 0 {
		// 目前一个地址只能投票一次，不允许追加投票，不允许撤销投票，不允许重新投票。
		// 后期如果此要求有变更，需要修改此处判断。
		return errors.New("this account has already voted")
	}

	if err := ctx.Put(XTokenContract, []byte(key), []byte(amount.String())); err != nil {
		return errors.Wrap(err, "save vote failed")
	}

	return nil
}

func (x *Contract) getVoteKey(token, topic, address string, option int, proposalID *big.Int) (string, error) {
	switch option {
	case voteAgreeOption:
		return KeyOfID2AddressAgreeVote(token, topic, address, proposalID), nil
	case voteOpposeOption:
		return KeyOfID2AddressOpposeVote(token, topic, address, proposalID), nil
	case voteWaiveOption:
		return KeyOfID2AddressWaiveVote(token, topic, address, proposalID), nil
	default:
		return "", errors.New("invalid vote option")
	}
}
func (x *Contract) getVoteKeyPrefix(token, topic string, option int, proposalID *big.Int) (string, error) {
	switch option {
	case voteAgreeOption:
		return KeyOfID2AddressAgreeVotePrefix(token, topic, proposalID), nil
	case voteOpposeOption:
		return KeyOfID2AddressOpposeVotePrefix(token, topic, proposalID), nil
	case voteWaiveOption:
		return KeyOfID2AddressWaiveVotePrefix(token, topic, proposalID), nil
	default:
		return "", errors.New("invalid vote option")
	}
}

// 查询地址下发起的进行中的提案。
func (x *Contract) getVotingProposalByProposer(ctx contract.KContext, token, address string) (map[string]string, error) {
	key := []byte(KeyOfProposer2Proposal(token, address))
	value, err := ctx.Get(XTokenContract, []byte(key))
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get address proposal failed")
	}

	result := make(map[string]string, 0)
	if len(value) != 0 {
		err := json.Unmarshal(value, &result)
		if err != nil {
			return nil, errors.Wrap(err, "get db data unmarshal failed")
		}

		// 此时 pid 有可能为0，因为提案结束后，会将此数据改为0，也就意味着没有进行中的提案。
		return result, nil
	}
	return result, nil
}

func (x *Contract) setVotingProposalByProposer(ctx contract.KContext, token, address string, pid, amount *big.Int) error {
	key := []byte(KeyOfProposer2Proposal(token, address))
	pid2value := map[string]string{
		pid.String(): amount.String(),
	}
	value, err := json.Marshal(pid2value)
	if err != nil {
		return err
	}
	return ctx.Put(XTokenContract, key, value)
}

func (x *Contract) delVotingProposalByProposer(ctx contract.KContext, token, address string, pid *big.Int) error {
	key := []byte(KeyOfProposer2Proposal(token, address))
	value, err := ctx.Get(XTokenContract, []byte(key))
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return errors.Wrap(err, "get address proposal failed")
	}

	result := make(map[string]string, 0)
	if len(value) != 0 {
		err := json.Unmarshal(value, &result)
		if err != nil {
			return err
		}
		_, ok := result[pid.String()]
		if ok {
			delete(result, pid.String())
			key := []byte(KeyOfProposer2Proposal(token, address))
			value, err := json.Marshal(result)
			if err != nil {
				return err
			}
			return ctx.Put(XTokenContract, key, value)
		}
	}
	return nil
}
