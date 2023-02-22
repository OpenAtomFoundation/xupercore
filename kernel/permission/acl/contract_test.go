package acl

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	pb "github.com/xuperchain/xupercore/protos"
)

var (
	testAccountNumber = []byte("1111111111111111")
	testBcName        = "xuper"
	testAccountName   = utils.MakeAccountKey(testBcName, string(testAccountNumber))
	valueTrue         = []byte("true")
)

type bucketData map[string][]byte
type argsData map[string][]byte

type mockContext struct {
	resourceLimit contract.Limits
	args          argsData

	// DB data
	accounts   bucketData
	ak2account bucketData
	contract   bucketData
}

func (m mockContext) getDBData() map[string]bucketData {
	tmp := map[string]bucketData{
		utils.GetAccountBucket():    m.accounts,
		utils.GetAK2AccountBucket(): m.ak2account,
		utils.GetContractBucket():   m.contract,
	}
	data := make(map[string]bucketData)
	for key, value := range tmp {
		if value != nil {
			data[key] = value
		}
	}
	return data
}

func (m mockContext) Args() map[string][]byte {
	return m.args
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
	if k == "" {
		return nil, sandbox.ErrNotFound
	} else if k == "XC9999999999999999@xuper" {
		return nil, errors.New("DB get error")
	}
	switch bucket {
	case utils.GetAccountBucket():
		return m.accounts[k], nil
	default:
		return nil, errors.New("unexpected bucket")
	}
}

func (m mockContext) Select(bucket string, startKey []byte, endKey []byte) (contract.Iterator, error) {
	panic("implement me")
}

func (m mockContext) Put(bucket string, key, value []byte) error {
	k := string(key)
	fmt.Println("Put: ", "bucket", bucket, "key", k, "value", string(value))

	switch bucket {
	case utils.GetAccountBucket():
		if k == "XC9999999999999998@xuper" {
			return errors.New("DB put account error")
		}
		m.accounts[k] = value
	case utils.GetAK2AccountBucket():
		if strings.HasSuffix(k, "XC9999999999999997@xuper") {
			return errors.New("DB put ak2account error")
		}
		m.ak2account[k] = value
	case utils.GetContractBucket():
		if strings.HasPrefix(k, "contractPutError") {
			return errors.New("DB put contract error")
		}
		m.contract[k] = value
	default:
		return errors.New("unexpected bucket")
	}
	return nil
}

func (m mockContext) Del(bucket string, key []byte) error {
	k := string(key)
	fmt.Println("Delete: ", "bucket", bucket, "key", k)

	switch bucket {
	case utils.GetAK2AccountBucket():
		delete(m.ak2account, k)
	default:
		return errors.New("unexpected bucket")
	}
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

func (m mockContext) AddResourceUsed(_ contract.Limits) {}

func (m mockContext) ResourceLimit() contract.Limits {
	return m.resourceLimit
}

func (m mockContext) Call(module, contract, method string, args map[string][]byte) (*contract.Response, error) {
	panic("implement me")
}

func (m mockContext) EmitAsyncTask(event string, args interface{}) error {
	panic("implement me")
}

func mockACL(rule pb.PermissionRule) []byte {
	var acl *pb.Acl
	switch rule {
	case pb.PermissionRule_SIGN_THRESHOLD:
		acl = &pb.Acl{
			Pm: &pb.PermissionModel{
				Rule: rule,
			},
			AksWeight: map[string]float64{
				"Threshold_AK_1": 0.5,
				"Threshold_AK_2": 0.4,
				"Overlap_AK":     0.1,
			},
		}
	case pb.PermissionRule_SIGN_AKSET:
		acl = &pb.Acl{
			Pm: &pb.PermissionModel{
				Rule: rule,
			},
			AkSets: &pb.AkSets{
				Sets: map[string]*pb.AkSet{
					"set_1": {
						Aks: []string{"AkSet_AK_1", "AkSet_AK_2"},
					},
					"set_2": {
						Aks: []string{"AkSet_AK_2", "AkSet_AK_3"},
					},
					"set_3": {
						Aks: []string{"Overlap_AK"},
					},
				},
			},
		}
	}
	data, err := json.Marshal(acl)
	if err != nil {
		return nil
	}
	return data
}

//func TestUpdateAK2AccountReflection(t *testing.T) {
//	type args struct {
//		ctx         contract.KContext
//		aclOldJSON  []byte
//		aclNewJSON  []byte
//		accountName string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "update between different rules",
//			args: args{
//				ctx:         new(mockContext),
//				aclOldJSON:  mockACL(pb.PermissionRule_SIGN_THRESHOLD),
//				aclNewJSON:  mockACL(pb.PermissionRule_SIGN_AKSET),
//				accountName: "account",
//			},
//		},
//		{
//			name: "add",
//			args: args{
//				ctx:         new(mockContext),
//				aclOldJSON:  nil,
//				aclNewJSON:  mockACL(pb.PermissionRule_SIGN_THRESHOLD),
//				accountName: "account",
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if err := UpdateAK2AccountReflection(tt.args.ctx, tt.args.aclOldJSON, tt.args.aclNewJSON, tt.args.accountName); (err != nil) != tt.wantErr {
//				t.Errorf("UpdateAK2AccountReflection() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

//func TestAK2AccountBucket_UpdateForAccount(t *testing.T) {
//	type fields struct {
//		DB contract.XMState
//	}
//	type args struct {
//		account string
//		oldAKs  []string
//		newAKs  []string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//
//		{
//			name: "update with overlap",
//			fields: fields{
//				DB: new(mockContext),
//			},
//			args: args{
//				account: "account",
//				oldAKs:  []string{"ak1", "ak2"},
//				newAKs:  []string{"ak2", "ak3"},
//			},
//		},
//		{
//			name: "add",
//			fields: fields{
//				DB: new(mockContext),
//			},
//			args: args{
//				account: "account",
//				oldAKs:  nil,
//				newAKs:  []string{"ak1", "ak2"},
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			b := &AK2AccountBucket{
//				DB: tt.fields.DB,
//			}
//			if err := b.UpdateForAccount(tt.args.account, tt.args.oldAKs, tt.args.newAKs); (err != nil) != tt.wantErr {
//				t.Errorf("AK2AccountBucket.UpdateForAccount() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

func TestKernMethod_NewAccount(t *testing.T) {
	type fields struct {
		BcName                   string
		NewAccountResourceAmount int64
	}
	type args struct {
		ctx contract.KContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *contract.Response
		wantDB  map[string]bucketData
		wantErr bool
	}{
		{
			name:   "fee not enough",
			fields: fields{NewAccountResourceAmount: 1000},
			args: args{
				mockContext{
					resourceLimit: contract.Limits{XFee: 0},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ACL",
			args: args{
				mockContext{
					args: argsData{
						"account_name": testAccountNumber,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid account name",
			args: args{
				mockContext{
					args: argsData{
						"acl": mockACL(pb.PermissionRule_SIGN_THRESHOLD),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no blockchain name",
			args: args{
				mockContext{
					args: argsData{
						"acl":          mockACL(pb.PermissionRule_SIGN_THRESHOLD),
						"account_name": testAccountNumber,
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "DB get error",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":          mockACL(pb.PermissionRule_SIGN_THRESHOLD),
						"account_name": []byte("9999999999999999"),
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "account exist",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":          mockACL(pb.PermissionRule_SIGN_THRESHOLD),
						"account_name": testAccountNumber,
					},
					accounts: bucketData{
						testAccountName: mockACL(pb.PermissionRule_SIGN_THRESHOLD),
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "DB put account error",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":          mockACL(pb.PermissionRule_SIGN_THRESHOLD),
						"account_name": []byte("9999999999999998"),
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "update ak2account error",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":          mockACL(pb.PermissionRule_SIGN_THRESHOLD),
						"account_name": []byte("9999999999999997"),
					},
					accounts: bucketData{},
				},
			},
			wantErr: true,
		},
		{
			name:   "create account succeed",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":          mockACL(pb.PermissionRule_SIGN_THRESHOLD),
						"account_name": testAccountNumber,
					},
					accounts:   bucketData{},
					ak2account: bucketData{},
				},
			},
			want: &contract.Response{
				Status:  contract.StatusOK,
				Message: "success",
				Body:    mockACL(pb.PermissionRule_SIGN_THRESHOLD),
			},
			wantDB: map[string]bucketData{
				utils.GetAccountBucket(): {
					testAccountName: mockACL(pb.PermissionRule_SIGN_THRESHOLD),
				},
				utils.GetAK2AccountBucket(): {
					utils.MakeAK2AccountKey("Threshold_AK_1", testAccountName): valueTrue,
					utils.MakeAK2AccountKey("Threshold_AK_2", testAccountName): valueTrue,
					utils.MakeAK2AccountKey("Overlap_AK", testAccountName):     valueTrue,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &KernMethod{
				BcName:                   tt.fields.BcName,
				NewAccountResourceAmount: tt.fields.NewAccountResourceAmount,
			}
			got, err := tr.NewAccount(tt.args.ctx)
			t.Logf("err: %s", err)
			if (err != nil) != tt.wantErr {
				t.Errorf("KernMethod.NewAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KernMethod.NewAccount() = %v, want %v", got, tt.want)
			}
			dbData := tt.args.ctx.(mockContext).getDBData()
			if !tt.wantErr && !reflect.DeepEqual(dbData, tt.wantDB) {
				t.Errorf("KernMethod.NewAccount(), DB = %v, want %v", dbData, tt.wantDB)
			}
		})
	}
}

func TestKernMethod_SetAccountACL(t *testing.T) {
	type fields struct {
		BcName                   string
		NewAccountResourceAmount int64
	}
	type args struct {
		ctx contract.KContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *contract.Response
		wantDB  map[string]bucketData
		wantErr bool
	}{

		{
			name:   "fee not enough",
			fields: fields{NewAccountResourceAmount: 1000},
			args: args{
				mockContext{
					resourceLimit: contract.Limits{XFee: 0},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ACL",
			args: args{
				mockContext{
					args: argsData{
						"account_name": []byte(testAccountName),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "account name not exist",
			args: args{
				mockContext{
					args: argsData{
						"acl": mockACL(pb.PermissionRule_SIGN_AKSET),
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "DB get error",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":          mockACL(pb.PermissionRule_SIGN_AKSET),
						"account_name": []byte("XC9999999999999999@xuper"),
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "update ak2account error",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":          mockACL(pb.PermissionRule_SIGN_AKSET),
						"account_name": []byte("XC9999999999999997@xuper"),
					},
					accounts: bucketData{
						"XC9999999999999997@xuper": mockACL(pb.PermissionRule_SIGN_THRESHOLD),
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "update account ACL: threshold -> akSets",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":          mockACL(pb.PermissionRule_SIGN_AKSET),
						"account_name": []byte(testAccountName),
					},
					accounts: bucketData{
						testAccountName: mockACL(pb.PermissionRule_SIGN_THRESHOLD),
					},
					ak2account: bucketData{
						utils.MakeAK2AccountKey("Threshold_AK_1", testAccountName): valueTrue,
						utils.MakeAK2AccountKey("Threshold_AK_2", testAccountName): valueTrue,
						utils.MakeAK2AccountKey("Overlap_AK", testAccountName):     valueTrue,
					},
				},
			},
			want: &contract.Response{
				Status:  contract.StatusOK,
				Message: "success",
				Body:    mockACL(pb.PermissionRule_SIGN_AKSET),
			},
			wantDB: map[string]bucketData{
				utils.GetAccountBucket(): {
					testAccountName: mockACL(pb.PermissionRule_SIGN_AKSET),
				},
				utils.GetAK2AccountBucket(): {
					utils.MakeAK2AccountKey("AkSet_AK_1", testAccountName): valueTrue,
					utils.MakeAK2AccountKey("AkSet_AK_2", testAccountName): valueTrue,
					utils.MakeAK2AccountKey("AkSet_AK_3", testAccountName): valueTrue,
					utils.MakeAK2AccountKey("Overlap_AK", testAccountName): valueTrue,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &KernMethod{
				BcName:                   tt.fields.BcName,
				NewAccountResourceAmount: tt.fields.NewAccountResourceAmount,
			}
			got, err := tr.SetAccountACL(tt.args.ctx)
			t.Logf("err: %s", err)
			if (err != nil) != tt.wantErr {
				t.Errorf("KernMethod.SetAccountACL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KernMethod.SetAccountACL() = %v, want %v", got, tt.want)
			}
			dbData := tt.args.ctx.(mockContext).getDBData()
			if !tt.wantErr && !reflect.DeepEqual(dbData, tt.wantDB) {
				t.Errorf("KernMethod.NewAccount(), DB = %v, want %v", dbData, tt.wantDB)
			}
		})
	}
}

func TestKernMethod_SetMethodACL(t *testing.T) {
	type fields struct {
		BcName                   string
		NewAccountResourceAmount int64
	}
	type args struct {
		ctx contract.KContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *contract.Response
		wantDB  map[string]bucketData
		wantErr bool
	}{
		{
			name:   "fee not enough",
			fields: fields{NewAccountResourceAmount: 1000},
			args: args{
				mockContext{
					resourceLimit: contract.Limits{XFee: 0},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid method",
			args: args{
				mockContext{
					args: argsData{
						"acl": mockACL(pb.PermissionRule_SIGN_AKSET),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ACL",
			args: args{
				mockContext{
					args: argsData{
						"contract_name": []byte("contract"),
						"method_name":   []byte("method"),
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "update contract error",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":           mockACL(pb.PermissionRule_SIGN_THRESHOLD),
						"contract_name": []byte("contractPutError"),
						"method_name":   []byte("method"),
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "update method ACL succeed",
			fields: fields{BcName: testBcName},
			args: args{
				mockContext{
					args: argsData{
						"acl":           mockACL(pb.PermissionRule_SIGN_THRESHOLD),
						"contract_name": []byte("contract"),
						"method_name":   []byte("method"),
					},
					contract: bucketData{},
				},
			},
			want: &contract.Response{
				Status:  contract.StatusOK,
				Message: "success",
				Body:    mockACL(pb.PermissionRule_SIGN_THRESHOLD),
			},
			wantDB: map[string]bucketData{
				utils.GetContractBucket(): {
					utils.MakeContractMethodKey("contract", "method"): mockACL(pb.PermissionRule_SIGN_THRESHOLD),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &KernMethod{
				BcName:                   tt.fields.BcName,
				NewAccountResourceAmount: tt.fields.NewAccountResourceAmount,
			}
			got, err := tr.SetMethodACL(tt.args.ctx)
			t.Logf("err: %s", err)
			if (err != nil) != tt.wantErr {
				t.Errorf("KernMethod.SetMethodACL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KernMethod.SetMethodACL() = %v, want %v", got, tt.want)
			}
			dbData := tt.args.ctx.(mockContext).getDBData()
			if !tt.wantErr && !reflect.DeepEqual(dbData, tt.wantDB) {
				t.Errorf("KernMethod.NewAccount(), DB = %v, want %v", dbData, tt.wantDB)
			}
		})
	}
}
