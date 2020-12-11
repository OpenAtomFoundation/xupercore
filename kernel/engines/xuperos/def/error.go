// error for pb ErrorType
package def

import (
	"errors"
	pb "github.com/xuperchain/xupercore/kernel/network/pb"
)

var (
	Success = errors.New("success")
)

// blockchain
var (
	ErrBlockChainExist    = errors.New("blockchain existed")
	ErrBlockChainNotExist = errors.New("blockchain not exist")
)

// p2p message
var (
	ErrMessageUnmarshal = errors.New("message unmarshal error")
	ErrMessageParam     = errors.New("message param error")
)

var errorType = map[error]pb.XuperMessage_ErrorType{
	Success:               pb.XuperMessage_SUCCESS,
	ErrBlockChainNotExist: pb.XuperMessage_BLOCKCHAIN_NOTEXIST,
	ErrMessageUnmarshal:   pb.XuperMessage_UNMARSHAL_MSG_BODY_ERROR,
}

func ErrorType(err error) pb.XuperMessage_ErrorType {
	if err == nil {
		return pb.XuperMessage_SUCCESS
	}

	if errorType, ok := errorType[err]; ok {
		return errorType
	}

	return pb.XuperMessage_UNKNOW_ERROR
}
