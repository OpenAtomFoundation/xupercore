package tx

import (
	"encoding/json"
	"fmt"

	"github.com/xuperchain/xupercore/example/xchain/cmd/client/client"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"

	"github.com/spf13/cobra"
)

type QueryBlockCmd struct {
	global.BaseCmd
	BlockId string
}

func GetQueryBlockCmd() *QueryBlockCmd {
	queryBlockCmdIns := new(QueryBlockCmd)

	queryBlockCmdIns.Cmd = &cobra.Command{
		Use:           "query",
		Short:         "print block details.",
		Example:       xdef.CmdLineName + " block query -b [blockId]",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return queryBlockCmdIns.printBlockInfo()
		},
	}

	// 设置命令行参数并绑定变量
	queryBlockCmdIns.Cmd.Flags().StringVarP(&queryBlockCmdIns.BlockId, "block_id", "b", "", "block id")

	return queryBlockCmdIns
}

func (t *QueryBlockCmd) printBlockInfo() error {
	xcli, err := client.NewXchainClient()
	if err != nil {
		fmt.Sprintf("grpc dial failed.err:%v\n", err)
		return fmt.Errorf("grpc dial failed")
	}

	resp, err := xcli.QueryBlock(t.BlockId)
	if err != nil {
		fmt.Sprintf("query block info failed.err:%v", err)
		return fmt.Errorf("query block info failed")
	}

	type outBlockInfo struct {
		Block  *client.InternalBlock `json:"block"`
		Status int                   `json:"status"`
	}
	blkInfo := &outBlockInfo{
		Block:  client.FromInternalBlockPB(resp.GetBlock()),
		Status: int(resp.GetStatus()),
	}

	output, err := json.MarshalIndent(blkInfo, "", "  ")
	if err != nil {
		fmt.Sprintf("json marshal block info failed.err:%v", err)
		return fmt.Errorf("json marshal block info failed")
	}

	fmt.Println(string(output))
	return nil
}
