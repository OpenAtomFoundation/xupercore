package xtoken

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

var (
	contractIns *Contract
	newTokenCtx *FakeKContext
)

func TestNewToken(t *testing.T) {
	if contractIns != nil {
		return
	}
	token := XToken{
		Name:                  "test",
		TotalSupply:           big.NewInt(1000),
		AddSupplyEnabled:      true,
		BurnEnabled:           true,
		Decimal:               8,
		InitialAllocation:     map[string]*big.Int{},
		ConvertibleProportion: big.NewInt(1000),
		GenesisProposal: &GenesisProposal{
			ProposeMinToken:         big.NewInt(1),
			ProposalEffectiveAmount: big.NewInt(2),
		},
	}
	value, err := json.Marshal(token)
	if err != nil {
		t.Fatal("json marshal failed", err)
		return
	}
	args := map[string][]byte{
		"token": value,
	}
	ctx := NewFakeKContext(args, map[string]map[string][]byte{})

	cc := &Context{}
	contract := NewContract(nil, nil, cc)
	_, err = contract.NewToken(ctx)
	if err != nil {
		t.Fatal("NewToken test failed", err)
		return
	}
}

func TestTransfer(t *testing.T) {
	if contractIns == nil {
		newTokenForTest()
	}
	args := map[string][]byte{
		"tokenName": []byte("test"),
		"to":        []byte("bob"),
		"value":     []byte("10"),
	}
	newTokenCtx.args = args
	_, err := contractIns.Transfer(newTokenCtx)
	if err != nil {
		t.Fatal("transfer test failed", err)
		return
	}

	argsTransfer := map[string][]byte{
		"tokenName": []byte("test"),
		"address":   []byte("bob"),
	}
	newTokenCtx.args = argsTransfer
	resp, err := contractIns.BalanceOf(newTokenCtx)
	if err != nil {
		t.Fatal("transfer test failed", err)
		return
	}
	bal := new(Balance)
	err = json.Unmarshal(resp.Body, bal)
	if err != nil {
		t.Fatal("transfer test json unmarshal failed", err)
		return
	}

	// 转账 to 地址余额应该是10
	if bal.BalanceTotal.Cmp(big.NewInt(10)) != 0 {
		t.Fatal("transfer test to address balance assert failed")
		return
	}
}

func TestApprove(t *testing.T) {
	if contractIns == nil {
		newTokenForTest()
	}
	args := map[string][]byte{
		"tokenName": []byte("test"),
		"spender":   []byte("bob"),
		"value":     []byte("10"),
	}
	newTokenCtx.args = args
	_, err := contractIns.Approve(newTokenCtx)
	if err != nil {
		t.Fatal("Approve test failed", err)
		return
	}

	argsTransfer := map[string][]byte{
		"tokenName": []byte("test"),
		"spender":   []byte("bob"),
	}
	newTokenCtx.args = argsTransfer
	resp, err := contractIns.Allowance(newTokenCtx)
	if err != nil {
		t.Fatal("Allowance test failed", err)
		return
	}
	if string(resp.Body) != "10" {
		t.Fatal("Allowance test assert failed", err)
		return
	}

	args = map[string][]byte{
		"tokenName": []byte("test"),
		"from":      []byte(newTokenCtx.initiator),
		"to":        []byte("alice"),
		"value":     []byte("1"),
	}
	newTokenCtx.args = args
	oldInitiator := newTokenCtx.initiator
	newTokenCtx.initiator = "bob"
	_, err = contractIns.TransferFrom(newTokenCtx)
	if err != nil {
		t.Fatal("TransferFrom test failed", err)
		return
	}

	newTokenCtx.initiator = oldInitiator
}

func TestAddSupplyBurn(t *testing.T) {
	if contractIns == nil {
		newTokenForTest()
	}
	args := map[string][]byte{
		"tokenName": []byte("test"),
		"value":     []byte("100000"),
	}
	newTokenCtx.args = args
	_, err := contractIns.AddSupply(newTokenCtx)
	if err != nil {
		t.Fatal("AddSupply test failed", err)
		return
	}

	args = map[string][]byte{
		"tokenName": []byte("test"),
		"value":     []byte("100000"),
	}
	newTokenCtx.args = args
	_, err = contractIns.Burn(newTokenCtx)
	if err != nil {
		t.Fatal("Burn test failed", err)
		return
	}

	args = map[string][]byte{
		"tokenName": []byte("test"),
	}
	newTokenCtx.args = args
	resp, err := contractIns.TotalSupply(newTokenCtx)
	if err != nil {
		t.Fatal("Burn test failed", err)
		return
	}
	if string(resp.Body) != "1000" {
		t.Fatal("Burn test assert failed", err)
		return
	}
}

func TestQuery(t *testing.T) {
	if contractIns == nil {
		newTokenForTest()
	}
	args := map[string][]byte{
		"tokenName": []byte("test"),
	}
	newTokenCtx.args = args
	resp, err := contractIns.QueryTokenOwner(newTokenCtx)
	if err != nil {
		t.Fatal("QueryTokenOwner test failed", err)
		return
	}
	if string(resp.Body) != newTokenCtx.initiator {
		t.Fatal("QueryTokenOwner test assert failed")
		return
	}

	resp, err = contractIns.TotalSupply(newTokenCtx)
	if err != nil {
		t.Fatal("TotalSupply test failed", err)
		return
	}
	if string(resp.Body) != "1000" {
		t.Fatal("TotalSupply test assert failed")
		return
	}

	resp, err = contractIns.QueryToken(newTokenCtx)
	if err != nil {
		t.Fatal("QueryToken test failed", err)
		return
	}

	actualToken := new(XToken)
	err = json.Unmarshal(resp.Body, actualToken)
	if err != nil {
		t.Fatal("json marshal failed", err)
		return
	}
	if actualToken.Name != "test" {
		t.Fatal("QueryToken test assert failed")
		return
	}
}

func TestProposeCheckVote(t *testing.T) {
	if contractIns == nil {
		newTokenForTest()
	}
	args := map[string][]byte{
		"tokenName": []byte("test"),
		"topic":     []byte("test"),
		"data":      []byte("test"),
	}
	newTokenCtx.args = args
	// 提案
	resp, err := contractIns.Propose(newTokenCtx)
	if err != nil {
		t.Fatal("Propose test failed", err)
		return
	}

	type ProposalResult struct {
		ProposalID *big.Int
	}
	proposalResult := new(ProposalResult)
	err = json.Unmarshal(resp.Body, proposalResult)
	if err != nil {
		t.Fatal("json unmarshal failed", err)
		return
	}
	if proposalResult.ProposalID.String() != "1" {
		t.Fatal("Propose test assert failed")
		return
	}

	args = map[string][]byte{
		"tokenName":  []byte("test"),
		"topic":      []byte("test"),
		"proposalID": []byte("1"),
		"value":      []byte("10"),
		"option":     []byte("1"),
	}
	newTokenCtx.args = args
	// 投票
	resp, err = contractIns.Vote(newTokenCtx)
	if err != nil {
		t.Fatal("Vote test failed", err)
		return
	}

	if resp.Status != Success {
		t.Fatal("Vote test assert failed")
		return
	}

	args = map[string][]byte{
		"tokenName":  []byte("test"),
		"topic":      []byte("test"),
		"proposalID": []byte("1"),
	}
	newTokenCtx.args = args
	resp, err = contractIns.QueryProposal(newTokenCtx)
	if err != nil {
		t.Fatal("QueryProposal test failed", err)
		return
	}

	if resp.Status != Success {
		t.Fatal("QueryProposal test assert failed")
		return
	}

	resp, err = contractIns.CheckVote(newTokenCtx)
	if err != nil {
		t.Fatal("CheckVote test failed", err)
		return
	}
	if resp.Status != Success {
		t.Fatal("CheckVote test assert failed")
		return
	}

	resp, err = contractIns.QueryProposalVotes(newTokenCtx)
	if err != nil {
		t.Fatal("QueryProposalVotes test failed", err)
		return
	}
	if resp.Status != Success {
		t.Fatal("QueryProposalVotes test assert failed")
		return
	}
}

func TestProposeStopVote(t *testing.T) {
	if contractIns == nil {
		newTokenForTest()
	}
	args := map[string][]byte{
		"tokenName": []byte("test"),
		"topic":     []byte("testStopVote"),
		"data":      []byte("test"),
	}
	newTokenCtx.args = args
	// 提案
	resp, err := contractIns.Propose(newTokenCtx)
	if err != nil {
		t.Fatal("Propose test failed", err)
		return
	}

	type ProposalResult struct {
		ProposalID *big.Int
	}
	proposalResult := new(ProposalResult)
	err = json.Unmarshal(resp.Body, proposalResult)
	if err != nil {
		t.Fatal("json unmarshal failed", err)
		return
	}
	if proposalResult.ProposalID.String() != "1" {
		t.Fatal("Propose test assert failed")
		return
	}

	args = map[string][]byte{
		"tokenName":  []byte("test"),
		"topic":      []byte("testStopVote"),
		"proposalID": []byte("1"),
		"value":      []byte("10"),
		"option":     []byte("1"),
	}
	newTokenCtx.args = args
	// 投票
	resp, err = contractIns.Vote(newTokenCtx)
	if err != nil {
		t.Fatal("Vote test failed", err)
		return
	}

	if resp.Status != Success {
		t.Fatal("Vote test assert failed")
		return
	}

	args = map[string][]byte{
		"tokenName":  []byte("test"),
		"topic":      []byte("testStopVote"),
		"proposalID": []byte("1"),
	}
	newTokenCtx.args = args
	resp, err = contractIns.QueryProposal(newTokenCtx)
	if err != nil {
		t.Fatal("QueryProposal test failed", err)
		return
	}

	if resp.Status != Success {
		t.Fatal("QueryProposal test assert failed")
		return
	}

	resp, err = contractIns.StopVote(newTokenCtx)
	if err != nil {
		t.Fatal("StopVote test failed", err)
		return
	}
	if resp.Status != Success {
		t.Fatal("StopVote test assert failed")
		return
	}
}

func TestPermission(t *testing.T) {
	if contractIns == nil {
		newTokenForTest()
	}
	addrs := []string{
		newTokenCtx.initiator,
		"bob",
	}
	value, err := json.Marshal(addrs)
	if err != nil {
		t.Fatal("json marshal failed", err)
		return
	}
	args := map[string][]byte{
		"addrs": value,
	}
	newTokenCtx.args = args
	_, err = contractIns.AddAdmins(newTokenCtx)
	if err != nil {
		t.Fatal("AddAdmins test failed", err)
		return
	}

	resp, err := contractIns.QueryAdmins(newTokenCtx)
	if err != nil {
		t.Fatal("QueryAdmins test failed", err)
		return
	}

	admins := new(map[string]bool)
	err = json.Unmarshal(resp.Body, admins)
	if err != nil {
		t.Fatal("json unmarshal failed", err)
		return
	}
	if len(*admins) != 2 {
		t.Fatal("QueryAdmins test assert failed", err)
	}

	addrs = []string{
		"bob",
	}
	value, _ = json.Marshal(addrs)
	args = map[string][]byte{
		"addrs": value,
	}
	newTokenCtx.args = args
	_, err = contractIns.DelAdmins(newTokenCtx)
	if err != nil {
		t.Fatal("DelAdmins test failed", err)
		return
	}
}

func TestFee(t *testing.T) {
	if contractIns == nil {
		newTokenForTest()
	}
	args := map[string][]byte{
		"method": []byte("NewToken"),
		"fee":    []byte("101"),
	}
	newTokenCtx.args = args
	_, err := contractIns.SetFee(newTokenCtx)
	if err != nil {
		t.Fatal("SetFee test failed", err)
		return
	}

	args = map[string][]byte{
		"method": []byte("NewToken"),
	}
	newTokenCtx.args = args
	_, err = contractIns.GetFee(newTokenCtx)
	if err != nil {
		t.Fatal("GetFee test failed", err)
		return
	}
}

func TestTransferFronzen(t *testing.T) {
	if contractIns == nil {
		newTokenForTest()
	}
	args := map[string][]byte{
		"tokenName": []byte("test"),
		"topic":     []byte("testTransferFronzen"),
		"data":      []byte("test"),
	}
	newTokenCtx.args = args
	// 提案
	_, err := contractIns.Propose(newTokenCtx)
	if err != nil {
		t.Fatal("Propose test failed", err)
		return
	}

	args = map[string][]byte{
		"tokenName":  []byte("test"),
		"topic":      []byte("testTransferFronzen"),
		"proposalID": []byte("1"),
		"value":      []byte("10"),
		"option":     []byte("1"),
	}
	newTokenCtx.args = args
	// 投票
	resp, err := contractIns.Vote(newTokenCtx)
	if err != nil {
		t.Fatal("Vote test failed", err)
		return
	}

	if resp.Status != Success {
		t.Fatal("Vote test assert failed")
		return
	}

	args = map[string][]byte{
		"tokenName": []byte("test"),
		"to":        []byte("bob"),
		"value":     []byte("10"),
	}
	newTokenCtx.args = args
	_, err = contractIns.Transfer(newTokenCtx)
	if err == nil { // 预期失败
		if !strings.Contains(err.Error(), "insufficient account balance") {
			t.Fatal("transfer test failed", err)
		}
	}
}

func newTokenForTest() {
	token := XToken{
		Name:             "test",
		TotalSupply:      big.NewInt(1000),
		AddSupplyEnabled: true,
		BurnEnabled:      true,
		Decimal:          8,
		InitialAllocation: map[string]*big.Int{
			"mask": big.NewInt(1),
		},
		ConvertibleProportion: big.NewInt(1000),
		GenesisProposal: &GenesisProposal{
			ProposeMinToken:         big.NewInt(1),
			ProposalEffectiveAmount: big.NewInt(2),
			InitialData: []*Proposal{
				{Topic: "initialTopic"},
			},
		},
	}
	value, err := json.Marshal(token)
	if err != nil {
		panic(err)
	}
	args := map[string][]byte{
		"token": value,
	}
	newTokenCtx = NewFakeKContext(args, map[string]map[string][]byte{})

	cc := &Context{}
	contractIns = NewContract(map[string]bool{newTokenCtx.initiator: true}, nil, cc)
	_, err = contractIns.NewToken(newTokenCtx)
	if err != nil {
		panic(err)
	}
	fmt.Println("new token success")
}
