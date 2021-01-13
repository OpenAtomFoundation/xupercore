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
	return CastErrorDefault(err, ErrUnknown)
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
	ErrUnknown            = &Error{ErrStatusInternalErr, 50001, "unknown error"}
	ErrForbidden          = &Error{ErrStatusRefused, 40300, "forbidden"}
	ErrUnauthorized       = &Error{ErrStatusRefused, 40100, "unauthorized"}
	ErrParameter          = &Error{ErrStatusRefused, 40001, "param error"}

	// engine
	ErrNewEngineCtxFailed = &Error{ErrStatusInternalErr, 50003, "create engine context failed"}
	ErrNotEngineType      = &Error{ErrStatusRefused, 40010, "transfer engine type failed"}
	ErrLoadEngConfFailed  = &Error{ErrStatusInternalErr, 50006, "load engine config failed"}
	ErrNewLogFailed       = &Error{ErrStatusInternalErr, 50006, "new logger failed"}

	// chain
	ErrNewChainCtxFailed  = &Error{ErrStatusInternalErr, 50011, "new chain context failed"}
	ErrChainExist         = &Error{ErrStatusRefused, 40002, "chain already exists"}
	ErrChainNotExist      = &Error{ErrStatusRefused, 40003, "chain not exist"}
	ErrChainAlreadyExist  = &Error{ErrStatusRefused, 40004, "chain already exist"}
	ErrChainStatus        = &Error{ErrStatusRefused, 40005, "chain status error"}
	ErrRootChainNotExist  = &Error{ErrStatusRefused, 40006, "root chain not exist"}
	ErrLoadChainFailed    = &Error{ErrStatusInternalErr, 50004, "load chain failed"}

	ErrContractNewCtxFailed = &Error{ErrStatusInternalErr, 50004, "contract new context failed"}
	ErrContractInvokeFailed = &Error{ErrStatusInternalErr, 50004, "contract invoke failed"}

	// tx
	ErrTxVerifyFailed     = &Error{ErrStatusInternalErr, 40011, "verify tx failed"}
	ErrTxAlreadyExist     = &Error{ErrStatusInternalErr, 40013, "tx already exist"}
	ErrTxNotExist         = &Error{ErrStatusInternalErr, 40014, "tx not exist"}
	ErrTxNotEnough        = &Error{ErrStatusInternalErr, 50017, "tx not enough"}
	ErrSubmitTxFailed     = &Error{ErrStatusInternalErr, 50013, "submit tx failed"}

	// block
	ErrBlockNotExist      = &Error{ErrStatusInternalErr, 50015, "block not exist"}
	ErrProcBlockFailed    = &Error{ErrStatusInternalErr, 50015, "process block failed"}

	// net
	ErrNewNetEventFailed  = &Error{ErrStatusInternalErr, 50005, "new net event failed"}
	ErrNewNetworkFailed   = &Error{ErrStatusInternalErr, 50010, "new network failed"}
	ErrSendMessageFailed  = &Error{ErrStatusInternalErr, 50016, "send message failed"}
	ErrNetworkNoResponse  = &Error{ErrStatusInternalErr, 50016, "network no response"}
)
