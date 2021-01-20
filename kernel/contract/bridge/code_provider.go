package bridge

import (
	"errors"
	"fmt"

	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/protos"

	"github.com/golang/protobuf/proto"
)

type stateReader interface {
	Get(bucket string, key []byte) ([]byte, error)
}

type xmStateReader struct {
	r ledger.XMReader
}

func fromXMReader(r ledger.XMReader) stateReader {
	return &xmStateReader{
		r: r,
	}
}

func (x *xmStateReader) Get(bucket string, key []byte) ([]byte, error) {
	value, err := x.r.Get(bucket, key)
	if err != nil {
		return nil, err
	}
	if sandbox.IsEmptyVersionedData(value) ||
		sandbox.IsDelFlag(value.PureData.Value) {
		return nil, errors.New("not found")
	}

	return value.PureData.Value, nil
}

type codeProvider struct {
	xstore stateReader
}

func newCodeProviderFromXMReader(r ledger.XMReader) ContractCodeProvider {
	return newCodeProvider(fromXMReader(r))
}

func newCodeProvider(xstore stateReader) ContractCodeProvider {
	return &codeProvider{
		xstore: xstore,
	}
}

func (c *codeProvider) GetContractCode(name string) ([]byte, error) {
	value, err := c.xstore.Get("contract", contractCodeKey(name))
	if err != nil {
		return nil, fmt.Errorf("get contract code for '%s' error:%s", name, err)
	}
	codebuf := value
	if len(codebuf) == 0 {
		return nil, errors.New("empty wasm code")
	}
	return codebuf, nil
}

func (c *codeProvider) GetContractAbi(name string) ([]byte, error) {
	value, err := c.xstore.Get("contract", contractAbiKey(name))
	if err != nil {
		return nil, fmt.Errorf("get contract abi for '%s' error:%s", name, err)
	}
	abiBuf := value
	if len(abiBuf) == 0 {
		return nil, errors.New("empty abi")
	}
	return abiBuf, nil
}

func (c *codeProvider) GetContractCodeDesc(name string) (*protos.WasmCodeDesc, error) {
	value, err := c.xstore.Get("contract", ContractCodeDescKey(name))
	if err != nil {
		return nil, fmt.Errorf("get contract desc for '%s' error:%s", name, err)
	}
	descbuf := value
	// FIXME: 如果key不存在ModuleCache不应该返回零长度的value
	if len(descbuf) == 0 {
		return nil, errors.New("empty wasm code desc")
	}
	var desc protos.WasmCodeDesc
	err = proto.Unmarshal(descbuf, &desc)
	if err != nil {
		return nil, err
	}
	return &desc, nil
}

type descProvider struct {
	ContractCodeProvider
	desc *protos.WasmCodeDesc
}

func newDescProvider(cp ContractCodeProvider, desc *protos.WasmCodeDesc) ContractCodeProvider {
	return &descProvider{
		ContractCodeProvider: cp,
		desc:                 desc,
	}
}

func (d *descProvider) GetContractCodeDesc(name string) (*protos.WasmCodeDesc, error) {
	return d.desc, nil
}
