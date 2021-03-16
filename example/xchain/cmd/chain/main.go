package main

import (
	"fmt"
	"log"

	"github.com/xuperchain/xupercore/example/xchain/cmd/chain/cmd"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"

	"github.com/spf13/cobra"
)

var (
	Version   = ""
	BuildTime = ""
	CommitID  = ""
)

func main() {
	rootCmd, err := NewServiceCommand()
	if err != nil {
		log.Fatalf("start service failed.err:%v", err)
	}

	if err = rootCmd.Execute(); err != nil {
		log.Fatalf("start service failed.err:%v", err)
	}
}

func NewServiceCommand() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:           xdef.ServerName + " <command> [arguments]",
		Short:         xdef.ServerName + " is a blockchain network building service.",
		Long:          xdef.ServerName + " is a blockchain network building service.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       xdef.ServerName + " startup --conf /home/rd/xuperos/conf/env.yaml",
	}

	// cmd service
	rootCmd.AddCommand(cmd.GetStartupCmd().GetCmd())
	// cmd version
	rootCmd.AddCommand(GetVersionCmd().GetCmd())

	return rootCmd, nil
}

type versionCmd struct {
	cmd.BaseCmd
}

func GetVersionCmd() *versionCmd {
	versionCmdIns := new(versionCmd)

	subCmd := &cobra.Command{
		Use:     "version",
		Short:   "view process version information.",
		Example: xdef.ServerName + " version",
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
