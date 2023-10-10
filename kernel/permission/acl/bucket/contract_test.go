package bucket

import (
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"reflect"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
)

func TestContractBucket_SetMethodACL(t *testing.T) {
	type fields struct {
		DB contract.XMState
	}
	type args struct {
		contract string
		method   string
		acl      []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    bucketData
	}{
		{
			name: "success",
			args: args{
				contract: "contract",
				method:   "method",
				acl:      []byte("acl"),
			},
			want: bucketData{
				utils.MakeContractMethodKey("contract", "method"): []byte("acl"),
			},
		},
		{
			name: "fail",
			args: args{
				contract: "Contract_put_error",
				method:   "method",
				acl:      []byte("acl"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &ContractBucket{
				DB: mockContext{
					Contract: bucketData{},
				},
			}
			if err := b.SetMethodACL(tt.args.contract, tt.args.method, tt.args.acl); (err != nil) != tt.wantErr {
				t.Errorf("ContractBucket.SetMethodACL() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := b.DB.(mockContext).Contract
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ContractBucket.SetMethodACL(), DB = %v, want %v", got, tt.want)
			}
		})
	}
}
