package utils

import (
	"testing"
)

func TestValidAccountNumber(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{
			name:    "valid",
			args:    "1234567890123456",
			wantErr: false,
		},
		{
			name:    "empty",
			args:    "",
			wantErr: true,
		},
		{
			name:    "wrong length",
			args:    "1234567890",
			wantErr: true,
		},
		{
			name:    "invalid character",
			args:    "123456789012345a",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidAccountNumber(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("ValidAccountNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseAddressType(t *testing.T) {
	tests := []struct {
		name            string
		args            string
		wantAddressType AddressType
		wantIsValid     bool
	}{
		{
			name:        "invalid",
			args:        "",
			wantIsValid: false,
		},
		{
			name:            "AK",
			args:            "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
			wantAddressType: AddressAK,
			wantIsValid:     true,
		},
		{
			name:            "account",
			args:            "XC1111111111111111@xuper",
			wantAddressType: AddressAccount,
			wantIsValid:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAddressType, gotIsValid := ParseAddressType(tt.args)
			if gotAddressType != tt.wantAddressType {
				t.Errorf("ParseAddressType() gotAddressType = %v, want %v", gotAddressType, tt.wantAddressType)
			}
			if gotIsValid != tt.wantIsValid {
				t.Errorf("ParseAddressType() gotIsValid = %v, want %v", gotIsValid, tt.wantIsValid)
			}
		})
	}
}

func TestIsAccount(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		{
			name: "invalid",
			args: "",
			want: false,
		},
		{
			name: "AK",
			args: "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
			want: false,
		},
		{
			name: "account",
			args: "XC1111111111111111@xuper",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAccount(tt.args); got != tt.want {
				t.Errorf("IsAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAK(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		{
			name: "invalid",
			args: "",
			want: false,
		},
		{
			name: "AK",
			args: "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
			want: true,
		},
		{
			name: "account",
			args: "XC1111111111111111@xuper",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAK(tt.args); got != tt.want {
				t.Errorf("IsAK() = %v, want %v", got, tt.want)
			}
		})
	}
}
