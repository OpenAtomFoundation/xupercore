package main

import (
	"log"

	"github.com/xuperchain/xupercore/example/xchain/cmd/xchain/cmd"

	"github.com/spf13/cobra"
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
		Use:           "xchain <command> [arguments]",
		Short:         "xchain is a blockchain network building service.",
		Long:          "xchain is a blockchain network building service.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       "xchain startup --conf /home/rd/xchain/conf/env.yaml",
	}

	// cmd service
	rootCmd.AddCommand(cmd.GetStartupCmd().GetCmd())
	// cmd version
	rootCmd.AddCommand(cmd.GetVersionCmd().GetCmd())

	return rootCmd, nil
}
