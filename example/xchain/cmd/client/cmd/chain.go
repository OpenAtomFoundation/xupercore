package cmd

import (
	chaincmd "github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd/chain"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"

	"github.com/spf13/cobra"
)

type ChainCmd struct {
	global.BaseCmd
}

func GetChainCmd() *ChainCmd {
	chainCmdIns := new(ChainCmd)

	chainCmdIns.Cmd = &cobra.Command{
		Use:           "chain",
		Short:         "query chain info.",
		Example:       "xchain-cli chain status",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// query chain status
	chainCmdIns.Cmd.AddCommand(chaincmd.GetChainStatusCmd().GetCmd())

	return chainCmdIns
}
