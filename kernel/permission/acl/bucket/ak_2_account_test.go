package bucket

import (
	"reflect"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
)

var (
	bucketValue = []byte("true")
	testAccount = "XC1111111111111111@xuper"
)

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
					AK2Account: bucketData{
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
				DB: &mockContext{AK2Account: bucketData{}},
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
				DB: &mockContext{AK2Account: bucketData{}},
			},
			args: args{
				account: testAccount,
				oldAKs:  []string{"AK_delete_error"},
				newAKs:  []string{"AK1", "AK2"},
			},
			wantErr: true,
		},
		{
			name: "put error",
			fields: fields{
				DB: &mockContext{AK2Account: bucketData{}},
			},
			args: args{
				account: testAccount,
				oldAKs:  nil,
				newAKs:  []string{"AK1", "AK_put_error"},
			},
			wantErr: true,
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
			got := tt.fields.DB.(*mockContext).AK2Account
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AK2AccountBucket.UpdateForAccount(), DB = %v, want %v", got, tt.want)
			}
		})
	}
}
