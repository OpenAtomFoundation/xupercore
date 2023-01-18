package utils

import (
	"testing"

	"github.com/tmthrgd/go-hex"

	pb "github.com/xuperchain/xupercore/protos"
)

// a set of data which can pass verification
var (
	testAk        = "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"
	testPublicKey = `{"Curvname":"P-256","X":36505150171354363400464126431978257855318414556425194490762274938603757905292,"Y":79656876957602994269528255245092635964473154458596947290316223079846501380076}`
	testSign      = hex.MustDecodeString("30450221009af3f6f448d4d855d97e5966e3302eb11b174fc4f1860cdd2e74be005a94c263022051e5a592b5d4acf284b080b1b007634938a632cb36837f479a9f297acf93d46e")
	testData      = []byte("test")
)

func TestIdentifyAK(t *testing.T) {
	type args struct {
		akURI string
		sign  *pb.SignatureInfo
		msg   []byte
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "verify sign pass",
			args: args{
				akURI: testAk,
				sign: &pb.SignatureInfo{
					PublicKey: testPublicKey,
					Sign:      testSign,
				},
				msg: testData,
			},
			want: true,
		},
		{
			name: "nil sign",
			args: args{
				akURI: testAk,
				sign:  nil,
				msg:   testData,
			},
			wantErr: true,
		},
		{
			name: "wrong ak",
			args: args{
				akURI: "XC1111111111111111@xuper/9LArZSMrrRorV7T6h5T32PVUrmdcYLbug",
				sign: &pb.SignatureInfo{
					PublicKey: testPublicKey,
					Sign:      testSign,
				},
				msg: testData,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IdentifyAK(tt.args.akURI, tt.args.sign, tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("IdentifyAK() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IdentifyAK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifySign(t *testing.T) {
	type args struct {
		ak   string
		si   *pb.SignatureInfo
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "verify pass",
			args: args{
				ak: testAk,
				si: &pb.SignatureInfo{
					PublicKey: testPublicKey,
					Sign:      testSign,
				},
				data: testData,
			},
			want: true,
		},
		{
			name: "verify ECDSA fail",
			args: args{
				ak: testAk,
				si: &pb.SignatureInfo{
					PublicKey: testPublicKey,
				},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "verify address fail",
			args: args{
				ak: testAk,
				si: &pb.SignatureInfo{
					PublicKey: `{"Curvname":"P-256","X":0,"Y":0}`,
				},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VerifySign(tt.args.ak, tt.args.si, tt.args.data)
			if err != nil {
				t.Logf("got err = %v", err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifySign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VerifySign() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractAkFromAuthRequire(t *testing.T) {
	tests := []struct {
		name        string
		authRequire string
		want        string
	}{
		{
			name:        "pass",
			authRequire: "XC1111111111111111@xuper/TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
			want:        "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractAkFromAuthRequire(tt.authRequire); got != tt.want {
				t.Errorf("ExtractAkFromAuthRequire() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractAddrFromAkURI(t *testing.T) {
	type args struct {
		akURI string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "AK",
			args: args{akURI: "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"},
			want: "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
		},
		{
			name: "account",
			args: args{akURI: "XC1111111111111111@xuper"},
			want: "XC1111111111111111@xuper",
		},
		{
			name: "auth requirement",
			args: args{akURI: "XC1111111111111111@xuper/TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"},
			want: "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
		},
		{
			name: "empty input",
			args: args{akURI: ""},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractAddrFromAkURI(tt.args.akURI)
			if got != tt.want {
				t.Errorf("ExtractAddrFromAkURI() = %v, want %v", got, tt.want)
			}
		})
	}
}
