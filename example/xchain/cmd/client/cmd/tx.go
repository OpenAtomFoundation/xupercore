package cmd

import (
	txcmd "github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd/tx"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"

	"github.com/spf13/cobra"
)

type TxCmd struct {
	global.BaseCmd
}

func GetTxCmd() *TxCmd {
	txCmdIns := new(TxCmd)

	txCmdIns.Cmd = &cobra.Command{
		Use:           "tx",
		Short:         "Transaction query operation.",
		Example:       "xchain-cli tx query -t txid",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// query tx
	txCmdIns.Cmd.AddCommand(txcmd.GetQueryTxCmd().GetCmd())
	// submit tx
	//txCmdIns.Cmd.AddCommand(txcmd.GetSubmitTxCmd().GetCmd())

	return txCmdIns
}
