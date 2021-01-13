package cmd

import (
	txcmd "github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd/tx"

	"github.com/spf13/cobra"
)

type TxCmd struct {
	BaseCmd
}

func GetTxCmd() *TxCmd {
	txCmdIns := new(TxCmd)

	txCmdIns.cmd = &cobra.Command{
		Use:           "tx",
		Short:         "Transaction query operation.",
		Example:       "xchain-cli tx query [txid]",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       "xchain-cli tx query -t txid",
	}

	// query tx
	txCmdIns.AddCommand(txcmd.GetQueryTxCmd().GetCmd())
	return txCmdIns
}
