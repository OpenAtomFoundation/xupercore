package cmd

import (
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	"github.com/xuperchain/xupercore/example/xchain/common/version"

	"github.com/spf13/cobra"
)

type versionCmd struct {
	global.BaseCmd
}

func GetVersionCmd() *versionCmd {
	versionCmdIns := new(versionCmd)

	versionCmdIns.Cmd = &cobra.Command{
		Use:     "version",
		Short:   "view process version information.",
		Example: "xchain version",
		Run: func(cmd *cobra.Command, args []string) {
			version.PrintVersion()
		},
	}

	return versionCmdIns
}
