package bucket

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
)

// AccountBucket can access DB bucket used for account
type AccountBucket struct {
	DB contract.XMState
}

// IsExist returns account existence in bucket
func (b *AccountBucket) IsExist(account string) (bool, error) {
	acl, err := b.GetAccountACL(account)
	if err != nil {
		if err == sandbox.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return acl != nil, nil
}

// GetAccountACL gets ACL by account name
func (b *AccountBucket) GetAccountACL(account string) ([]byte, error) {
	return b.DB.Get(utils.GetAccountBucket(), []byte(account))
}

// SetAccountACL stores account's ACL in DB with:
//
//	key: <ExtUtxoPrefix><AccountBucketPrefix>/<Account>
//	value: ACL
func (b *AccountBucket) SetAccountACL(account string, acl []byte) error {
	return b.DB.Put(utils.GetAccountBucket(), []byte(account), acl)
}
