package bridge

import (
	"fmt"
	"path/filepath"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/logs"

	"github.com/xuperchain/xupercore/protos"
)

// XBridge 用于注册用户虚拟机以及向Xchain Core注册可被识别的vm.VirtualMachine
type XBridge struct {
	ctxmgr         *ContextManager
	syscallService *SyscallService
	basedir        string
	vmconfigs      map[ContractType]VMConfig
	creators       map[ContractType]InstanceCreator
	xmodel         ledger.XMReader
	config         contract.ContractConfig
	core           contract.ChainCore

	debugLogger logs.LogDriver

	*contractManager
}

type XBridgeConfig struct {
	Basedir   string
	VMConfigs map[ContractType]VMConfig
	XModel    ledger.XMReader
	Config    contract.ContractConfig
	LogDriver logs.LogDriver
	Core      contract.ChainCore
}

// New instances a new XBridge
func New(cfg *XBridgeConfig) (*XBridge, error) {
	ctxmgr := NewContextManager()
	xbridge := &XBridge{
		ctxmgr:      ctxmgr,
		basedir:     cfg.Basedir,
		vmconfigs:   cfg.VMConfigs,
		creators:    make(map[ContractType]InstanceCreator),
		xmodel:      cfg.XModel,
		core:        cfg.Core,
		config:      cfg.Config,
		debugLogger: cfg.LogDriver,
	}
	xbridge.contractManager = &contractManager{
		xbridge:      xbridge,
		codeProvider: newCodeProviderFromXMReader(cfg.XModel),
	}

	syscallService := NewSyscallService(ctxmgr, xbridge)
	xbridge.syscallService = syscallService
	err := xbridge.initVM()
	if err != nil {
		return nil, err
	}
	return xbridge, nil
}

func (v *XBridge) initVM() error {
	types := []ContractType{TypeWasm, TypeNative, TypeEvm, TypeKernel}
	for _, tp := range types {
		vmconfig, ok := v.vmconfigs[tp]
		if !ok {
			// log.Error("config for contract type not found", "type", tp)
			continue
		}
		if !vmconfig.IsEnable() {
			// log.Info("contract type disabled", "type", tp)
			continue
		}
		creatorConfig := &InstanceCreatorConfig{
			Basedir:        filepath.Join(v.basedir, vmconfig.DriverName()),
			SyscallService: v.syscallService,
			VMConfig:       vmconfig,
		}
		creator, err := Open(tp, vmconfig.DriverName(), creatorConfig)
		if err != nil {
			return err
		}
		v.creators[tp] = creator
	}
	return nil
}

func (v *XBridge) getCreator(tp ContractType) InstanceCreator {
	return v.creators[tp]
}

func (v *XBridge) NewContext(ctxCfg *contract.ContextConfig) (contract.Context, error) {
	var desc *protos.WasmCodeDesc
	var err error

	if ctxCfg.Module == string(TypeKernel) {
		desc = &protos.WasmCodeDesc{
			ContractType: ctxCfg.Module,
		}
	} else {
		// test if contract exists
		desc, err = newCodeProvider(ctxCfg.State).GetContractCodeDesc(ctxCfg.ContractName)
		if err != nil {
			return nil, err
		}
	}
	tp, err := getContractType(desc)
	if err != nil {
		return nil, err
	}
	vm := v.getCreator(tp)
	if vm == nil {
		return nil, fmt.Errorf("vm for contract type %s not supported", tp)
	}
	var cp ContractCodeProvider
	// 如果当前在部署合约，合约代码从cache获取
	// 合约调用的情况则从model中拿取合约代码，避免交易中包含合约代码的引用。
	if ctxCfg.ContractCodeFromCache {
		cp = newCodeProvider(ctxCfg.State)
	} else {
		cp = newDescProvider(v.codeProvider, desc)
	}

	ctx := v.ctxmgr.MakeContext()
	ctx.State = ctxCfg.State
	ctx.Core = v.core
	ctx.Module = ctxCfg.Module
	ctx.ContractName = ctxCfg.ContractName
	ctx.Initiator = ctxCfg.Initiator
	ctx.Caller = ctxCfg.Caller
	ctx.AuthRequire = ctxCfg.AuthRequire
	ctx.ResourceLimits = ctxCfg.ResourceLimits
	ctx.CanInitialize = ctxCfg.CanInitialize
	ctx.TransferAmount = ctxCfg.TransferAmount
	ctx.ContractSet = ctxCfg.ContractSet
	if ctx.ContractSet == nil {
		ctx.ContractSet = make(map[string]bool)
		ctx.ContractSet[ctx.ContractName] = true
	}
	// lifecycle of debug logger driver is coincident with bridge
	// while ctx.Logger's coincident with context
	if v.debugLogger != nil {
		ctx.Logger = v.debugLogger
	} else {
		// use contract Name for convience of filter log from specific contract using grep or other logging processing stack
		ctx.Logger, err = logs.NewLogger(fmt.Sprintf("%016d", ctx.ID), "contract_"+ctx.ContractName)
	}

	if err != nil {
		return nil, err
	}
	release := func() {
		v.ctxmgr.DestroyContext(ctx)
	}

	instance, err := vm.CreateInstance(ctx, cp)
	if err != nil {
		v.ctxmgr.DestroyContext(ctx)
		return nil, err
	}
	ctx.Instance = instance
	return &vmContextImpl{
		ctx:      ctx,
		instance: instance,
		release:  release,
	}, nil
}
