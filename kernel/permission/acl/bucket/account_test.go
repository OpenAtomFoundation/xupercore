package bucket

import (
	"reflect"
	"testing"

	"github.com/xuperchain/xupercore/kernel/contract"
)

func TestAccountBucket_IsExist(t *testing.T) {
	type args struct {
		account string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "not exist",
			args: args{
				account: "Account_get_not_found",
			},
			want: false,
		},
		{
			name: "other error",
			args: args{
				account: "Account_get_error_other",
			},
			wantErr: true,
		},
		{
			name: "exist",
			args: args{
				account: "Account_exist",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &AccountBucket{
				DB: mockContext{},
			}
			got, err := b.IsExist(tt.args.account)
			if (err != nil) != tt.wantErr {
				t.Errorf("AccountBucket.IsExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AccountBucket.IsExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccountBucket_SetAccountACL(t *testing.T) {
	type fields struct {
		DB contract.XMState
	}
	type args struct {
		account string
		acl     []byte
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
				account: "Account_succ",
				acl:     []byte("acl"),
			},
			want: bucketData{
				"Account_succ": []byte("acl"),
			},
		},
		{
			name: "fail",
			args: args{
				account: "Account_put_error",
				acl:     []byte("acl"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &AccountBucket{
				DB: mockContext{
					Account: bucketData{},
				},
			}
			if err := b.SetAccountACL(tt.args.account, tt.args.acl); (err != nil) != tt.wantErr {
				t.Errorf("AccountBucket.SetAccountACL() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := b.DB.(mockContext).Account
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want){
				t.Errorf("AccountBucket.SetAccountACL(), DB = %v, want %v", got, tt.want)
			}
		})
	}
}
