package evm

import (
	"encoding/hex"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/txs"
	"github.com/xuperchain/xupercore/kernel/contract"
)

type EVMProxy interface {
}

func NewEVMProxy(manager contract.Manager) (EVMProxy, error) {
	registry := manager.GetKernRegistry()
	p := proxy{}
	p.verifySignatureFunc = p.verifySignature
	registry.RegisterKernMethod("$evm", "SendTransaction", p.sendTransaction)
	return &p, nil
}

type proxy struct {
	//  func field for unit test convenience
	verifySignatureFunc func(nonce, gasPrice, gasLimit uint64, to, value, data []byte, net, V uint64, S, R []byte,
	) error
}

func (p *proxy) sendTransaction(ctx contract.KContext) (*contract.Response, error) {
	//if err:=p.verifySignature();err!=nil{
	//	return nil,err
	//}
	return p.contractCall(ctx)

}
func (p *proxy) contractCall(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()

	// TODO length check
	// TODO existence check
	input, err := hex.DecodeString(string(args["input"]))
	if err != nil {
		return nil, err
	}

	args1 := map[string][]byte{
		"input":       input,
		"jsonEncoded": []byte("false"),
	}
	address, err := crypto.AddressFromHexString(string(args["to"]))
	if err != nil {
		return nil, err
	}
	contractName, err := DetermineContractNameFromEVM(address)
	if err != nil {
		return nil, err
	}

	//Initiator, err := evm.EVMAddressToXchain(fromAddress)
	// TODO
	// 1.地址转换相关问题
	// 2. 跨合约调用
	// 3.合约部署与合约升级
	resp, err := ctx.Call("evm", contractName, "", args1)
	return resp, err

}

// Too many parameters
func (p *proxy) verifySignature(
	nonce, gasPrice, gasLimit uint64,
	to, value, data []byte,
	net, V uint64,
	S, R []byte,
) error {

	enc, err := txs.RLPEncode(nonce, gasPrice, gasLimit, to, value, data)
	if err != nil {
		return err
	}

	sig := crypto.CompressedSignatureFromParams(V-net-8-1, R, S)
	pub, err := crypto.PublicKeyFromSignature(sig, crypto.Keccak256(enc))
	if err != nil {
		return err
	}
	//from := pub.GetAddress()
	unc := crypto.UncompressedSignatureFromParams(R, S)
	signature, err := crypto.SignatureFromBytes(unc, crypto.CurveTypeSecp256k1)
	if err != nil {
		return err
	}
	if err := pub.Verify(enc, signature); err != nil {
		return err
	}
	return nil
}
