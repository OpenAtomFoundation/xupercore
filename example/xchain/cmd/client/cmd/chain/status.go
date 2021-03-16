package chain

import (
	"encoding/json"
	"fmt"

	"github.com/xuperchain/xupercore/example/xchain/cmd/client/client"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"

	"github.com/spf13/cobra"
)

type ChainStatusCmd struct {
	global.BaseCmd
}

func GetChainStatusCmd() *ChainStatusCmd {
	chainStatusCmdIns := new(ChainStatusCmd)

	chainStatusCmdIns.Cmd = &cobra.Command{
		Use:           "status",
		Short:         "print chain status.",
		Example:       xdef.CmdLineName + " chain status",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return chainStatusCmdIns.printChainStatus()
		},
	}

	return chainStatusCmdIns
}

func (t *ChainStatusCmd) printChainStatus() error {
	xcli, err := client.NewXchainClient()
	if err != nil {
		fmt.Sprintf("grpc dial failed.err:%v\n", err)
		return fmt.Errorf("grpc dial failed")
	}

	resp, err := xcli.QueryChainStatus()
	if err != nil {
		fmt.Sprintf("query chain status failed.err:%v", err)
		return fmt.Errorf("query chain status failed")
	}

	outInfo := client.FromChainStatusPB(resp)
	output, err := json.MarshalIndent(outInfo, "", "  ")
	if err != nil {
		fmt.Sprintf("json marshal chain status failed.err:%v", err)
		return fmt.Errorf("json marshal chain status failed")
	}

	fmt.Println(string(output))
	return nil
}
