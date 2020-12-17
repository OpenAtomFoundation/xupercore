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
var (
	ErrSuccess           = &Error{ErrStatusSucc, 0, "success"}
	ErrInternal          = &Error{ErrStatusInternalErr, 50000, "internal error"}
	ErrForbidden         = &Error{ErrStatusRefused, 40300, "forbidden"}
	ErrUnauthorized      = &Error{ErrStatusRefused, 40100, "unauthorized"}
	ErrParameter         = &Error{ErrStatusRefused, 40001, "param error"}
	ErrChainExist        = &Error{ErrStatusRefused, 40002, "chain already exists"}
	ErrChainNotExist     = &Error{ErrStatusRefused, 40003, "chain not exist"}
	ErrRootChainNotExist = &Error{ErrStatusRefused, 40004, "root chain not exist"}
)
