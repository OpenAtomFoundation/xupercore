package xuperos

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/patrickmn/go-cache"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/common/xaddress"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/miner"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

var (
	adminTxId   = []byte(``)
	adminAmount = big.NewInt(0)
)

func init() {
	adminAmount, _ = big.NewInt(0).SetString("100000000000000000000", 10)
	adminTxId, _ = hex.DecodeString(`5aa155b99f5f405c6c05238abbc3163bd22d8452181405b3508d80b2ae646e0e`)
}

func mockTransferTx(chain common.Chain) (*lpb.Transaction, error) {
	conf := chain.Context().EngCtx.EnvCfg
	addr, err := xaddress.LoadAddrInfo(conf.GenDataAbsPath(conf.KeyDir), chain.Context().Crypto)
	if err != nil {
		return nil, err
	}

	amount, ok := big.NewInt(0).SetString("10000", 10)
	if !ok {
		return nil, fmt.Errorf("amount error")
	}

	change := big.NewInt(0).Sub(adminAmount, amount)

	tx := &lpb.Transaction{
		Version:     1,
		Coinbase:    false,
		Desc:        []byte(`mock transfer`),
		Nonce:       utils.GenNonce(),
		Timestamp:   time.Now().UnixNano(),
		Initiator:   addr.Address,
		AuthRequire: []string{addr.Address},
		TxInputs: []*protos.TxInput{
			{
				RefTxid:   adminTxId,
				RefOffset: 0,
				FromAddr:  []byte(addr.Address),
				Amount:    adminAmount.Bytes(),
			},
		},
		TxOutputs: []*protos.TxOutput{
			{
				ToAddr: []byte(addr.Address),
				Amount: change.Bytes(),
			}, {
				ToAddr: []byte(`SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co`),
				Amount: amount.Bytes(),
			},
		},
	}

	// 签名
	sign, err := txhash.ProcessSignTx(chain.Context().Crypto, tx, []byte(addr.PrivateKeyStr))
	if err != nil {
		return nil, err
	}
	signs := []*protos.SignatureInfo{
		{
			PublicKey: addr.PublicKeyStr,
			Sign:      sign,
		},
	}
	tx.InitiatorSigns = signs
	tx.AuthRequireSigns = signs

	// txID
	tx.Txid, err = txhash.MakeTransactionID(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func TestChain_SubmitTx_case_transfer(t *testing.T) {
	patch := setup(t)
	defer patch.Reset()

	engine, err := MockEngine("p2pv2/node1/conf/env.yaml")
	if err != nil {
		t.Fatalf("%v", err)
	}
	go engine.Run()
	defer engine.Exit()

	chain, err := engine.Get("xuper")
	if err != nil {
		t.Fatalf("get chain error: %v", err)
	}

	tx, err := mockTransferTx(chain)
	if err != nil {
		t.Fatalf("mock tx error: %v", err)
	}

	err = chain.SubmitTx(chain.Context(), tx)
	if err != nil {
		t.Fatalf("submit tx error: %v", err)
	}
}

func mockContractTx(chain common.Chain) (*lpb.Transaction, error) {
	conf := chain.Context().EngCtx.EnvCfg
	addr, err := xaddress.LoadAddrInfo(conf.GenDataAbsPath(conf.KeyDir), chain.Context().Crypto)
	if err != nil {
		return nil, err
	}

	reqs := []*protos.InvokeRequest{
		{
			ModuleName:   "xkernel",
			ContractName: "$acl",
			MethodName:   "NewAccount",
			Args: map[string][]byte{
				"account_name": []byte("1234567890123456"),
				"acl":          []byte(`{"pm": {"rule": 1,"acceptValue": 1.0},"aksWeight": {"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY": 1}}`),
			},
		},
	}
	response, err := chain.PreExec(chain.Context(), reqs, addr.Address, []string{addr.Address})
	if err != nil {
		return nil, err
	}

	amount := big.NewInt(response.GasUsed)
	change := big.NewInt(0).Sub(adminAmount, amount)

	tx := &lpb.Transaction{
		Version:     1,
		Coinbase:    false,
		Desc:        []byte(`mock contract`),
		Nonce:       utils.GenNonce(),
		Timestamp:   time.Now().UnixNano(),
		Initiator:   addr.Address,
		AuthRequire: []string{addr.Address},
		TxInputs: []*protos.TxInput{
			{
				RefTxid:   adminTxId,
				RefOffset: 0,
				FromAddr:  []byte(addr.Address),
				Amount:    adminAmount.Bytes(),
			},
		},
		TxOutputs: []*protos.TxOutput{
			{
				ToAddr: []byte(addr.Address),
				Amount: change.Bytes(),
			}, {
				ToAddr: []byte(`$`),
				Amount: amount.Bytes(),
			},
		},
		TxInputsExt:      response.Inputs,
		TxOutputsExt:     response.Outputs,
		ContractRequests: response.Requests,
	}

	// 签名
	sign, err := txhash.ProcessSignTx(chain.Context().Crypto, tx, []byte(addr.PrivateKeyStr))
	if err != nil {
		return nil, err
	}
	signs := []*protos.SignatureInfo{
		{
			PublicKey: addr.PublicKeyStr,
			Sign:      sign,
		},
	}
	tx.InitiatorSigns = signs
	tx.AuthRequireSigns = signs

	// txID
	tx.Txid, err = txhash.MakeTransactionID(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func TestChain_SubmitTx_case_contract(t *testing.T) {
	patch := setup(t)
	defer patch.Reset()

	engine, err := MockEngine("p2pv2/node1/conf/env.yaml")
	if err != nil {
		t.Fatalf("%v", err)
	}
	go engine.Run()
	defer engine.Exit()

	chain, err := engine.Get("xuper")
	if err != nil {
		t.Fatalf("get chain error: %v", err)
		return
	}

	tx, err := mockContractTx(chain)
	if err != nil {
		t.Fatalf("mock tx error: %v", err)
	}

	err = chain.SubmitTx(chain.Context(), tx)
	if err != nil {
		t.Fatalf("submit tx error: %v", err)
	}
}

func setup(t *testing.T) *gomonkey.Patches {
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
	return patch
}

func TestChain_PreExec(t *testing.T) {
	patch := setup(t)
	defer patch.Reset()

	engine, err := MockEngine("p2pv2/node1/conf/env.yaml")
	if err != nil {
		t.Fatalf("%v", err)
	}
	go engine.Run()
	defer engine.Exit()

	chain, err := engine.Get("xuper")
	if err != nil {
		t.Fatalf("get chain error: %v", err)
		return
	}

	type fields struct {
		log       logs.Logger
		miner     *miner.Miner
		relyAgent common.ChainRelyAgent
		txIdCache *cache.Cache
	}
	type args struct {
		ctx          xctx.XContext
		reqs         []*protos.InvokeRequest
		initiator    string
		authRequires []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		// don't care much about the specific content returned, so simply judge gasUsed
		want    *protos.InvokeResponse
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				ctx:  &mockXContext{logger: new(mockLogger)},
				reqs: []*protos.InvokeRequest{mockReq("kernel")},
			},
			want: &protos.InvokeResponse{GasUsed: 1000},
		},
		{
			name: "contract method with invalid args",
			args: args{
				ctx:  &mockXContext{logger: new(mockLogger)},
				reqs: []*protos.InvokeRequest{mockReq("invalid args")},
			},
			wantErr: true,
		},
		{
			name: "method not exist",
			args: args{
				ctx:  &mockXContext{logger: new(mockLogger)},
				reqs: []*protos.InvokeRequest{mockReq("method not exist")},
			},
			wantErr: true,
		},
		{
			name: "contract not exist",
			args: args{
				ctx:  &mockXContext{logger: new(mockLogger)},
				reqs: []*protos.InvokeRequest{mockReq("contract not exist")},
			},
			wantErr: true,
		},
		{
			name: "no request",
			args: args{
				ctx: &mockXContext{logger: new(mockLogger)},
			},
			want: &protos.InvokeResponse{},
		},
		{
			name: "no logger",
			args: args{
				ctx: &mockXContext{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Chain{
				ctx:       chain.Context(),
				log:       tt.fields.log,
				miner:     tt.fields.miner,
				relyAgent: tt.fields.relyAgent,
				txIdCache: tt.fields.txIdCache,
			}
			got, err := tr.PreExec(tt.args.ctx, tt.args.reqs, tt.args.initiator, tt.args.authRequires)
			if (err != nil) != tt.wantErr {
				t.Errorf("Chain.PreExec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !(got == tt.want || got.GasUsed == tt.want.GasUsed) {
				t.Errorf("Chain.PreExec() = %v\n"+
					"\twant %v",
					got, tt.want)
			}
		})
	}
}

func TestChain_preExecWithReservedReqs(t *testing.T) {
	type fields struct {
		ctx       *common.ChainCtx
		log       logs.Logger
		miner     *miner.Miner
		relyAgent common.ChainRelyAgent
		txIdCache *cache.Cache
	}
	type args struct {
		reqCtx      *reqContext
		contractCtx *contract.ContextConfig
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *protos.InvokeResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Chain{
				ctx:       tt.fields.ctx,
				log:       tt.fields.log,
				miner:     tt.fields.miner,
				relyAgent: tt.fields.relyAgent,
				txIdCache: tt.fields.txIdCache,
			}
			got, err := tr.preExecWithReservedReqs(tt.args.reqCtx, tt.args.contractCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Chain.preExecWithReservedReqs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Chain.preExecWithReservedReqs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_preExecOnce(t *testing.T) {
	type fields struct {
		ctx       *common.ChainCtx
		log       logs.Logger
		miner     *miner.Miner
		relyAgent common.ChainRelyAgent
		txIdCache *cache.Cache
	}
	type args struct {
		logger        xctx.XContext
		contractCtx   *contract.ContextConfig
		req           *protos.InvokeRequest
		isReservedReq bool
		invokeResp    *protos.InvokeResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Chain{
				ctx:       tt.fields.ctx,
				log:       tt.fields.log,
				miner:     tt.fields.miner,
				relyAgent: tt.fields.relyAgent,
				txIdCache: tt.fields.txIdCache,
			}
			if err := tr.preExecOnce(tt.args.logger, tt.args.contractCtx, tt.args.req, tt.args.isReservedReq, tt.args.invokeResp); (err != nil) != tt.wantErr {
				t.Errorf("Chain.preExecOnce() error = %v\n"+
					"\twantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

type mockXContext struct {
	logger logs.Logger
}

func (m mockXContext) Deadline() (deadline time.Time, ok bool) {
	panic("implement me")
}

func (m mockXContext) Done() <-chan struct{} {
	panic("implement me")
}

func (m mockXContext) Err() error {
	panic("implement me")
}

func (m mockXContext) Value(_ interface{}) interface{} {
	panic("implement me")
}

func (m mockXContext) GetLog() logs.Logger {
	return m.logger
}

func (m mockXContext) GetTimer() *timer.XTimer {
	panic("implement me")
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
	panic("implement me")
}

func (m mockLogger) Debug(msg string, ctx ...interface{}) {
	panic("implement me")
}

func mockReq(req string) *protos.InvokeRequest {
	switch req {
	case "contract not exist":
		return &protos.InvokeRequest{
			ContractName: "notExist",
		}
	case "method not exist":
		return &protos.InvokeRequest{
			ModuleName:   "xkernel",
			ContractName: "$acl",
			MethodName:   "notExist",
		}
	case "invalid args":
		return &protos.InvokeRequest{
			ModuleName:   "xkernel",
			ContractName: "$acl",
			MethodName:   "NewAccount",
			Args:         nil,
		}
	default:
		return &protos.InvokeRequest{
			ModuleName:   "xkernel",
			ContractName: "$acl",
			MethodName:   "NewAccount",
			Args: map[string][]byte{
				"account_name": []byte("1234567890123456"),
				"acl":          []byte(`{"pm": {"rule": 1,"acceptValue": 1.0},"aksWeight": {"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY": 1}}`),
			},
		}
	}
}

func Test_reqContext_GetTransAmount(t *testing.T) {
	type args struct {
		contractName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "target contract",
			args: args{
				contractName: "transfer",
			},
			want: "100",
		},
		{
			name: "not target contract",
			args: args{
				contractName: "reserved",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mockReqCtx()
			if got := c.GetTransAmount(tt.args.contractName); got != tt.want {
				t.Errorf("reqContext.GetTransAmount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reqContext_IsReservedReq(t *testing.T) {
	type args struct {
		index int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "reversed req",
			args: args{
				index: 0,
			},
			want: true,
		},
		{
			name: "not reversed req",
			args: args{
				index: 1,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mockReqCtx()
			if got := c.IsReservedReq(tt.args.index); got != tt.want {
				t.Errorf("reqContext.IsReservedReq() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mockReqCtx() *reqContext {
	return &reqContext{
		transContractName: "transfer",
		transAmount:       big.NewInt(100),
		requests: []*protos.InvokeRequest{
			{
				ContractName: "reversed",
			},
			{
				ContractName: "transfer",
			},
		},
		reservedReqCnt: 1,
	}
}
