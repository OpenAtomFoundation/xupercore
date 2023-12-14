package bls

import (
	"reflect"
	"testing"
)

const (
	testAccount1 = `{"index":"5068161082419015240502520392764951298362816233167274727352107679212581830087892411256674283175582004835800509618391184401826892729061875636925728556009094","public_key":{"p":[164,221,255,34,120,66,140,120,163,230,208,249,96,42,222,231,194,36,30,244,137,96,213,46,176,93,46,89,68,243,12,96,47,240,60,88,234,72,58,247,142,105,57,139,4,34,199,166,21,218,230,130,179,163,111,135,245,255,146,42,84,213,78,12,203,60,198,79,49,4,168,66,220,234,78,122,153,164,15,62,154,215,52,45,226,9,170,8,82,26,241,74,132,7,27,158]},"private_key":{"x":[126,216,212,87,252,248,31,247,22,253,251,174,250,126,133,143,126,198,80,240,71,33,187,76,120,15,42,84,195,63,242,10]}}`
	testAccount2 = `{"index":"257006940103443962868532947373235556938271693750136125808684034232840884830638208732593735565637276170886442912667767381105455605885467612946853679655161","public_key":{"p":[169,63,130,44,124,41,185,231,49,229,169,230,236,191,119,183,4,112,139,183,168,59,194,72,98,180,198,46,210,24,250,22,148,209,179,146,156,26,249,90,15,187,58,205,155,3,160,80,6,42,138,63,246,4,214,33,158,33,27,171,56,221,230,3,26,237,14,14,2,118,110,250,255,90,48,117,217,188,141,5,17,2,51,250,168,16,86,155,139,33,151,233,182,100,29,132]},"private_key":{"x":[191,221,254,233,125,198,203,133,10,143,8,111,70,134,181,224,108,192,24,15,179,45,55,110,159,120,218,127,214,121,57,16]}}`
	testAccount3 = `{"index":"1368445478336047788671563167816769065025819906297195306486206640016117886676338763178411004319413299193570518260109002737706123789987243019974525949913989","public_key":{"p":[184,218,106,191,140,15,62,11,237,138,34,150,245,179,186,91,63,162,67,110,202,20,162,205,58,212,132,189,215,247,6,53,236,133,53,9,194,85,222,183,0,8,255,214,152,3,243,169,7,19,149,12,36,48,231,150,100,133,58,242,212,122,201,127,57,216,170,26,7,150,6,140,38,199,56,134,94,195,102,91,237,165,77,145,238,189,125,173,203,233,14,246,157,165,101,94]},"private_key":{"x":[167,8,106,250,211,151,167,45,169,99,223,47,114,152,242,220,254,243,216,36,205,110,87,211,42,64,177,205,71,244,232,32]}}`

	testIndex1 = "5068161082419015240502520392764951298362816233167274727352107679212581830087892411256674283175582004835800509618391184401826892729061875636925728556009094"
)

var testAccounts = []string{testAccount1, testAccount2, testAccount3}

func TestNewAccount(t *testing.T) {
	account, err := NewAccount()
	if err != nil {
		t.Fatal(err)
	}
	if account.Index == "" {
		t.Fatalf("Expected non-empty account index, got %q", account.Index)
	}
	if len(account.PublicKey) == 0 {
		t.Fatalf("Expected non-empty account public key, got %q", account.PublicKey)
	}
	if len(account.PrivateKey) == 0 {
		t.Fatalf("Expected non-empty account private key, got %q", account.PrivateKey)
	}
	t.Logf("Account: %+v", account)
}

func TestNewAccountFromJson(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		args    args
		want    *Account
		wantErr bool
	}{
		{
			name: "valid account json",
			args: args{
				data: testAccount1,
			},
			want: &Account{
				Index: testIndex1,
			},
			wantErr: false,
		},
		{
			name: "invalid account json",
			args: args{
				data: `{"index":16,"public_key":"9D5F6A67D4B4A564C16CCF0F8D60A8595","private_key":"B8A706C3DA498993EACD49474AABE8F7A6D"}`,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAccountFromJson(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewAccountFromJson() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.want != nil && !reflect.DeepEqual(got.Index, tt.want.Index) {
				t.Errorf("NewAccountFromJson() = %v, want %v", got, tt.want)
			}
		})
	}
}
