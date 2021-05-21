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
	ErrSuccess      = &Error{ErrStatusSucc, 0, "success"}
	ErrInternal     = &Error{ErrStatusInternalErr, 50000, "internal error"}
	ErrUnknown      = &Error{ErrStatusInternalErr, 50001, "unknown error"}
	ErrForbidden    = &Error{ErrStatusRefused, 40000, "forbidden"}
	ErrParameter    = &Error{ErrStatusRefused, 40001, "param error"}
	ErrUnauthorized = &Error{ErrStatusRefused, 40002, "unauthorized"}

	// engine
	ErrNewEngineCtxFailed = &Error{ErrStatusInternalErr, 50100, "create engine context failed"}
	ErrNotEngineType      = &Error{ErrStatusInternalErr, 50101, "transfer engine type failed"}
	ErrLoadEngConfFailed  = &Error{ErrStatusInternalErr, 50102, "load engine config failed"}
	ErrNewLogFailed       = &Error{ErrStatusInternalErr, 50103, "new logger failed"}

	// chain
	ErrNewChainCtxFailed = &Error{ErrStatusInternalErr, 50200, "new chain context failed"}
	ErrLoadChainFailed   = &Error{ErrStatusInternalErr, 50201, "load chain failed"}
	ErrRootChainNotExist = &Error{ErrStatusInternalErr, 50202, "root chain not exist"}
	ErrChainStatus       = &Error{ErrStatusInternalErr, 50203, "chain status error"}
	ErrChainExist        = &Error{ErrStatusInternalErr, 50204, "chain already exists"}
	ErrChainNotExist     = &Error{ErrStatusInternalErr, 50205, "chain not exist"}
	ErrChainAlreadyExist = &Error{ErrStatusInternalErr, 50206, "chain already exist"}

	// block
	ErrBlockNotExist    = &Error{ErrStatusInternalErr, 50300, "block not exist"}
	ErrProcBlockFailed  = &Error{ErrStatusInternalErr, 50301, "process block failed"}
	ErrGenesisBlockDiff = &Error{ErrStatusInternalErr, 50302, "genesis block diff"}

	// tx
	ErrTxVerifyFailed        = &Error{ErrStatusInternalErr, 50400, "verify tx failed"}
	ErrTxAlreadyExist        = &Error{ErrStatusInternalErr, 50401, "tx already exist"}
	ErrTxNotExist            = &Error{ErrStatusInternalErr, 50402, "tx not exist"}
	ErrTxNotEnough           = &Error{ErrStatusInternalErr, 50403, "tx not enough"}
	ErrSubmitTxFailed        = &Error{ErrStatusInternalErr, 50404, "submit tx failed"}
	ErrGenerateTimerTxFailed = &Error{ErrStatusInternalErr, 50405, "generate timer tx failed"}

	// contract
	ErrContractNewCtxFailed     = &Error{ErrStatusInternalErr, 50500, "contract new context failed"}
	ErrContractInvokeFailed     = &Error{ErrStatusInternalErr, 50501, "contract invoke failed"}
	ErrContractNewSandboxFailed = &Error{ErrStatusInternalErr, 50502, "contract new sandbox failed"}

	// net
	ErrNewNetEventFailed = &Error{ErrStatusInternalErr, 50600, "new net event failed"}
	ErrNewNetworkFailed  = &Error{ErrStatusInternalErr, 50601, "new network failed"}
	ErrSendMessageFailed = &Error{ErrStatusInternalErr, 50602, "send message failed"}
	ErrNetworkNoResponse = &Error{ErrStatusInternalErr, 50603, "network no response"}

	// consensus
	ErrConsensusStatus = &Error{ErrStatusInternalErr, 50701, "consensus status error"}
)
