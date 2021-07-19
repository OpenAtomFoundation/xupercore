package kernel

import (
	"context"
	"encoding/json"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xupercore/kernel/contract/bridge/pb"
	"github.com/xuperchain/xupercore/protos"
)

type kcontextImpl struct {
	ctx     *bridge.Context
	syscall *bridge.SyscallService
	contract.StateSandbox
	contract.ChainCore
	used, limit contract.Limits
}

func newKContext(ctx *bridge.Context, syscall *bridge.SyscallService) *kcontextImpl {
	return &kcontextImpl{
		ctx:          ctx,
		syscall:      syscall,
		limit:        ctx.ResourceLimits,
		StateSandbox: ctx.State,
		ChainCore:    ctx.Core,
	}
}

// 交易相关数据
func (k *kcontextImpl) Args() map[string][]byte {
	return k.ctx.Args
}

func (k *kcontextImpl) Initiator() string {
	return k.ctx.Initiator
}

func (k *kcontextImpl) Caller() string {
	return k.ctx.Caller
}

func (k *kcontextImpl) AuthRequire() []string {
	return k.ctx.AuthRequire
}

func (k *kcontextImpl) AddResourceUsed(delta contract.Limits) {
	k.used.Add(delta)
}

func (k *kcontextImpl) ResourceLimit() contract.Limits {
	return k.limit
}

func (k *kcontextImpl) Call(module, contractName, method string, args map[string][]byte) (*contract.Response, error) {
	var argPairs []*pb.ArgPair
	for k, v := range args {
		argPairs = append(argPairs, &pb.ArgPair{
			Key:   k,
			Value: v,
		})
	}
	request := &pb.ContractCallRequest{
		Header: &pb.SyscallHeader{
			Ctxid: k.ctx.ID,
		},
		Module:   module,
		Contract: contractName,
		Method:   method,
		Args:     argPairs,
	}
	resp, err := k.syscall.ContractCall(context.TODO(), request)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status:  int(resp.Response.GetStatus()),
		Message: resp.Response.GetMessage(),
		Body:    resp.Response.GetBody(),
	}, nil
}

// EmitAsyncTask 异步发送订阅事件
func (k *kcontextImpl) EmitAsyncTask(event string, args interface{}) (err error) {
	var rawBytes []byte
	// 见asyncworker.TaskContextImpl, Unmarshal函数对应为json.Unmarshal
	rawBytes, err = json.Marshal(args)
	if err != nil {
		return
	}
	e := protos.ContractEvent{
		Contract: k.ctx.ContractName,
		Name:     event,
		Body:     rawBytes,
	}
	k.AddEvent(&e)
	return
}
