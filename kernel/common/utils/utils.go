package utils

import (
	"os"
)

// FileExists reports whether the named file or directory exists.
func FileIsExist(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// Set environment variable:XCHAIN_ROOT_PATH
func GetXchainRootPath() string {
	return os.Getenv(XchainEnvVarRootPath)
}
