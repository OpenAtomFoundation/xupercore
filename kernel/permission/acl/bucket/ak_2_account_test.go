package bucket

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	pb "github.com/xuperchain/xupercore/protos"
)

var (
	bucketValue = []byte("true")
	testAccount = "XC1111111111111111@xuper"
)

type bucketData = map[string][]byte

func TestAK2AccountBucket_UpdateForAccount(t *testing.T) {
	type fields struct {
		DB contract.XMState
	}
	type args struct {
		account string
		oldAKs  []string
		newAKs  []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    bucketData
	}{

		{
			name: "update with overlap",
			fields: fields{
				DB: &mockContext{
					Data: bucketData{
						utils.MakeAK2AccountKey("AK1", testAccount): bucketValue,
						utils.MakeAK2AccountKey("AK2", testAccount): bucketValue,
					},
				},
			},
			args: args{
				account: testAccount,
				oldAKs:  []string{"AK1", "AK2"},
				newAKs:  []string{"AK2", "AK3"},
			},
			want: bucketData{
				utils.MakeAK2AccountKey("AK2", testAccount): bucketValue,
				utils.MakeAK2AccountKey("AK3", testAccount): bucketValue,
			},
		},
		{
			name: "add",
			fields: fields{
				DB: &mockContext{Data: bucketData{}},
			},
			args: args{
				account: testAccount,
				oldAKs:  nil,
				newAKs:  []string{"AK1", "AK2"},
			},
			want: bucketData{
				utils.MakeAK2AccountKey("AK1", testAccount): bucketValue,
				utils.MakeAK2AccountKey("AK2", testAccount): bucketValue,
			},
		},
		{
			name: "delete error",
			fields: fields{
				DB: &mockContext{Data: bucketData{}},
			},
			args: args{
				account: testAccount,
				oldAKs:  []string{"AK_delete_error"},
				newAKs:  []string{"AK1", "AK2"},
			},
			wantErr: true,
			want:    bucketData{},
		},
		{
			name: "put error",
			fields: fields{
				DB: &mockContext{Data: bucketData{}},
			},
			args: args{
				account: testAccount,
				oldAKs:  nil,
				newAKs:  []string{"AK1", "AK_put_error"},
			},
			wantErr: true,
			want: bucketData{
				utils.MakeAK2AccountKey("AK1", testAccount): bucketValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &AK2AccountBucket{
				DB: tt.fields.DB,
			}
			if err := b.UpdateForAccount(tt.args.account, tt.args.oldAKs, tt.args.newAKs); (err != nil) != tt.wantErr {
				t.Errorf("AK2AccountBucket.UpdateForAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got := tt.fields.DB.(*mockContext).Data
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AccountBucket.UpdateForAccount(), DB = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockContext struct {
	Data bucketData
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
	panic("implement me")
}

func (m mockContext) Select(bucket string, startKey []byte, endKey []byte) (contract.Iterator, error) {
	panic("implement me")
}

func (m mockContext) Put(bucket string, key, value []byte) error {
	k := string(key)
	if strings.HasPrefix(k, "AK_put_error") {
		return errors.New(k)
	}
	m.Data[string(key)] = value
	fmt.Println("Put: ", "bucket", bucket, "key", k, "value", string(value))
	return nil
}

func (m mockContext) Del(bucket string, key []byte) error {
	k := string(key)
	if strings.HasPrefix(k, "AK_delete_error") {
		return errors.New(k)
	}
	delete(m.Data, k)
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
