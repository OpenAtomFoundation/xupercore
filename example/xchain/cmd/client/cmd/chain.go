package cmd

import (
	chaincmd "github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd/chain"

	"github.com/spf13/cobra"
)

type ChainCmd struct {
	BaseCmd
}

func GetChainCmd() *ChainCmd {
	chainCmdIns := new(ChainCmd)

	chainCmdIns.cmd = &cobra.Command{
		Use:           "chain",
		Short:         "Chain info query operation.",
		Example:       "xchain-cli chain status",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       "xchain-cli chain status",
	}

	// query chain status
	chainCmdIns.AddCommand(chaincmd.GetStatusCmd().GetCmd())

	return chainCmdIns
}
