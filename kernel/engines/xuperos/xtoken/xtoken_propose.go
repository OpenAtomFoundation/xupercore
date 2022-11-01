package xtoken

import (
	"encoding/json"
	"math/big"
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

	// 3、更新提案数据
	pID, err := x.getLatestProposalID(ctx, tokenName, topic)
	if err != nil {
		return nil, err
	}
	if err := x.saveNewProposal(ctx, tokenName, topic, data, pID.Add(pID, big.NewInt(1))); err != nil {
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

func (x *Contract) Vote(ctx contract.KContext) (*contract.Response, error) {
	// 1、检查参数是否正确（token、proposal 是否存在且提案状态是否为 voting）
	args := ctx.Args()
	tokenName := string(args["tokenName"])
	topic := string(args["topic"])
	proposalID := string(args["proposalID"])
	value := string(args["value"])
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

	// 3、更新投票数据
	if err := x.saveVote(ctx, tokenName, topic, pID, voteAmount); err != nil {
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

	// 1、获取提案数据
	token, err := x.getToken(ctx, tokenName)
	if err != nil {
		return nil, err
	}
	rate := big.NewInt(int64(token.GenesisProposal.FavourRate))
	a := big.NewInt(0).Mul(rate, token.TotalSupply)
	minPassAmount := big.NewInt(0).Div(a, big.NewInt(100))
	proposal, err := x.getProposal(ctx, tokenName, topic, pID)
	if err != nil {
		return nil, err
	}
	// 2、遍历所有投票地址计算总投票金额
	start := KeyOfID2AddressVotePrefix(tokenName, topic, pID)
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
		// 3、去除不记投票地址
		if _, ok := token.GenesisProposal.ExcludeCheckVoteAddress[address]; ok {
			continue // 不统计不参与计票的地址的投票
		}
		voteAmount, ok := big.NewInt(0).SetString(value, 10)
		if !ok {
			// 此处不应有错误，如果有，说明投票时检查的有问题。
			return nil, errors.New("vote value invalid, address: " + address)
		}
		voteCount = voteCount.Add(voteCount, voteAmount)

		// 4、更新每个地址下相关的voting状态的提案相关数据
		err = x.delAddressVotingProposal(ctx, tokenName, address, topic, pID)
		if err != nil {
			return nil, err
		}
	}

	// 5、更新投票状态以及topic最新数据
	if voteCount.Cmp(minPassAmount) >= 0 {
		// 提案通过
		proposal.Status = ProposalSuccess
		if err := x.setTopicData(ctx, tokenName, topic, proposal.Data); err != nil {
			return nil, err
		}
	} else {
		// 提案失败
		proposal.Status = Proposalfailure
	}
	if err := x.saveProposal(ctx, tokenName, proposal); err != nil {
		return nil, err
	}

	type CheckVoteResult struct {
		Status int
	}

	result := &CheckVoteResult{Status: proposal.Status}
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
	start := KeyOfID2AddressVotePrefix(tokenName, topic, pID)
	iter, err := ctx.Select(XTokenContract, []byte(start), []byte(start+"~"))
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	token, err := x.getToken(ctx, tokenName)
	if err != nil {
		return nil, err
	}

	voteCount := big.NewInt(0)
	for iter.Next() {
		key := iter.Key()
		value := string(iter.Value())
		address := strings.TrimPrefix(string(key), start)
		if _, ok := token.GenesisProposal.ExcludeCheckVoteAddress[address]; ok {
			continue // 不统计不参与计票的地址的投票
		}
		voteAmount, ok := big.NewInt(0).SetString(value, 10)
		if !ok {
			// 此处不应有错误，如果有，说明投票时检查的有问题。
			return nil, errors.New("vote value invalid, address: " + address)
		}
		voteCount = voteCount.Add(voteCount, voteAmount)
	}

	err = x.addFee(ctx, QueryProposalVotes)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   []byte(voteCount.String()),
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
		// 不应有此错误，若有说明投票或者检票时数据处理有问题。
		return errors.New("address voting topic empty when delete proposal by ID")
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

func (x *Contract) saveVote(ctx contract.KContext, token, topic string, proposalID, amount *big.Int) error {
	keyAddr2ID := []byte(KeyOfAddress2IDVote(token, topic, ctx.Initiator(), proposalID))
	value, err := ctx.Get(XTokenContract, keyAddr2ID)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return errors.Wrap(err, "address vote failed")
	}
	if len(value) != 0 {
		// 目前一个地址只能投票一次，不允许追加投票，不允许撤销投票，不允许重新投票。
		// 后期如果此要求有变更，需要修改此处判断。
		return errors.New("this account has already voted")
	}

	if err := ctx.Put(XTokenContract, keyAddr2ID, []byte(amount.String())); err != nil {
		return errors.Wrap(err, "save vote failed")
	}

	keyID2Addr := []byte(KeyOfID2AddressVote(token, topic, ctx.Initiator(), proposalID))
	if err := ctx.Put(XTokenContract, keyID2Addr, []byte(amount.String())); err != nil {
		return errors.Wrap(err, "save vote failed")
	}

	return nil
}
