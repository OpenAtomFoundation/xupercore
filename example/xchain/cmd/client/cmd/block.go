package cmd

import (
	//blkcmd "github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd/block"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"

	"github.com/spf13/cobra"
)

type BlockCmd struct {
	global.BaseCmd
}

func GetBlockCmd() *BlockCmd {
	blockCmdIns := new(BlockCmd)

	blockCmdIns.Cmd = &cobra.Command{
		Use:           "block",
		Short:         "Block info query operation.",
		Example:       "xchain-cli block query [block_id]",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// query block info
	//blockCmdIns.AddCommand(blkcmd.GetQueryCmd().GetCmd())
	// query block info
	//blockCmdIns.AddCommand(blkcmd.GetQueryByHeight().GetCmd())

	return blockCmdIns
}
