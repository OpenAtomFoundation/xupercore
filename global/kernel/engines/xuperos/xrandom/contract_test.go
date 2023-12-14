package xrandom

import "testing"

func Test_publicKeyToAddress(t *testing.T) {
	type args struct {
		compressed string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test valid compressed public key",
			args: args{
				compressed: "0x037518a65fb5fb4a713f1ed346271636be885d27bfdda553e8c5c9f0b2e70fddc8",
			},
			want: "0xa416d967996b6C1de929C2A07e313f6EC9973853",
		},
		{
			name: "test invalid compressed public key",
			args: args{
				compressed: "0x037518a65fb5fb4a713f1ed346271636",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := publicKeyToAddress(tt.args.compressed)
			if (err != nil) != tt.wantErr {
				t.Errorf("publicKeyToAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("publicKeyToAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
