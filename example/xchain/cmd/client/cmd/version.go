package cmd

import (
	"fmt"

	"github.com/xuperchain/xupercore/example/xchain/common/version"

	"github.com/spf13/cobra"
)

type versionCmd struct {
	BaseCmd
}

func GetVersionCmd() *versionCmd {
	versionCmdIns := new(versionCmd)

	versionCmdIns.cmd = &cobra.Command{
		Use:     "version",
		Short:   "view process version information.",
		Example: "xchain version",
		Run: func(cmd *cobra.Command, args []string) {
			version.PrintVersion()
		},
	}

	return versionCmdIns
}
