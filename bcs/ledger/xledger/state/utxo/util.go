package utxo

import (
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"github.com/xuperchain/xupercore/protos"
)

// queryContractStatData query stat data about contract, such as total contract and total account
func (uv *UtxoVM) queryContractStatData(bucket string) (int64, error) {
	dataCount := int64(0)
	prefixKey := pb.ExtUtxoTablePrefix + bucket + "/"
	it := uv.ldb.NewIteratorWithPrefix([]byte(prefixKey))
	defer it.Release()

	for it.Next() {
		dataCount++
	}
	if it.Error() != nil {
		return int64(0), it.Error()
	}

	return dataCount, nil
}

func (uv *UtxoVM) QueryContractStatData() (*protos.ContractStatData, error) {

	accountCount, accountCountErr := uv.queryContractStatData(utils.GetAccountBucket())
	if accountCountErr != nil {
		return &protos.ContractStatData{}, accountCountErr
	}

	contractCount, contractCountErr := uv.queryContractStatData(utils.GetContract2AccountBucket())
	if contractCountErr != nil {
		return &protos.ContractStatData{}, contractCountErr
	}

	data := &protos.ContractStatData{
		AccountCount:  accountCount,
		ContractCount: contractCount,
	}

	return data, nil
}
