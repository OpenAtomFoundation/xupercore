package chain_config

import (
	"fmt"

	"github.com/xuperchain/xupercore/kernel/contract/proposal/utils"
)

type Manager struct {
	Ctx *ChainConfigCtx
}

func NewChainConfigManager(ctx *ChainConfigCtx) (*Manager, error) {
	if ctx == nil || ctx.Contract == nil || ctx.BcName == "" {
		return nil, fmt.Errorf("update config ctx set error")
	}
	t := NewKernMethod(ctx)
	register := ctx.Contract.GetKernRegistry()
	register.RegisterKernMethod(utils.ChainConfigKernelContract, utils.GetUpdateGasPriceMethod(), t.updateGasPrice)
	register.RegisterKernMethod(utils.ChainConfigKernelContract, utils.GetUpdateMaxBlockSizeMethod(), t.updateMaxBlockSize)
	mg := &Manager{
		Ctx: ctx,
	}
	return mg, nil
}
