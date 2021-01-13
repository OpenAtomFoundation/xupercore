package main

import (
	"log"

	"github.com/xuperchain/xupercore/example/xchain/cmd/client/cmd"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd, err := NewClientCommand()
	if err != nil {
		log.Fatalf("new client command failed.err:%v", err)
	}

	if err = rootCmd.Execute(); err != nil {
		log.Fatalf("command exec failed.err:%v", err)
	}
}

func NewClientCommand() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:           "xchain-cli <command> [arguments]",
		Short:         "xchain-cli is a blockchain terminal client.",
		Long:          "xchain-cli is a blockchain terminal client.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       "xchain-cli tx query [txid]",
	}

	// cmd version
	rootCmd.AddCommand(cmd.GetVersionCmd().GetCmd())
	// contract client
	rootCmd.AddCommand(cmd.GetContractCmd().GetCmd())
	// tx client
	rootCmd.AddCommand(cmd.GetTxCmd().GetCmd())
	// block client
	rootCmd.AddCommand(cmd.GetBlockCmd().GetCmd())
	// blockchain client
	rootCmd.AddCommand(cmd.GetChainCmd().GetCmd())

	return rootCmd, nil
}
