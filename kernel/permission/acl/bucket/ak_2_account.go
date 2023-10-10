package bucket

import (
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
)

// AK2AccountBucket can access DB bucket used for AK -> Account reflection
type AK2AccountBucket struct {
	DB contract.XMState
}

// UpdateForAccount update bucket for an account about its related AK list
// duplication is allowed for input AK list.
func (b *AK2AccountBucket) UpdateForAccount(account string, oldAKs, newAKs []string) error {

	// deduplicate AK
	deleteAKs, insertAKs := make(map[string]bool), make(map[string]bool)
	for _, ak := range oldAKs {
		deleteAKs[ak] = true
	}
	for _, ak := range newAKs {
		insertAKs[ak] = true
	}

	for _, ak := range oldAKs {
		// remove unchanged AK, which to be inserted after deleted
		if deleteAKs[ak] && insertAKs[ak] {
			delete(deleteAKs, ak)
			delete(insertAKs, ak)
		}
	}

	for ak := range deleteAKs {
		if err := b.delete(ak, account); err != nil {
			return err
		}
	}
	for ak := range insertAKs {
		if err := b.add(ak, account); err != nil {
			return err
		}
	}
	return nil
}

// delete removes AK -> Account info from DB
func (b *AK2AccountBucket) delete(ak string, account string) error {
	key := utils.MakeAK2AccountKey(ak, account)
	return b.DB.Del(utils.GetAK2AccountBucket(), []byte(key))
}

// add stores AK -> Account info in DB with:
//
//	key: <ExtUtxoPrefix><AK2AccountBucketPrefix>/<AK><Separator><Account>
//	value: useless yet
func (b *AK2AccountBucket) add(ak string, account string) error {
	key := utils.MakeAK2AccountKey(ak, account)
	return b.DB.Put(utils.GetAK2AccountBucket(), []byte(key), []byte("true"))
}
