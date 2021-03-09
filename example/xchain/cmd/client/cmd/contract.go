package cmd

import (
	//contractcmd "github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd/contract"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"

	"github.com/spf13/cobra"
)

type ContractCmd struct {
	global.BaseCmd
}

func GetContractCmd() *ContractCmd {
	contractCmdIns := new(ContractCmd)

	contractCmdIns.Cmd = &cobra.Command{
		Use:           "contract",
		Short:         "Contract operation.",
		Example:       xdef.CmdLineName + " contract invoke [options]",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// native、wasm合约通过参数区分

	// deploy contract
	//contractCmdIns.AddCommand(contractcmd.GetDeployCmd().GetCmd())
	// invoke contract
	//contractCmdIns.AddCommand(contractcmd.GetInvokeCmd().GetCmd())
	// query contract
	//contractCmdIns.AddCommand(contractcmd.GetQueryCmd().GetCmd())
	// upgrade contract
	//contractCmdIns.AddCommand(contractcmd.GetUpgradeCmd().GetCmd())

	return contractCmdIns
}
