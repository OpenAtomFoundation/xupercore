package tx

import (
	"encoding/json"
	"fmt"

	"github.com/xuperchain/xupercore/example/xchain/cmd/client/client"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"

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
		Example:       xdef.CmdLineName + " tx query -t [txid]",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return queryTxCmdIns.printTxInfo()
		},
	}

	// 设置命令行参数并绑定变量
	queryTxCmdIns.Cmd.Flags().StringVarP(&queryTxCmdIns.TxId, "txid", "t", "", "transaction txid")

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

	type outTxStruct struct {
		Tx       *client.Transaction `json:"tx"`
		Status   int                 `json:"status"`
		Distance int64               `json:"distance"`
	}
	txInfo := &outTxStruct{
		Tx:       client.FromPBTx(resp.GetTx()),
		Status:   int(resp.GetStatus()),
		Distance: int64(resp.GetDistance()),
	}

	output, err := json.MarshalIndent(txInfo, "", "  ")
	if err != nil {
		fmt.Sprintf("json marshal tx failed.err:%v", err)
		return fmt.Errorf("json marshal tx failed")
	}

	fmt.Println(string(output))
	return nil
}
