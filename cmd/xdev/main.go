package main

import "github.com/xuperchain/xupercore/cmd/xdev/internal/cmd"

var (
	buildVersion = ""
	buildDate    = ""
	commitHash   = ""
)

func main() {
	cmd.SetVersion(buildVersion, buildDate, commitHash)
	cmd.Main()
}
