package main

import (
	"fmt"
	"log"

	"github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"

	"github.com/spf13/cobra"
)

var (
	Version   = ""
	BuildTime = ""
	CommitID  = ""
)

func main() {
	rootCmd, err := NewClientCommand()
	if err != nil {
		log.Printf("new client command failed.err:%v", err)
		return
	}

	if err = rootCmd.Execute(); err != nil {
		log.Printf("command exec failed.err:%v", err)
		return
	}
}

func NewClientCommand() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:           xdef.CmdLineName + " <command> [arguments]",
		Short:         xdef.CmdLineName + " is a blockchain terminal client.",
		Long:          xdef.CmdLineName + " is a blockchain terminal client.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       xdef.CmdLineName + " tx query [txid]",
	}

	// cmd version
	rootCmd.AddCommand(GetVersionCmd().GetCmd())
	// contract client
	rootCmd.AddCommand(cmd.GetContractCmd().GetCmd())
	// tx client
	rootCmd.AddCommand(cmd.GetTxCmd().GetCmd())
	// block client
	rootCmd.AddCommand(cmd.GetBlockCmd().GetCmd())
	// blockchain client
	rootCmd.AddCommand(cmd.GetChainCmd().GetCmd())

	// 添加全局Flags
	rootFlag := rootCmd.PersistentFlags()
	rootFlag.StringVarP(&global.GFlagConf, "conf", "c", "./conf/client.yaml", "client config")
	rootFlag.StringVarP(&global.GFlagCrypto, "crypto", "", "default", "crypto type")
	rootFlag.StringVarP(&global.GFlagHost, "host", "H", "127.0.0.1:36101", "node host")
	rootFlag.StringVarP(&global.GFlagKeys, "keys", "", "./data/keys", "account address")
	rootFlag.StringVarP(&global.GFlagBCName, "name", "", xdef.DefChainName, "chain name")

	return rootCmd, nil
}

type versionCmd struct {
	global.BaseCmd
}

func GetVersionCmd() *versionCmd {
	versionCmdIns := new(versionCmd)

	subCmd := &cobra.Command{
		Use:     "version",
		Short:   "view process version information.",
		Example: xdef.CmdLineName + " version",
		Run: func(cmd *cobra.Command, args []string) {
			versionCmdIns.PrintVersion()
		},
	}
	versionCmdIns.SetCmd(subCmd)

	return versionCmdIns
}

func (t *versionCmd) PrintVersion() {
	fmt.Printf("%s-%s %s\n", Version, CommitID, BuildTime)
}
