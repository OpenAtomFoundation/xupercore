package xrandom

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/xrandom/bls"
	eth "github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/xrandom/ecdsa"
	"github.com/OpenAtomFoundation/xupercore/global/lib/logs"
)

type Manager struct {
	Ctx *Context
}

var manager *Manager
var log *logs.LogFitter

func NewManager(ctx *Context) error {
	ctx.GetLog().Info("xrandom.NewManager()", "ctx", ctx)
	if ctx == nil || ctx.Contract == nil {
		return fmt.Errorf("%s contract ctx set error", ContractName)
	}

	// init account
	blsAccount, err := loadBlsAccount(ctx)
	if err != nil {
		ctx.GetLog().Warn("load BLS Account fail", "error", err)
		blsAccount = bls.SetAccount(nil)

		if err = saveBlsAccount(ctx, blsAccount); err != nil {
			ctx.GetLog().Error("save BLS Account fail", "error", err)
		}
	}
	_ = bls.SetAccount(blsAccount)
	ethAccount, err := loadEthAccount(ctx)
	if err != nil {
		ctx.GetLog().Error("load ETH Account fail", "error", err)
		return errors.Wrap(err, "load ETH Account fail")
	}
	eth.SetAccount(ethAccount)

	// init engine
	err = bls.SetEngine(ctx.ChainCtx.EngCtx)
	if err != nil {
		return err
	}

	// init methods
	admins := ctx.ChainCtx.Ledger.GenesisBlock.GetConfig().XRandom.Admins
	x := NewContract(admins, ctx)

	register := ctx.Contract.GetKernRegistry()
	kMethods := map[string]contract.KernMethod{
		"AddNode":            x.AddNode,
		"DeleteNode":         x.DeleteNode,
		"QueryAccessList":    x.QueryAccessList,
		"SubmitRandomNumber": x.SubmitRandomNumber,
		"QueryRandomNumber":  x.QueryRandomNumber,
	}

	for method, f := range kMethods {
		if _, err := register.GetKernMethod(ContractName, method); err != nil {
			register.RegisterKernMethod(ContractName, method, f)
		}
	}
	manager = &Manager{Ctx: ctx}
	log, _ = logs.NewLogger("", "xrandom")
	ctx.GetLog().Debug("XRandom manager init done")
	return nil
}
