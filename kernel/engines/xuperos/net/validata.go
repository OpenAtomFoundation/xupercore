package xuperos

import (
	"errors"

	"github.com/golang/protobuf/proto"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
)

var (
	// ErrBlockChainNameEmpty is returned when blockchain name is empty
	ErrBlockChainNameEmpty = errors.New("validation error: validatePostTx TxStatus.Bcname can't be null")
	// ErrTxNil is returned when tx is nil
	ErrTxNil = errors.New("validation error: validatePostTx TxStatus.Tx can't be null")
	// ErrBlockIDNil is returned when blockid is nil
	ErrBlockIDNil = errors.New("validation error: validateSendBlock Block.Blockid can't be null")
	// ErrBlockNil is returned when block is nil
	ErrBlockNil = errors.New("validation error: validateSendBlock Block.Block can't be null")
	// ErrTxInvalid is returned when tx invaild
	ErrTxInvalid = errors.New("validation error: tx info is invaild")
)

func validatePostTx(tx *lpb.Transaction) error {
	if tx == nil || len(tx.Txid) == 0 {
		return ErrTxNil
	}

	// 为了兼容pb和json序列化时，对于空byte数组的处理行为不同导致txid计算错误的问题
	// 先对输入参数统一做一次序列化，防止交易被打包入块，utxoVM校验不通过，阻塞walk
	// 可能会导致一些语言的sdk受影响，需要在计算txid时统一把空byte数组明确置null处理
	prtBuf, err := proto.Marshal(tx)
	if err != nil {
		return ErrTxInvalid
	}
	err = proto.Unmarshal(prtBuf, tx)
	if err != nil {
		return ErrTxInvalid
	}

	return nil
}

func validateSendBlock(block *lpb.InternalBlock) error {
	if block == nil {
		return ErrBlockNil
	}

	if len(block.Blockid) == 0 {
		return ErrBlockIDNil
	}

	return nil
}
