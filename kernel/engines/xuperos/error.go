package xuperos

import (
	"errors"
	"github.com/xuperchain/xuperchain/core/pb"
)

var (
	ErrParamError = errors.New("param error")

	// blockchain
	ErrBlockChainExist    = errors.New("blockchain already exists")
	ErrBlockChainNotExist = errors.New("blockchain not exist")
	ErrRootChainNotExist  = errors.New("root chain not exist")

	ErrReadDataDirError = errors.New("read data dir error")
	ErrLoadChainError   = errors.New("load chain error")

	// ErrBlockExist used to return the error while block already exit
	ErrBlockExist = errors.New("block already exists")

	// tx
	ErrTxDuplicate = errors.New("tx duplicate")
	ErrTxIdNil     = errors.New("tx id nil")

	// ErrServiceRefused used to return the error while service refused
	ErrServiceRefused = errors.New("service refused")
	// ErrConfirmBlock used to return the error while confirm block error
	ErrConfirmBlock = errors.New("confirm block error")
	// ErrCreateBlockChain is returned when create block chain error
	ErrCreateBlockChain = errors.New("create block chain error")
)

func HandleVerifyError(err error) pb.XChainErrorEnum {
	switch err {
	case utxo.ErrGasNotEnough:
		return pb.XChainErrorEnum_GAS_NOT_ENOUGH_ERROR
	case utxo.ErrRWSetInvalid, utxo.ErrInvalidTxExt:
		return pb.XChainErrorEnum_RWSET_INVALID_ERROR
	case utxo.ErrACLNotEnough:
		return pb.XChainErrorEnum_RWACL_INVALID_ERROR
	case utxo.ErrVersionInvalid:
		return pb.XChainErrorEnum_TX_VERSION_INVALID_ERROR
	case utxo.ErrInvalidSignature:
		return pb.XChainErrorEnum_TX_SIGN_ERROR
	case ErrTxDuplicate:
		return pb.XChainErrorEnum_TX_DUPLICATE_ERROR
	default:
		return pb.XChainErrorEnum_TX_VERIFICATION_ERROR
	}
}

// HandleStateError used to handle error of state
func HandleStateError(err error) pb.XChainErrorEnum {
	switch err {
	case utxo.ErrAlreadyInUnconfirmed:
		return pb.XChainErrorEnum_UTXOVM_ALREADY_UNCONFIRM_ERROR
	case utxo.ErrNoEnoughUTXO:
		return pb.XChainErrorEnum_NOT_ENOUGH_UTXO_ERROR
	case utxo.ErrUTXONotFound:
		return pb.XChainErrorEnum_UTXOVM_NOT_FOUND_ERROR
	case utxo.ErrInputOutputNotEqual:
		return pb.XChainErrorEnum_INPUT_OUTPUT_NOT_EQUAL_ERROR
	case utxo.ErrTxNotFound:
		return pb.XChainErrorEnum_TX_NOT_FOUND_ERROR
	case utxo.ErrTxSizeLimitExceeded:
		return pb.XChainErrorEnum_TX_SLE_ERROR
	case utxo.ErrRWSetInvalid:
		return pb.XChainErrorEnum_RWSET_INVALID_ERROR
	default:
		return pb.XChainErrorEnum_UNKNOW_ERROR
	}
}

// HandleLedgerError used to handle error of ledger
func HandleLedgerError(err error) pb.XChainErrorEnum {
	switch err {
	case ledger.ErrRootBlockAlreadyExist:
		return pb.XChainErrorEnum_ROOT_BLOCK_EXIST_ERROR
	case ledger.ErrTxDuplicated:
		return pb.XChainErrorEnum_TX_DUPLICATE_ERROR
	case ledger.ErrTxNotFound:
		return pb.XChainErrorEnum_TX_NOT_FOUND_ERROR
	default:
		return pb.XChainErrorEnum_UNKNOW_ERROR
	}
}
