package version

// Default build-time variable for library-import.
// This file is overridden on build with build-time informations.
var (
	Version   = ""
	BuildTime = ""
	CommitID  = ""
)

func Version() {
	fmt.Printf("%s-%s %s\n", Version, CommitID, BuildTime)
}
