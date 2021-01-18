package tx

import (
	"encoding/json"
	"fmt"

	"github.com/xuperchain/xupercore/example/xchain/cmd/client/client"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"

	"github.com/spf13/cobra"
)

type QueryTxCmd struct {
	global.BaseCmd
	TxId string
}

func GetQueryTxCmd() *QueryTxCmd {
	queryTxCmdIns := new(QueryTxCmd)

	queryTxCmdIns.Cmd = &cobra.Command{
		Use:           "query",
		Short:         "print transaction details.",
		Example:       "xchain-cli tx query -t [txid]",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return queryTxCmdIns.printTxInfo()
		},
	}

	// 设置命令行参数并绑定变量
	queryTxCmdIns.Cmd.Flags().StringVarP(&queryTxCmdIns.TxId, "txid", "", "", "transaction txid")

	return queryTxCmdIns
}

func (t *QueryTxCmd) printTxInfo() error {
	xcli, err := client.NewXchainClient()
	if err != nil {
		fmt.Sprintf("grpc dial failed.err:%v\n", err)
		return fmt.Errorf("grpc dial failed")
	}

	resp, err := xcli.QueryTx(t.TxId)
	if err != nil {
		fmt.Sprintf("query tx failed.err:%v", err)
		return fmt.Errorf("query tx failed")
	}

	output, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		fmt.Sprintf("json marshal tx failed.err:%v", err)
		return fmt.Errorf("json marshal tx failed")
	}

	fmt.Println(string(output))
	return nil
}
