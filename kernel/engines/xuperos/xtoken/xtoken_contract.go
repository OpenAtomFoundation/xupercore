package xtoken

import (
	"encoding/json"
	"math/big"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

// IMPORTANT 合约内会使用到 big.Int 类型，无论作为参数还是存储数据时，都要先转换成string再转成bytes，不要使用big.Int.Bytes()。
// don't ask me why!

type Contract struct {
	Admins map[string]bool  // 创世配置的或者节点配置文件写的，如果通过交易设置admin，此字段会作废。
	Fees   map[string]int64 // 创世配置的或者节点配置文件写的，如果通过交易设置fee，此字段会作废。

	proposalChecking sync.Map
	lastCheckVoteErr error

	contractCtx *Context
}

func NewContract(admins map[string]bool, fee map[string]int64, ctx *Context) *Contract {
	adminsMap := make(map[string]bool, 0)
	if len(admins) > 0 {
		adminsMap = admins
	}
	feeMap := make(map[string]int64, 0)
	if len(fee) > 0 {
		feeMap = fee
	}
	return &Contract{
		Admins:      adminsMap,
		Fees:        feeMap,
		contractCtx: ctx,
	}
}

// ---------- ERC20 ----------

func (c *Contract) NewToken(ctx contract.KContext) (*contract.Response, error) {
	ok, err := c.checkPermission(ctx, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("check permission failed")
	}
	args := ctx.Args()
	token := new(XToken)
	if value, ok := args["token"]; !ok {
		return nil, errors.New("invalid param")
	} else if err := json.Unmarshal(value, token); err != nil {
		return nil, errors.Wrap(err, "invalid token")
	}
	// 1、检查token是否符合要求
	if err := c.checkTokenData(token, ctx); err != nil {
		return nil, err
	}
	c.contractCtx.XLog.Info("XToken", "tokenTotalSupply", token.TotalSupply, "tokenName", token.Name)

	// 2、保存token以及初始化提案数据
	if err := c.saveTokenAndProposal(token, ctx); err != nil {
		return nil, err
	}
	// 3、保存token owner以及账户余额
	// 4、保存初始分配方案下每个账户的余额
	if err := c.saveTokenBalance(token, ctx); err != nil {
		return nil, err
	}

	err = c.addFee(ctx, NewToken)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (c *Contract) TotalSupply(ctx contract.KContext) (*contract.Response, error) {
	tokenName := ctx.Args()["tokenName"]
	token, err := c.getToken(ctx, string(tokenName))
	if err != nil {
		return nil, err
	}
	err = c.addFee(ctx, TotalSupply)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   []byte(token.TotalSupply.String()),
	}, nil
}

func (c *Contract) BalanceOf(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	address := string(ctx.Args()["address"])
	if len(tokenName) == 0 || len(address) == 0 {
		return nil, errors.New("tokenName and address param can not be empty")
	}
	fromTotal, err := c.balanceOf(ctx, tokenName, address)
	if err != nil {
		return nil, err
	}
	balance := Balance{
		BalanceTotal: fromTotal,
	}
	if fromTotal.Cmp(big.NewInt(0)) == 0 {
		// 如果地址总余额为0，则不用统计冻结的金额。
		value, err := json.Marshal(balance)
		if err != nil {
			return nil, err
		}
		return &contract.Response{
			Status: Success,
			Body:   value,
		}, nil
	}
	frozen, err := c.getFrozenBalance(ctx, tokenName, address)
	if err != nil {
		return nil, err
	}
	balance.Frozen = frozen
	value, err := json.Marshal(balance)
	if err != nil {
		return nil, err
	}

	err = c.addFee(ctx, BalanceOf)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   value,
	}, nil
}

func (c *Contract) Transfer(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	to := string(ctx.Args()["to"])
	valueParam := string(ctx.Args()["value"])
	if len(tokenName) == 0 || len(to) == 0 || len(valueParam) == 0 {
		return nil, errors.New("tokenName and to and value param can not be empty")
	}

	value, ok := big.NewInt(0).SetString(valueParam, 10)
	if !ok {
		return nil, errors.New("invalid value param")
	}
	if value.Cmp(big.NewInt(0)) <= 0 {
		return nil, errors.New("invalid transfer value")
	}
	from := ctx.Initiator()
	fromTotal, err := c.balanceOf(ctx, tokenName, from)
	if err != nil {
		return nil, err
	}

	if fromTotal.Cmp(big.NewInt(0)) == 0 || fromTotal.Cmp(value) < 0 {
		return nil, errors.New("insufficient account balance")
	}

	frozen, err := c.getFrozenBalance(ctx, tokenName, from)
	if err != nil {
		return nil, err
	}

	// 判断冻结余额和本次转账金额
	usable := big.NewInt(0).Sub(fromTotal, frozen)
	if usable.Cmp(value) < 0 {
		return nil, errors.New("insufficient account balance transfer")
	}

	// 更新发起人所参与的提案
	if err = c.updateAddressVotingProposal(ctx, tokenName, ctx.Initiator()); err != nil {
		return nil, err
	}
	if err = c.updateVotingProposalByProposal(ctx, tokenName, ctx.Initiator()); err != nil {
		return nil, err
	}

	// 更新 from 地址余额
	fromBal := big.NewInt(0).Sub(fromTotal, value)
	err = c.saveAddressBalance(ctx, tokenName, from, fromBal)
	if err != nil {
		return nil, err
	}

	// 更新 to 地址余额
	toBalOld, err := c.balanceOf(ctx, tokenName, to)
	if err != nil {
		return nil, err
	}
	toBalNew := big.NewInt(0).Add(toBalOld, value)
	err = c.saveAddressBalance(ctx, tokenName, to, toBalNew)
	if err != nil {
		return nil, err
	}

	err = c.addFee(ctx, Transfer)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (c *Contract) TransferFrom(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	from := string(ctx.Args()["from"])
	to := string(ctx.Args()["to"])
	valueParam := string(ctx.Args()["value"])
	if len(tokenName) == 0 || len(from) == 0 || len(to) == 0 || len(valueParam) == 0 {
		return nil, errors.New("tokenName, from, to and value can not be empty")
	}
	value, ok := big.NewInt(0).SetString(valueParam, 10)
	if !ok {
		return nil, errors.New("invalid value param")
	}
	if value.Cmp(big.NewInt(0)) <= 0 {
		return nil, errors.New("invalid transfer value")
	}
	// 1、判断发起人是否有权限使用from的余额
	approveData, err := c.getApproveData(ctx, tokenName, from)
	if err != nil {
		return nil, err
	}
	approveValue, ok := approveData[ctx.Initiator()]
	if !ok {
		return nil, errors.New("check appvore permission failed")
	}
	// 2、检查from授权给发起人的金额
	if approveValue.Cmp(value) < 0 {
		return nil, errors.New("check appvore value failed")
	}

	// 3、检查from账户可用的余额是否足够本次转账
	fromBal, err := c.balanceOf(ctx, tokenName, from)
	if err != nil {
		return nil, err
	}
	if fromBal.Cmp(value) < 0 {
		return nil, errors.New("from address insufficient balance")
	}
	frozen, err := c.getFrozenBalance(ctx, tokenName, from)
	if err != nil {
		return nil, err
	}
	usable := big.NewInt(0).Sub(fromBal, frozen)
	if usable.Cmp(value) < 0 {
		return nil, errors.New("from address insufficient usable balance")
	}

	if err = c.updateAddressVotingProposal(ctx, tokenName, from); err != nil {
		return nil, err
	}
	if err = c.updateVotingProposalByProposal(ctx, tokenName, from); err != nil {
		return nil, err
	}

	// 4、更新from账户和to账户的余额
	fromNewBal := big.NewInt(0).Sub(fromBal, value)
	err = c.saveAddressBalance(ctx, tokenName, from, fromNewBal)
	if err != nil {
		return nil, err
	}

	// 更新 to 地址余额
	toBalOld, err := c.balanceOf(ctx, tokenName, to)
	if err != nil {
		return nil, err
	}
	toBalNew := big.NewInt(0).Add(toBalOld, value)
	err = c.saveAddressBalance(ctx, tokenName, to, toBalNew)
	if err != nil {
		return nil, err
	}

	// 更新授权的金额
	newApproveValue := approveValue.Sub(approveValue, value)
	approveData[ctx.Initiator()] = newApproveValue
	err = c.saveApproveData(ctx, tokenName, from, approveData)
	if err != nil {
		return nil, err
	}

	err = c.addFee(ctx, TransferFrom)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

// Approve 授权接口
func (c *Contract) Approve(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	spender := string(ctx.Args()["spender"])
	valueParam := string(ctx.Args()["value"])
	if len(tokenName) == 0 || len(spender) == 0 || len(valueParam) == 0 {
		return nil, errors.New("tokenName, spender and value can not be empty")
	}
	value, ok := big.NewInt(0).SetString(valueParam, 10)
	if !ok {
		return nil, errors.New("invalid value param")
	}
	if value.Cmp(big.NewInt(0)) <= 0 {
		return nil, errors.New("invalid approve value")
	}
	from := ctx.Initiator()

	bal, err := c.balanceOf(ctx, tokenName, from)
	if err != nil {
		return nil, err
	}

	if bal.Cmp(value) < 0 {
		return nil, errors.New("insufficient balance for approve, please check approve value")
	}

	data, err := c.getApproveData(ctx, tokenName, from)
	if err != nil {
		return nil, err
	}
	data[spender] = value
	err = c.saveApproveData(ctx, tokenName, from, data)
	if err != nil {
		return nil, err
	}

	err = c.addFee(ctx, Approve)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (c *Contract) Allowance(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	spender := string(ctx.Args()["spender"])
	if len(tokenName) == 0 || len(spender) == 0 {
		return nil, errors.New("tokenName and spender can not be empty")
	}
	data, err := c.getApproveData(ctx, tokenName, ctx.Initiator())
	if err != nil {
		return nil, err
	}

	err = c.addFee(ctx, Allowance)
	if err != nil {
		return nil, err
	}

	if value, ok := data[spender]; !ok {
		return &contract.Response{
			Status: Success,
			Body:   []byte(big.NewInt(0).String()),
		}, nil
	} else {
		return &contract.Response{
			Status: Success,
			Body:   []byte(value.String()),
		}, nil
	}
}

func (c *Contract) AddSupply(ctx contract.KContext) (*contract.Response, error) {
	tokenName := ctx.Args()["tokenName"]
	value := ctx.Args()["value"]
	if len(tokenName) == 0 || len(value) == 0 {
		return nil, errors.New("invalid tokenName")
	}
	token, err := c.getToken(ctx, string(tokenName))
	if err != nil {
		return nil, err
	}

	if err = c.requireOwner(ctx, string(tokenName)); err != nil {
		return nil, err
	}

	if !token.AddSupplyEnabled {
		return nil, errors.New("token add supply disabled")
	}

	add, ok := big.NewInt(0).SetString(string(value), 10)
	if !ok {
		return nil, errors.New("invalid value")
	}
	if add.Cmp(big.NewInt(0)) <= 0 {
		return nil, errors.New("invalid add supply value")
	}
	token.TotalSupply = big.NewInt(0).Add(token.TotalSupply, add)
	tokenValue, err := json.Marshal(token)
	if err != nil {
		return nil, errors.Wrap(err, "save token json marsha token failed")
	}
	// 更新token。
	err = ctx.Put(XTokenContract, []byte(KeyOfToken(token.Name)), tokenValue)
	if err != nil {
		return nil, errors.Wrap(err, "save token failed")
	}

	// 更新 owner 余额。
	bal, err := c.balanceOf(ctx, token.Name, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	err = c.saveAddressBalance(ctx, token.Name, ctx.Initiator(), bal.Add(bal, add))
	if err != nil {
		return nil, err
	}

	err = c.addFee(ctx, AddSupply)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (c *Contract) Burn(ctx contract.KContext) (*contract.Response, error) {
	tokenName := ctx.Args()["tokenName"]
	value := ctx.Args()["value"]
	if len(tokenName) == 0 || len(value) == 0 {
		return nil, errors.New("invalid tokenName")
	}
	token, err := c.getToken(ctx, string(tokenName))
	if err != nil {
		return nil, err
	}

	if !token.BurnEnabled {
		return nil, errors.New("token burn disabled")
	}

	burn, ok := big.NewInt(0).SetString(string(value), 10)
	if !ok {
		return nil, errors.New("invalid value")
	}
	if burn.Cmp(big.NewInt(0)) <= 0 {
		return nil, errors.New("invalid burn value")
	}
	// 检查账户余额
	bal, err := c.balanceOf(ctx, token.Name, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if bal.Cmp(burn) < 0 {
		return nil, errors.New("insufficient balance to burn")
	}

	// 检查可用的余额
	frozen, err := c.getFrozenBalance(ctx, token.Name, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	usable := big.NewInt(0).Sub(bal, frozen)
	if usable.Cmp(burn) < 0 {
		return nil, errors.New("insufficient usalbe balance to burn")
	}

	if err = c.updateAddressVotingProposal(ctx, token.Name, ctx.Initiator()); err != nil {
		return nil, err
	}
	if err = c.updateVotingProposalByProposal(ctx, token.Name, ctx.Initiator()); err != nil {
		return nil, err
	}

	// 更新账户余额
	err = c.saveAddressBalance(ctx, token.Name, ctx.Initiator(), bal.Sub(bal, burn))
	if err != nil {
		return nil, err
	}

	token.TotalSupply = big.NewInt(0).Sub(token.TotalSupply, burn)
	tokenValue, err := json.Marshal(token)
	if err != nil {
		return nil, errors.Wrap(err, "save token json marsha token failed")
	}
	// 更新token。
	err = ctx.Put(XTokenContract, []byte(KeyOfToken(token.Name)), tokenValue)
	if err != nil {
		return nil, errors.Wrap(err, "save token failed")
	}

	err = c.addFee(ctx, Burn)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (c *Contract) QueryToken(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	if len(tokenName) == 0 {
		return nil, errors.New("invalid tokenName")
	}
	token, err := c.getToken(ctx, tokenName)
	if err != nil {
		return nil, err
	}
	value, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	err = c.addFee(ctx, QueryToken)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   value,
	}, nil
}

func (c *Contract) QueryTokenOwner(ctx contract.KContext) (*contract.Response, error) {
	tokenName := string(ctx.Args()["tokenName"])
	if len(tokenName) == 0 {
		return nil, errors.New("invalid tokenName")
	}
	owner, err := c.getTokenOwner(ctx, tokenName)
	if err != nil {
		return nil, err
	}

	err = c.addFee(ctx, QueryTokenOwner)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   []byte(owner),
	}, nil
}

func (c *Contract) checkTokenData(token *XToken, ctx contract.KContext) error {
	value, err := ctx.Get(XTokenContract, []byte(KeyOfToken(token.Name)))
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return err
	}
	if len(value) != 0 { // 检查token是否已经存在。
		return errors.New("token already exist")
	}
	if strings.Contains(token.Name, "_") {
		// 限制 token name 不能有下划线
		return errors.New("token name cannot contain underscores")
	}
	// 检查token相关参数。
	if token.ConvertibleProportion.Cmp(big.NewInt(0)) < 0 {
		return errors.New("invalid token ConvertibleProportion")
	}
	owner := ctx.Initiator()
	if len(token.InitialAllocation) > 0 {
		count := big.NewInt(0)
		for addr, value := range token.InitialAllocation {
			if value.Cmp(big.NewInt(0)) <= 0 {
				return errors.New("invalid InitialAllocation value")
			}
			if len(addr) <= 0 {
				return errors.New("invalid token InitialAllocation")
			}
			if addr == owner {
				return errors.New("the initial allocation scheme cannot include owner")
			}
			count = count.Add(count, value)
		}
		if count.Cmp(token.TotalSupply) > 0 {
			return errors.New("invalid token InitialAllocation")
		}
	}

	if token.GenesisProposal == nil {
		return errors.New("invalid token GenesisProposal")
	}
	// 提案生效限制
	if token.GenesisProposal.ProposalEffectiveAmount == nil || token.GenesisProposal.ProposalEffectiveAmount.Cmp(big.NewInt(0)) < 0 {
		return errors.New("invalid token ProposalEffectiveAmount")
	}

	// 创建token时的初始化提案数据的topic不能为空，且提案ID必须是0。
	if len(token.GenesisProposal.InitialData) > 0 {
		for _, p := range token.GenesisProposal.InitialData {
			if p.Topic == "" {
				return errors.New("invalid token proposal initial topic")
			}
			if p.ID == nil {
				p.ID = big.NewInt(0)
			}
			if p.ID.Cmp(big.NewInt(0)) > 0 {
				return errors.New("invalid token proposal initial ID")
			}
		}
	}

	return nil
}

func (c *Contract) saveTokenAndProposal(token *XToken, ctx contract.KContext) error {
	value, err := json.Marshal(token)
	if err != nil {
		return errors.Wrap(err, "save token json marsha token failed")
	}
	// 保存token，包括token数据以及提案配置
	err = ctx.Put(XTokenContract, []byte(KeyOfToken(token.Name)), value)
	if err != nil {
		return errors.Wrap(err, "save token failed")
	}
	// 保存owner
	owner := ctx.Initiator()
	err = ctx.Put(XTokenContract, []byte(KeyOfTokenOwner(token.Name)), []byte(owner))
	if err != nil {
		return errors.Wrap(err, "save token owner failed")
	}

	// 保存初始提案数据，如果有。
	if token.GenesisProposal != nil {
		for _, p := range token.GenesisProposal.InitialData {
			if err := c.saveProposal(ctx, token.Name, p); err != nil {
				return err
			}
			if err := c.setTopicData(ctx, token.Name, p.Topic, p.Data); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Contract) saveTokenBalance(token *XToken, ctx contract.KContext) error {
	// 保存相关账户余额
	count := big.NewInt(0)
	for addr, value := range token.InitialAllocation {
		count = count.Add(count, value)
		err := c.saveAddressBalance(ctx, token.Name, addr, value)
		if err != nil {
			return err
		}
	}
	ownerBal := big.NewInt(0)
	ownerBal = ownerBal.Sub(token.TotalSupply, count)
	err := c.saveAddressBalance(ctx, token.Name, ctx.Initiator(), ownerBal)
	if err != nil {
		return err
	}

	return nil
}
func (c *Contract) saveAddressBalance(ctx contract.KContext, tokenName, address string, value *big.Int) error {
	err := ctx.Put(XTokenContract, []byte(KeyOfAddress2TokenBalance(tokenName, address)), []byte(value.String()))
	if err != nil {
		return errors.Wrap(err, "save address balance failed")
	}
	err = ctx.Put(XTokenContract, []byte(KeyOfToken2AddressBalance(tokenName, address)), []byte(value.String()))
	if err != nil {
		return errors.Wrap(err, "save address balance failed")
	}
	return nil
}

func (c *Contract) getApproveData(ctx contract.KContext, token, address string) (map[string]*big.Int, error) {
	key := []byte(KeyOfAllowances(token, address))
	value, err := ctx.Get(XTokenContract, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get allwance failed")
	}
	data := make(map[string]*big.Int, 0)
	if len(value) == 0 {
		return data, nil
	}
	err = json.Unmarshal(value, &data)
	if err != nil {
		return nil, err
	}
	return data, nil

}

func (c *Contract) saveApproveData(ctx contract.KContext, token, address string, data map[string]*big.Int) error {
	value, err := json.Marshal(data)
	if err != nil {
		return err
	}
	key := []byte(KeyOfAllowances(token, address))
	return ctx.Put(XTokenContract, key, value)
}

// 查询余额时，需要根据提案的状态过滤真正冻结的金额
func (c *Contract) getFrozenBalance(ctx contract.KContext, tokenName, address string) (*big.Int, error) {
	votingProposalMap, err := c.getAddressVotingProposal(ctx, tokenName, address)
	if err != nil {
		return nil, err
	}
	max := big.NewInt(0)
	// 此函数的复杂度On^2，实际上一个地址参与的不同类型下的不同提案且为voting状态的不会很多。
	// 所以复杂度中的n为一个地址在一个token下，所有参与的提案topic下所有状态为voting的提案的个数。
	// 并且实际业务场景中，大多数一个地址参与的不同类型的提案不会很多，且同时为voting状态的提案也不会很多。
	for topic, pid2amount := range votingProposalMap {
		for pidstr, amount := range pid2amount {
			if amount.Cmp(max) <= 0 {
				continue
			}
			pid, _ := big.NewInt(0).SetString(pidstr, 10)
			p, err := c.getProposal(ctx, tokenName, topic, pid)
			if err != nil {
				return nil, err
			}
			if p.Status == ProposalVoting {
				max = amount
			}
		}
	}

	// 查询是否有因为发起提案锁定的余额。
	proposerProposalMap, err := c.getVotingProposalByProposer(ctx, tokenName, address)
	if err != nil {
		return nil, errors.Wrap(err, "getVotingProposalByProposer failed")
	}

	for topic, pid2amount := range proposerProposalMap {
		for pidstr, amount := range pid2amount {
			if amount.Cmp(max) <= 0 {
				continue
			}
			pid, _ := big.NewInt(0).SetString(pidstr, 10)
			p, err := c.getProposal(ctx, tokenName, topic, pid)
			if err != nil {
				return nil, err
			}
			if p.Status == ProposalVoting {
				max = amount
			}
		}
	}
	return max, nil
}

func (c *Contract) getToken(ctx contract.KContext, tokenName string) (*XToken, error) {
	value, err := ctx.Get(XTokenContract, []byte(KeyOfToken(string(tokenName))))
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get token failed")
	}
	if len(value) == 0 {
		return nil, errors.New("token no exist")
	}
	token := new(XToken)
	err = json.Unmarshal(value, token)
	if err != nil {
		return nil, errors.Wrap(err, "token unmarshal failed")
	}
	return token, nil
}

func (c *Contract) getTokenOwner(ctx contract.KContext, tokenName string) (string, error) {
	value, err := ctx.Get(XTokenContract, []byte(KeyOfTokenOwner(string(tokenName))))
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return "", errors.Wrap(err, "get token owner failed")
	}
	if len(value) == 0 {
		return "", errors.New("token owner no exist")
	}
	return string(value), nil
}

func (c *Contract) requireOwner(ctx contract.KContext, tokenName string) error {
	owner, err := c.getTokenOwner(ctx, string(tokenName))
	if err != nil {
		return err
	}
	if ctx.Initiator() != owner {
		return errors.New("check token owner failed")
	}
	return nil
}

func (c *Contract) balanceOf(ctx contract.KContext, token, address string) (*big.Int, error) {
	key := []byte(KeyOfToken2AddressBalance(token, address))
	value, err := ctx.Get(XTokenContract, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get address balance failed")
	}
	bal := big.NewInt(0)
	if len(value) == 0 {
		return bal, nil
	}
	bal, ok := bal.SetString(string(value), 10)
	if !ok {
		return nil, errors.New("get address balance bigInt set string failed")
	}
	return bal, nil
}

func (c *Contract) addFee(ctx contract.KContext, method string) error {
	key := []byte(KeyOfFee(method))
	value, err := ctx.Get(XTokenContract, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return errors.Wrap(err, "get fee failed")
	}
	fee := int64(0)
	if len(value) == 0 {
		// 如果数据库中没有，则从配置中读取。
		// 配置中没有则为0。
		fee = c.Fees[method]
	} else {
		// 如果数据库中有，则以数据库中为主。

		feeBig, ok := big.NewInt(0).SetString(string(value), 10)
		if !ok {
			return errors.New("get fee bigInt set string failed")
		}
		fee = feeBig.Int64()
	}

	delta := contract.Limits{
		XFee: fee,
	}
	ctx.AddResourceUsed(delta)
	return nil
}
