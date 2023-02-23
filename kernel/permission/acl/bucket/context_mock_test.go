package bucket

import (
	"errors"
	"fmt"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"math/big"
	"strings"

	"github.com/xuperchain/xupercore/kernel/contract"
	pb "github.com/xuperchain/xupercore/protos"
)

type bucketData = map[string][]byte

type mockContext struct {
	AK2Account bucketData
	Account    bucketData
	Contract   bucketData
}

func (m mockContext) Args() map[string][]byte {
	panic("implement me")
}

func (m mockContext) Initiator() string {
	panic("implement me")
}

func (m mockContext) Caller() string {
	panic("implement me")
}

func (m mockContext) AuthRequire() []string {
	panic("implement me")
}

func (m mockContext) Get(bucket string, key []byte) ([]byte, error) {
	k := string(key)
	switch bucket {
	case utils.GetAccountBucket():
		if k == "Account_get_not_found" {
			return nil, sandbox.ErrNotFound
		} else if k == "Account_get_error_other" {
			return nil, errors.New(k)
		}
	default:
		return nil, errors.New("unexpected bucket")
	}
	fmt.Println("Get: ", "bucket", bucket, "key")
	return key, nil
}

func (m mockContext) Select(bucket string, startKey []byte, endKey []byte) (contract.Iterator, error) {
	panic("implement me")
}

func (m mockContext) Put(bucket string, key, value []byte) error {
	k := string(key)
	switch bucket {
	case utils.GetAK2AccountBucket():
		if strings.HasPrefix(k, "AK_put_error") {
			return errors.New(k)
		}
		m.AK2Account[k] = value
	case utils.GetAccountBucket():
		if k == "Account_put_error" {
			return errors.New(k)
		}
		m.Account[k] = value
	case utils.GetContractBucket():
		if strings.HasPrefix(k,"Contract_put_error") {
			return errors.New(k)
		}
		m.Contract[k] = value
	default:
		return errors.New("unexpected bucket")
	}
	fmt.Println("Put: ", "bucket", bucket, "key", k, "value", string(value))
	return nil
}

func (m mockContext) Del(bucket string, key []byte) error {
	k := string(key)
	switch bucket {
	case utils.GetAK2AccountBucket():

		if strings.HasPrefix(k, "AK_delete_error") {
			return errors.New(k)
		}
		delete(m.AK2Account, k)
	default:
		return errors.New("unexpected bucket")
	}
	fmt.Println("Delete: ", "bucket", bucket, "key", string(key))
	return nil
}

func (m mockContext) Transfer(from string, to string, amount *big.Int) error {
	panic("implement me")
}

func (m mockContext) AddEvent(events ...*pb.ContractEvent) {
	panic("implement me")
}

func (m mockContext) Flush() error {
	panic("implement me")
}

func (m mockContext) RWSet() *contract.RWSet {
	panic("implement me")
}

func (m mockContext) UTXORWSet() *contract.UTXORWSet {
	panic("implement me")
}

func (m mockContext) AddResourceUsed(delta contract.Limits) {
	panic("implement me")
}

func (m mockContext) ResourceLimit() contract.Limits {
	panic("implement me")
}

// used for test verification
func (m mockContext) Call(module, contract, method string, args map[string][]byte) (*contract.Response, error) {
	panic("implement me")
}

func (m mockContext) EmitAsyncTask(event string, args interface{}) error {
	panic("implement me")
}
