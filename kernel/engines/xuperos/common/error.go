package common

import (
	"fmt"
)

const (
	// 处理成功类
	ErrStatusSucc = 200
	// 拒绝处理类错误状态
	ErrStatusRefused = 400
	// 内部错误类错误状态
	ErrStatusInternalErr = 500
)

type Error struct {
	// 用于统计和监控的错误分类（类似http的2xx、4xx、5xx）
	Status int
	// 用于标识具体错误的详细错误码
	Code int
	// 用于说明具体错误的说明信息
	Msg string
}

func CastError(err error) *Error {
	return CastErrorDefault(err, ErrUnknow)
}

func CastErrorDefault(err error, defaultErr *Error) *Error {
	if err == nil {
		return nil
	}
	if defErr, ok := err.(*Error); ok {
		return defErr
	}

	return defaultErr.More(err.Error())
}

func (t *Error) Error() string {
	return fmt.Sprintf("Err:%d-%d-%s", t.Status, t.Code, t.Msg)
}

func (t *Error) More(format string, args ...interface{}) *Error {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	return &Error{t.Status, t.Code, t.Msg + "+" + msg}
}

func (t *Error) Equal(rhs *Error) bool {
	if rhs == nil {
		return false
	}

	return t.Code == rhs.Code
}

// define std error
// 预留xxx9xx的错误码给上层业务扩展用，这里不要使用xxx9xx的错误码
var (
	ErrSuccess            = &Error{ErrStatusSucc, 0, "success"}
	ErrInternal           = &Error{ErrStatusInternalErr, 50000, "internal error"}
	ErrUnknow             = &Error{ErrStatusInternalErr, 50001, "unknow error"}
	ErrForbidden          = &Error{ErrStatusRefused, 40300, "forbidden"}
	ErrUnauthorized       = &Error{ErrStatusRefused, 40100, "unauthorized"}
	ErrParameter          = &Error{ErrStatusRefused, 40001, "param error"}
	ErrChainExist         = &Error{ErrStatusRefused, 40002, "chain already exists"}
	ErrChainNotExist      = &Error{ErrStatusRefused, 40003, "chain not exist"}
	ErrChainAlreadyExist  = &Error{ErrStatusRefused, 40004, "chain already exist"}
	ErrRootChainNotExist  = &Error{ErrStatusRefused, 40005, "root chain not exist"}
	ErrNotEngineType      = &Error{ErrStatusRefused, 40010, "transfer engine type failed"}
	ErrTxVerifyFailed     = &Error{ErrStatusRefused, 40011, "verify tx failed"}
	ErrTxAlreadyExist     = &Error{ErrStatusRefused, 40013, "tx already exist"}
	ErrNewEngineCtxFailed = &Error{ErrStatusInternalErr, 50003, "create engine context failed"}
	ErrLoadChainFailed    = &Error{ErrStatusInternalErr, 50004, "load chain failed"}
	ErrNewNetEventFailed  = &Error{ErrStatusInternalErr, 50005, "new net event failed"}
	ErrNewLogFailed       = &Error{ErrStatusInternalErr, 50006, "new logger failed"}
	ErrLoadEngConfFailed  = &Error{ErrStatusInternalErr, 50006, "load engine config failed"}
	ErrNewNetworkFailed   = &Error{ErrStatusInternalErr, 50010, "new network failed"}
	ErrNewChainCtxFailed  = &Error{ErrStatusInternalErr, 50011, "new chain context failed"}
	ErrSubmitTxFailed     = &Error{ErrStatusInternalErr, 50013, "submit tx failed"}
	ErrProcBlockFailed    = &Error{ErrStatusInternalErr, 50015, "process block failed"}
	ErrNetworkNoResponse  = &Error{ErrStatusInternalErr, 50016, "network no response"}
	ErrTxNotEnough        = &Error{ErrStatusInternalErr, 50017, "tx not enough"}
)
