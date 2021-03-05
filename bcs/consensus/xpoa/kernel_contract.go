package xpoa

import (
	"encoding/json"
	"strings"

	"github.com/xuperchain/xupercore/kernel/contract"
)

func NewContractErrResponse(status int, msg string) *contract.Response {
	return &contract.Response{
		Status:  status,
		Message: msg,
	}
}

func NewContractOKResponse(json []byte) *contract.Response {
	return &contract.Response{
		Status:  statusOK,
		Message: "success",
		Body:    json,
	}
}

// runChangeValidates 候选人变更，替代原三代合约的add_validates/delete_validates/change_validates三个操作方法
// Args: validates::候选人钱包地址
func (x *xpoaConsensus) methodEditValidates(contractCtx contract.KContext) (*contract.Response, error) {
	// 核查变更候选人合约参数有效性
	txArgs := contractCtx.Args()
	validatesBytes := txArgs["validates"]
	validatesAddrs := string(validatesBytes)
	if validatesAddrs == "" {
		return NewContractErrResponse(statusBadRequest, targetParamErr.Error()), targetParamErr
	}
	validators := strings.Split(validatesAddrs, ";")
	rawV := &ValidatorsInfo{
		Validators: validators,
	}
	rawBytes, err := json.Marshal(rawV)
	if err != nil {
		return NewContractErrResponse(statusErr, err.Error()), err
	}
	if err := contractCtx.Put(contractBucket, []byte(validateKeys), rawBytes); err != nil {
		return NewContractErrResponse(statusErr, err.Error()), err
	}
	delta := contract.Limits{
		XFee: fee,
	}
	contractCtx.AddResourceUsed(delta)
	return NewContractOKResponse(rawBytes), nil
}

// methodGetValidates 候选人获取
// Return: validates::候选人钱包地址
func (x *xpoaConsensus) methodGetValidates(contractCtx contract.KContext) (*contract.Response, error) {
	var jsonBytes []byte
	validatesBytes, err := contractCtx.Get(contractBucket, []byte(validateKeys))
	if err != nil {
		originValidators := x.election.GetValidators(x.election.ledger.GetTipBlock().GetHeight())
		returnV := map[string][]string{
			"validators": originValidators,
		}
		jsonBytes, err = json.Marshal(returnV)
		if err != nil {
			return NewContractErrResponse(statusErr, err.Error()), err
		}
	} else {
		jsonBytes = validatesBytes
	}
	delta := contract.Limits{
		XFee: fee / 1000,
	}
	contractCtx.AddResourceUsed(delta)
	return NewContractOKResponse(jsonBytes), nil
}
