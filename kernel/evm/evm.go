package evm

import (
	"encoding/hex"
	//"github.com/hyperledger/burrow/acm/balance"
	"github.com/hyperledger/burrow/crypto"
	x "github.com/hyperledger/burrow/encoding/hex"
	"github.com/hyperledger/burrow/encoding/rlp"
	"github.com/hyperledger/burrow/rpc"
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
	registry.RegisterKernMethod("$evm", "SendRawTransaction", p.sendRawTransaction)
	registry.RegisterKernMethod("$evm", "ContractCall", p.contractCall)
	return &p, nil
}

type proxy struct {
	//  func field for unit test convenience
	verifySignatureFunc func(nonce, gasPrice, gasLimit uint64, to, value, data []byte, net, V uint64, S, R []byte,
	) error
}

func (p *proxy) sendTransaction(ctx contract.KContext) (*contract.Response, error) {
	//args := ctx.Args()
	//
	//var nonce, gasPrice, gasLimit uint64
	//var to, value, data []byte
	//var net, V uint64
	//var S, R []byte

	//if err := p.verifySignature(); err != nil {
	//	return nil, err
	//}
	return p.contractCall(ctx)

}

func (p *proxy) sendRawTransaction(ctx contract.KContext) (*contract.Response, error) {
	args := ctx.Args()
	signed_tx := args["signed_tx"]
	_ = signed_tx
	data, err := x.DecodeToBytes(string(signed_tx))
	if err != nil {
		return nil, err
	}

	rawTx := new(rpc.RawTx)
	err = rlp.Decode(data, rawTx)
	if err != nil {
		return nil, err
	}
	//chainID := 1

	//net := uint64(chainID)
	//enc, err := txs.RLPEncode(rawTx.Nonce, rawTx.GasPrice, rawTx.GasLimit, rawTx.To, rawTx.Value, rawTx.Data)
	//if err != nil {
	//	return nil, err
	//}

	//sig := crypto.CompressedSignatureFromParams(rawTx.V-net-8-1, rawTx.R, rawTx.S)
	//pub, err := crypto.PublicKeyFromSignature(sig, crypto.Keccak256(enc))
	//if err != nil {
	//	return nil, err
	//}
	//
	//	func (p *proxy) verifySignature(
	//		nonce, gasPrice, gasLimit uint64,
	//		to, value, data []byte,
	//		net, V uint64,
	//		S, R []byte,
	//)
	//if err := p.verifySignature(
	//	rawTx.Nonce, rawTx.GasPrice, rawTx.GasLimit,
	//	rawTx.To, rawTx.Value, rawTx.Data,
	//	net, rawTx.V, rawTx.S, rawTx.R,
	//); err != nil {
	//	return nil, err
	//}
	//from := pub.GetAddress()
	//unc := crypto.UncompressedSignatureFromParams(rawTx.R, rawTx.S)
	//signature, err := crypto.SignatureFromBytes(unc, crypto.CurveTypeSecp256k1)
	//if err != nil {
	//	return nil, err
	//}

	to, err := crypto.AddressFromBytes(rawTx.To)
	if err != nil {
		return nil, err
	}
	//amount := balance.WeiToNative(rawTx.Value).Uint64()

	//to, err := crypto.AddressFromHexString(string(rawTx.To))
	//if err != nil {
	//	return nil, err
	//}
	contractName, err := DetermineContractNameFromEVM(to)
	if err != nil {
		return nil, err
	}

	args1 := map[string][]byte{
		"input":       rawTx.Data,
		"jsonEncoded": []byte("false"),
	}
	//TODO 如果value 非空，那么需要有 transfer

	//Initiator, err := evm.EVMAddressToXchain(fromAddress)
	// TODO
	// 1.地址转换相关问题
	// 2. 跨合约调用
	// 3.合约部署与合约升级
	resp, err := ctx.Call("evm", contractName, "", args1)
	return resp, err

	//txEnv := &txs.Envelope{
	//	Signatories: []txs.Signatory{
	//		{
	//			Address:   &from,
	//			PublicKey: pub,
	//			Signature: signature,
	//		},
	//	},
	//	Encoding: txs.Envelope_RLP,
	//	Tx: &txs.Tx{
	//		ChainID: srv.blockchain.ChainID(),
	//		Payload: &payload.CallTx{
	//			Input: &payload.TxInput{
	//				Address: from,
	//				Amount:  amount,
	//				// first tx sequence should be 1,
	//				// but metamask starts at 0
	//				Sequence: rawTx.Nonce + 1,
	//			},
	//			Address:  &to,
	//			GasLimit: rawTx.GasLimit,
	//			GasPrice: rawTx.GasPrice,
	//			Data:     rawTx.Data,
	//		},
	//	},
	//}
	//
	//ctx := context.Background()
	//txe, err := srv.trans.BroadcastTxSync(ctx, txEnv)
	//if err != nil {
	//	return nil, err
	//} else if txe.Exception != nil {
	//	return nil, txe.Exception.AsError()
	//}
	//
	//return &web3.EthSendRawTransactionResult{
	//	TransactionHash: x.EncodeBytes(txe.GetTxHash().Bytes()),
	//}, nil
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
	// TODO more robust
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
	// 2.跨合约调用
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
	//
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
