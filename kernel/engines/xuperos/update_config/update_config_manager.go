package update_config

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
)

type Manager struct {
	Ctx *UpdateConfigCtx
}

func NewUpdateConfigManager(ctx *UpdateConfigCtx) (*Manager, error) {
	if ctx == nil || ctx.Contract == nil || ctx.BcName == "" {
		return nil, fmt.Errorf("update config ctx set error")
	}
	t := NewKernMethod(ctx)
	register := ctx.Contract.GetKernRegistry()
	register.RegisterKernMethod(utils.UpdateConfigKernelContract, updateGasPriceMethod, t.updateGasPrice)
	register.RegisterKernMethod(utils.UpdateConfigKernelContract, updateMaxBlockSize, t.updateMaxBlockSize)
	mg := &Manager{
		Ctx: ctx,
	}
	return mg, nil
}
