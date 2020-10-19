package utils

import (
	"fmt"
)

// 转换账本句柄类型
func TransLedgerHandle(handle interface{}) (*Ledger, error) {

}

// 转换Submit执行响应结构
func TransSubmitResp(resp interface{}) (*SubmitTxRes, error) {
	if resp == nil {
		return nil, fmt.Errorf("transfer submit resp type failed because param is nil")
	}

	if v, ok := resp.(*SubmitTxRes); ok {
		return v, nil
	}

	return nil, fmt.Errorf("transfer submit resp type failed by type assert")
}

// 转换Submit请求结构
func transSubmitReq(req interface{}) (*SubmitTxReq, error) {
	if req == nil {
		return nil, fmt.Errorf("transfer submit req type failed because param is nil")
	}

	if v, ok := db.(*SubmitTxReq); ok {
		return v, nil
	}

	return nil, fmt.Errorf("transfer submit req type failed by type assert")
}
