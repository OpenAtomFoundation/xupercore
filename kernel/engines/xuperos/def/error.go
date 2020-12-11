package def

import (
	"errors"
	"github.com/xuperchain/xuperchain/core/pb"

	netPB "github.com/xuperchain/xupercore/kernel/network/pb"

)

var (
	Success = errors.New("success")
)

// blockchain
var (
	ErrBlockChainNotReady = errors.New("block chain not ready")
	ErrBlockChainExist    = errors.New("blockchain existed")
	ErrBlockChainNotExist = errors.New("blockchain not exist")
)

// p2p message
var (
	ErrMessageUnmarshal = errors.New("message unmarshal error")
	ErrMessageParam = errors.New("message param error")

	ErrNoResponse = errors.New("no response")
	ErrGetBlockError = errors.New("get block error")
	ErrGetBlockChainError = errors.New("get block chain error")
	ErrGetBlockChainStatusError = errors.New("get block chain status error")
)

// error for p2p pb ErrorType
var errorType = map[error]netPB.XuperMessage_ErrorType{
	Success:               netPB.XuperMessage_SUCCESS,
	ErrBlockChainNotExist: netPB.XuperMessage_BLOCKCHAIN_NOTEXIST,
	ErrMessageUnmarshal:   netPB.XuperMessage_UNMARSHAL_MSG_BODY_ERROR,
	// TODO: 定义新的pb错误类型
	ErrMessageParam: 	   netPB.XuperMessage_UNMARSHAL_MSG_BODY_ERROR,
	ErrGetBlockError:      netPB.XuperMessage_GET_BLOCK_ERROR,
	ErrGetBlockChainError: netPB.XuperMessage_GET_BLOCKCHAIN_ERROR,
	ErrGetBlockChainStatusError: netPB.XuperMessage_CONFIRM_BLOCKCHAINSTATUS_ERROR,
}

func ErrorType(err error) netPB.XuperMessage_ErrorType {
	if err == nil {
		return netPB.XuperMessage_SUCCESS
	}

	if typ, ok := errorType[err]; ok {
		return typ
	}

	return netPB.XuperMessage_UNKNOW_ERROR
}

// error for chain pb XChainErrorEnum
var errorEnum = map[error]pb.XChainErrorEnum {
	ErrMessageParam: pb.XChainErrorEnum_VALIDATE_ERROR,
	ErrBlockChainNotExist: pb.XChainErrorEnum_BLOCKCHAIN_NOTEXIST,
	ErrBlockChainNotReady: pb.XChainErrorEnum_NOT_READY_ERROR,

	// verify
	ErrGasNotEnough: pb.XChainErrorEnum_GAS_NOT_ENOUGH_ERROR,
	ErrRWSetInvalid: pb.XChainErrorEnum_RWSET_INVALID_ERROR,
	ErrInvalidTxExt: pb.XChainErrorEnum_RWSET_INVALID_ERROR,
	ErrACLNotEnough: pb.XChainErrorEnum_RWACL_INVALID_ERROR,
	ErrVersionInvalid: pb.XChainErrorEnum_TX_VERSION_INVALID_ERROR,
	ErrInvalidSignature: pb.XChainErrorEnum_TX_SIGN_ERROR,
	ErrVerification: pb.XChainErrorEnum_TX_VERIFICATION_ERROR,

	// utxo
	ErrAlreadyInUnconfirmed: pb.XChainErrorEnum_UTXOVM_ALREADY_UNCONFIRM_ERROR,
	ErrNoEnoughUTXO: pb.XChainErrorEnum_NOT_ENOUGH_UTXO_ERROR,
	ErrUTXONotFound: pb.XChainErrorEnum_UTXOVM_NOT_FOUND_ERROR,
	ErrInputOutputNotEqual: pb.XChainErrorEnum_INPUT_OUTPUT_NOT_EQUAL_ERROR,
	ErrTxNotFound: pb.XChainErrorEnum_TX_NOT_FOUND_ERROR,
	ErrTxSizeLimitExceeded: pb.XChainErrorEnum_TX_SLE_ERROR,
	ErrRWSetInvalid: pb.XChainErrorEnum_RWSET_INVALID_ERROR,

	// ledger
	ErrRootBlockAlreadyExist: pb.XChainErrorEnum_ROOT_BLOCK_EXIST_ERROR,
	ErrTxDuplicated: pb.XChainErrorEnum_TX_DUPLICATE_ERROR,
	ErrTxNotFound: pb.XChainErrorEnum_TX_NOT_FOUND_ERROR,
}

func ErrorEnum(err error) pb.XChainErrorEnum {
	if err == nil {
		return pb.XChainErrorEnum_SUCCESS
	}

	if enum, ok := errorEnum[err]; ok {
		return enum
	}

	return pb.XChainErrorEnum_UNKNOW_ERROR
}

