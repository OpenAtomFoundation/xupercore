package xuperos

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/agiledragon/gomonkey/v2"

	_ "github.com/xuperchain/xupercore/bcs/consensus/pow"
	_ "github.com/xuperchain/xupercore/bcs/consensus/single"
	_ "github.com/xuperchain/xupercore/bcs/consensus/tdpos"
	_ "github.com/xuperchain/xupercore/bcs/consensus/xpoa"
	_ "github.com/xuperchain/xupercore/bcs/contract/evm"
	_ "github.com/xuperchain/xupercore/bcs/contract/native"
	_ "github.com/xuperchain/xupercore/bcs/contract/xvm"
	xledger "github.com/xuperchain/xupercore/bcs/ledger/xledger/utils"
	_ "github.com/xuperchain/xupercore/bcs/network/p2pv1"
	_ "github.com/xuperchain/xupercore/bcs/network/p2pv2"
	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	_ "github.com/xuperchain/xupercore/kernel/contract/manager"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/config"
	xnet "github.com/xuperchain/xupercore/kernel/engines/xuperos/net"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/parachain"
	"github.com/xuperchain/xupercore/kernel/mock"
	_ "github.com/xuperchain/xupercore/lib/crypto/client"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
)

func CreateLedger(conf *xconf.EnvConf) error {
	mockConf, err := mock.NewEnvConfForTest()
	if err != nil {
		return fmt.Errorf("new mock env conf error: %v", err)
	}

	genesisPath := mockConf.GenDataAbsPath("genesis/xuper.json")
	err = xledger.CreateLedger("xuper", genesisPath, conf)
	if err != nil {
		log.Printf("create ledger failed.err:%v\n", err)
		return fmt.Errorf("create ledger failed")
	}
	return nil
}

func RemoveLedger(conf *xconf.EnvConf) {
	path := conf.GenDataAbsPath("blockchain")
	if err := os.RemoveAll(path); err != nil {
		log.Printf("remove ledger failed.err:%v\n", err)
	}
}

func MockEngine(path string) (common.Engine, error) {
	conf, err := mock.NewEnvConfForTest(path)
	if err != nil {
		return nil, fmt.Errorf("new env conf error: %v", err)
	}

	RemoveLedger(conf)
	if err = CreateLedger(conf); err != nil {
		return nil, err
	}

	engine := NewEngine()
	if err := engine.Init(conf); err != nil {
		return nil, fmt.Errorf("init engine error: %v", err)
	}

	eng, err := EngineConvert(engine)
	if err != nil {
		return nil, fmt.Errorf("engine convert error: %v", err)
	}

	return eng, nil
}

type mockLogger struct {
}

func (m mockLogger) GetLogId() string {
	panic("implement me")
}

func (m mockLogger) SetCommField(key string, value interface{}) {
	panic("implement me")
}

func (m mockLogger) SetInfoField(key string, value interface{}) {
	panic("implement me")
}

func (m mockLogger) Error(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
}

func (m mockLogger) Warn(msg string, ctx ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Info(msg string, ctx ...interface{}) {
	panic("implement me")
}

func (m mockLogger) Trace(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
}

func (m mockLogger) Debug(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
}

func TestEngine_loadChains(t *testing.T) {
	conf, patch := setup(t)
	defer patch.Reset()

	type fields struct {
		engCtx    *common.EngineCtx
		netEvent  *xnet.NetEvent
		relyAgent common.EngineRelyAgent
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "mock data",
			fields: fields{
				engCtx: &common.EngineCtx{
					EnvCfg: conf,
					EngCfg: &config.EngineConf{
						RootChain: "xuper",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Engine{
				engCtx:    tt.fields.engCtx,
				log:       new(mockLogger),
				netEvent:  tt.fields.netEvent,
				relyAgent: tt.fields.relyAgent,
			}
			if err := tr.loadChains(); (err != nil) != tt.wantErr {
				t.Errorf("Engine.loadChains() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockRootChainReader struct {
}

func (m mockRootChainReader) Get(_ string, key []byte) ([]byte, error) {
	switch string(key) {
	case "ErrNotFount":
		return nil, errors.New("not found")
	case "ErrOther":
		return nil, errors.New("other")
	case "Disable":
		disabledGroup := parachain.Group{Status: parachain.ParaChainStatusStop}
		return json.Marshal(disabledGroup)
	default:
		enabledGroup := parachain.Group{Status: parachain.ParaChainStatusStart}
		return json.Marshal(enabledGroup)
	}
}

func TestEngine_tryLoadParaChain(t *testing.T) {
	conf, patch := setup(t)
	defer patch.Reset()

	type fields struct {
		netEvent  *xnet.NetEvent
		relyAgent common.EngineRelyAgent
	}
	type args struct {
		chainName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "paraChain loaded",
			args: args{
				chainName: "xuper",
			},
			want: true,
		},
		{
			name: "paraChain group not fount in rootChain info",
			args: args{
				chainName: "ErrNotFount",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "get paraChain group error",
			args: args{
				chainName: "ErrOther",
			},
			want: false,
		},
		{
			name: "paraChain disabled",
			args: args{
				chainName: "Disable",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Engine{
				engCtx: &common.EngineCtx{
					EnvCfg: conf,
					EngCfg: &config.EngineConf{
						RootChain: "non-xuper",
					},
				},
				log:       new(mockLogger),
				netEvent:  tt.fields.netEvent,
				relyAgent: tt.fields.relyAgent,
			}
			got, err := tr.tryLoadParaChain(conf.GenDataAbsPath(conf.ChainDir),
				tt.args.chainName,
				new(mockRootChainReader))
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.tryLoadParaChain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Engine.tryLoadParaChain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func setup(t *testing.T) (*xconf.EnvConf, *gomonkey.Patches) {
	conf, err := mock.NewEnvConfForTest("p2pv2/node1/conf/env.yaml")
	if err != nil {
		t.Fatalf("new env conf error: %v", err)
	}

	RemoveLedger(conf)
	if err = CreateLedger(conf); err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS == "darwin" {
		t.Skip()
	}
	mockLookPath := func(arg string) (string, error) {
		if arg == "wasm2c" {
			wasm2cPath := filepath.Join(filepath.Dir(os.Args[0]), "wasm2c")
			fmt.Println(filepath.Dir(os.Args[0]))
			return filepath.Abs(wasm2cPath)
		}
		return exec.LookPath(arg)
	}
	patch := gomonkey.ApplyFunc(exec.LookPath, mockLookPath)
	return conf, patch
}
