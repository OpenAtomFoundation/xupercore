package cmd

import (
	blkcmd "github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd/block"

	"github.com/spf13/cobra"
)

type BlockCmd struct {
	BaseCmd
}

func GetBlockCmd() *BlockCmd {
	blockCmdIns := new(BlockCmd)

	blockCmdIns.cmd = &cobra.Command{
		Use:           "block",
		Short:         "Block info query operation.",
		Example:       "xchain-cli block query [block_id]",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       "xchain-cli block query [block_id]",
	}

	// query block info
	blockCmdIns.AddCommand(blkcmd.GetQueryCmd().GetCmd())
	// query block info
	blockCmdIns.AddCommand(blkcmd.GetQueryByHeight().GetCmd())

	return blockCmdIns
}
