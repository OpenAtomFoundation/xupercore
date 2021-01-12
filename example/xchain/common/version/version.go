package version

import (
	"fmt"
)

// Default build-time variable for library-import.
// This file is overridden on build with build-time informations.
var (
	Version   = ""
	BuildTime = ""
	CommitID  = ""
)

func PrintVersion() {
	fmt.Printf("%s-%s %s\n", Version, CommitID, BuildTime)
}
