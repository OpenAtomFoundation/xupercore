package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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
		Short:   "View process version information.",
		Example: "xchain version",
		Run: func(cmd *cobra.Command, args []string) {
			Version()
		},
	}

	return versionCmdIns
}

func Version() {
	fmt.Printf("%s-%s %s\n", buildVersion, commitHash, buildDate)
}
