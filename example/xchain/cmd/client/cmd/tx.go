package cmd

import (
	txcmd "github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd/tx"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"

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
		Example:       xdef.CmdLineName + " tx query -t txid",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// query tx
	txCmdIns.Cmd.AddCommand(txcmd.GetQueryTxCmd().GetCmd())
	// submit tx
	//txCmdIns.Cmd.AddCommand(txcmd.GetSubmitTxCmd().GetCmd())
	// transfer
	txCmdIns.Cmd.AddCommand(txcmd.GetTransferTxCmd().GetCmd())

	return txCmdIns
}
