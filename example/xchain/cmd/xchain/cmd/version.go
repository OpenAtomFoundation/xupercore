package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xuperchain/xupercore/example/xchain/common/version"
)

// 通过编译参数设置
var (
	buildVersion = "0.0.0"
	commitHash   = "default"
	buildDate    = "default"
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
			version.Version()
		},
	}

	return versionCmdIns
}
