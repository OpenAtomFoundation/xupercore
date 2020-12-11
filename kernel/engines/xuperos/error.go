package xuperos

import (
	"errors"
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
