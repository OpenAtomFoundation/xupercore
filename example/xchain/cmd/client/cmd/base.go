package cmd

import (
	"github.com/spf13/cobra"
)

type BaseCmd struct {
	// cobra command
	cmd *cobra.Command
}

func (t *BaseCmd) SetCmd(cmd *cobra.Command) {
	t.cmd = cmd
}

func (t *BaseCmd) GetCmd() *cobra.Command {
	return t.cmd
}
