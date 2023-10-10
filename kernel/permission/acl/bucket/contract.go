package bucket

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
)

// ContractBucket can access DB bucket used for contract
type ContractBucket struct {
	DB contract.XMState
}

// SetMethodACL stores contract method's ACL in DB with:
//
//	key: <ExtUtxoPrefix><ContractBucketPrefix>/<Contract><Separator><Method>
//	value: ACL
func (b *ContractBucket) SetMethodACL(contract, method string, acl []byte) error {
	key := utils.MakeContractMethodKey(contract, method)
	return b.DB.Put(utils.GetContractBucket(), []byte(key), acl)
}
