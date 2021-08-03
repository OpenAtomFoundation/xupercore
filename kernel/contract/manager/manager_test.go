package manager

import (
	"encoding/json"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/rpc/web3"
	"github.com/hyperledger/burrow/txs"
	"testing"

	"encoding/hex"
	log15 "github.com/xuperchain/log15"
	_ "github.com/xuperchain/xupercore/bcs/contract/evm"
	"github.com/xuperchain/xupercore/kernel/contract"
	_ "github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/contract/mock"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
)

var contractConfig = &contract.ContractConfig{
	Xkernel: contract.XkernelConfig{
		Enable: true,
		Driver: "default",
	},
	LogDriver: mock.NewMockLogger(),
}

func TestCreate(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
}

func TestCreateSandbox(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()

	r := sandbox.NewMemXModel()
	state, err := m.NewStateSandbox(&contract.SandboxConfig{
		XMReader: r,
	})
	if err != nil {
		t.Fatal(err)
	}
	state.Put("test", []byte("key"), []byte("value"))
	if string(state.RWSet().WSet[0].Value) != "value" {
		t.Error("unexpected value")
	}
}

func TestInvoke(t *testing.T) {
	th := mock.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()

	m.GetKernRegistry().RegisterKernMethod("$hello", "Hi", new(helloContract).Hi)

	resp, err := th.Invoke("xkernel", "$hello", "Hi", map[string][]byte{
		"name": []byte("xuper"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", resp.Body)
}

type evmTransaction struct {
}

type helloContract struct {
}

func (h *helloContract) Hi(ctx contract.KContext) (*contract.Response, error) {
	name := ctx.Args()["name"]
	ctx.Put("test", []byte("k1"), []byte("v1"))
	return &contract.Response{
		Body: []byte("hello " + string(name)),
	}, nil
}

func TestUnmarshal(t *testing.T) {
	params := `{"from": "b60e8dd61c5d32be8058bb8eb970870f07233155", "to": "313131312D2D2D2D2D2D2D2D2D636F756E746572", "gas": "76c0","gasPrice": "9184e72a000", "value": "9184e72a", "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"}`
	//data, err := hex.DecodeString("d46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675")
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	//t.Log(string(data))
	tx := &web3.EthSendTransactionParams{}
	//tx := &web3.Transaction{}
	if err := json.Unmarshal([]byte(params), tx); err != nil {
		t.Error(err)
		return
	}
	t.Logf("%v", tx)
}

func TestRLP(t *testing.T) {
	r, err := hex.DecodeString("3c46a1ff9d0dd2129a7f8fbc3e45256d85890d9d63919b42dac1eb8dfa443a32")
	if err != nil {
		t.Error(err)
		return
	}
	s, err := hex.DecodeString("6b2be3f225ae31f7ca18efc08fa403eb73b848359a63cd9fdeb61e1b83407690")
	if err != nil {
		t.Error(err)
		return
	}
	value, err := hex.DecodeString("616263646566")
	if err != nil {
		t.Error(err)
		return
	}
	address, err := hex.DecodeString("0100000000000000000000000000000000000000")
	if err != nil {
		t.Error(err)
		return
	}
	data, err := hex.DecodeString("616263646566")
	if err != nil {
		t.Error(err)
		return
	}
	enc, err := txs.RLPEncode(0, 0, 0, address, value, data)

	sig := crypto.CompressedSignatureFromParams(0, r, s)
	pub, err := crypto.PublicKeyFromSignature(sig, crypto.Keccak256(enc))
	if err != nil {
		t.Error(err)
		return
	}
	unc := crypto.UncompressedSignatureFromParams(r, s)
	signature, err := crypto.SignatureFromBytes(unc, crypto.CurveTypeSecp256k1)
	if err != nil {
		t.Error(err)
		return
	}
	if err := pub.Verify(unc, signature); err != nil {
		t.Error(err)
		return
	}
	//	ref rpc/eth.go EthSendRawTransaction
}
