package xrandom

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/OpenAtomFoundation/xupercore/crypto-dll-go/bls"
	"github.com/stretchr/testify/assert"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/common/xconfig"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"
)

func newMockNodeCtx(node string) *Context {
	return newTestNodeCtx("../../../../data/mock/" + node + "/data/keys")
}

func newTestNodeCtx(keyDir string) *Context {
	return &Context{
		ChainCtx: &common.ChainCtx{
			EngCtx: &common.EngineCtx{
				EnvCfg: &xconfig.EnvConf{
					KeyDir: keyDir,
				},
			},
		},
	}
}

func Test_loadAccount(t *testing.T) {
	type args struct {
		ctx *Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "mock node 1",
			args: args{
				ctx: newMockNodeCtx("node1"),
			},
		},
		{
			name: "mock node 2",
			args: args{
				ctx: newMockNodeCtx("node2"),
			},
		},
		{
			name: "mock node 3",
			args: args{
				ctx: newMockNodeCtx("node3"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := loadBlsAccount(tt.args.ctx)
			if err != nil {
				t.Error(err)
			}
			t.Log(account)
			assert.NotNil(t, account)
			assert.Greater(t, len(account.Index), 0)
			assert.Greater(t, len(account.PublicKey), 0)
			assert.Greater(t, len(account.PrivateKey), 0)
		})
	}
}

func Test_saveBlsAccount(t *testing.T) {
	type args struct {
		account *bls.Account
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "no account error",
			args: args{
				account: &bls.Account{},
			},
		},
		{
			name: "save account error",
			args: args{
				account: &bls.Account{
					Index:      "test",
					PublicKey:  []byte("test"),
					PrivateKey: []byte("test"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := saveBlsAccount(newTestNodeCtx("./testdata/keys"), tt.args.account); (err != nil) != tt.wantErr {
				t.Errorf("saveBlsAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_saveEthAccount(t *testing.T) {
	account, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf(err.Error())
	}
	type args struct {
		account *ecdsa.PrivateKey
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "save account normal",
			args: args{
				account: account,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := saveEthAccount(newTestNodeCtx("./testdata/keys"), tt.args.account); (err != nil) != tt.wantErr {
				t.Errorf("saveEthAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_loadEthAccount(t *testing.T) {
	type args struct {
		ctx *Context
	}
	tests := []struct {
		name    string
		args    args
		want    *ecdsa.PrivateKey
		wantErr bool
	}{
		{
			name: "Test random loadEthAccount",
			args: args{
				ctx: newTestNodeCtx("./testdata/keys"),
			},
			wantErr: false,
		},
		{
			name: "Test mock loadEthAccount",
			args: args{
				ctx: newMockNodeCtx("node1"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadEthAccount(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadEthAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			publicKey := got.Public().(*ecdsa.PublicKey)
			t.Logf("public key object: %s", publicKey)
			t.Logf("public key string: %#x", crypto.CompressPubkey(publicKey))
		})
	}
}

//37f05aac8ec6cbcad4b8bc936aee56c7b1a1f551c263f0bb5e75bd8672b3ef72

//acc360f32d9215f0b29c77f3fd2a7674651b33f0f29a2e86914deedcaabf8181
//ef305c06443bc9322eae2cc0c7c92682
//d7d7ab6163c13d25267f2091e449748ca28e9dfa8c03c9be5a6a5a104c6841c6
//a9935aa36dedff047bec0003bf1a819cf6fbcb5c9c289bb1d03a5970bd93a0ad

//aa871204fd8b75de7e68b176baa65afa3b52d718279925deed4d01d0cf845c40
//37f05aac8ec6cbcad4b8bc936aee56c7b1a1f551c263f0bb5e75bd8672b3ef72
//b1acbef26ca447c3043f6140f7547923736e2f1da06f0c1d3f7a21a9f1702ad0
//9f73af0df22d35d8090f4d7a955334325747225e5f8efe7df1b3e52cfd46b32c
