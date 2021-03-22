package xpoa

import (
	"encoding/json"
	"sort"
	"strconv"
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
	// 1. 核查发起者的权限
	aks := make(map[string]float64)
	if err := json.Unmarshal([]byte(txArgs["aksWeight"]), &aks); err != nil {
		return NewContractErrResponse(statusBadRequest, "invalid acl: unmarshal err."), err
	}
	totalBytes := txArgs["rule"]
	totalStr := string(totalBytes)
	total, err := strconv.ParseInt(totalStr, 10, 32)
	if total != 1 || err != nil { // 目前必须是阈值模型
		return NewContractErrResponse(statusBadRequest, "invalid acl: rule should eq 1."), err
	}
	acceptBytes := txArgs["acceptValue"]
	acceptStr := string(acceptBytes)
	acceptValue, err := strconv.ParseFloat(acceptStr, 64)
	if err != nil {
		return NewContractErrResponse(statusBadRequest, "invalid acl: pls check accept value."), err
	}
	if !x.isAuthAddress(aks, acceptValue) {
		return NewContractErrResponse(statusBadRequest, aclErr.Error()), aclErr
	}

	// 2. 检查desc参数权限
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

// isAuthAddress 判断输入aks是否能在贪心下仍能满足签名数量>33%(Chained-BFT装载) or 50%(一般情况)
func (x *xpoaConsensus) isAuthAddress(aks map[string]float64, threshold float64) bool {
	// 1. 判断aks中的地址是否是当前集合地址
	for addr, _ := range aks {
		if !Find(addr, x.election.validators) {
			return false
		}
	}
	// 2. 判断贪心下签名集合数目仍满足要求
	var s aksSlice
	for k, v := range aks {
		s = append(s, aksItem{
			Address: k,
			Weight:  v,
		})
	}
	sort.Stable(s)
	greedyCount := 0
	sum := threshold
	for i := 0; i < len(aks); i++ {
		if sum > 0 {
			sum -= s[i].Weight
			greedyCount++
			continue
		}
		break
	}
	if !x.election.enableBFT {
		return greedyCount >= len(x.election.validators)/2+1
	}
	return CalFault(int64(greedyCount), int64(len(x.election.validators)))
}
