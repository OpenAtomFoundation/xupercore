// 交易处理
package xuper

import ()

type TxProcessor interface {
	VerifyTx(tx *pb.TxStatus) (bool, error)
	SubmitTx(tx *pb.TxStatus) error
}

type TxProcImpl struct {
}

func (t *TxProcImpl) VerifyTx(tx *pb.TxStatus) (bool, error) {

}

func (t *TxProcImpl) SubmitTx(tx *pb.TxStatus) error {

}
