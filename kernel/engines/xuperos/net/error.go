package xuperos

import (
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/protos"
)

var errorType = map[error]protos.XuperMessage_ErrorType{
	nil:                     protos.XuperMessage_SUCCESS,
	common.ErrChainNotExist: protos.XuperMessage_BLOCKCHAIN_NOTEXIST,
	common.ErrBlockNotExist: protos.XuperMessage_GET_BLOCK_ERROR,
	common.ErrParameter:     protos.XuperMessage_UNMARSHAL_MSG_BODY_ERROR,
}

func ErrorType(err error) protos.XuperMessage_ErrorType {
	if err == nil {
		return protos.XuperMessage_SUCCESS
	}

	if et, ok := errorType[err]; ok {
		return et
	}

	return protos.XuperMessage_UNKNOW_ERROR
}
