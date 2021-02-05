package xutils

import (
	"os"
	"path/filepath"

	"github.com/xuperchain/xupercore/lib/utils"
)

// Set environment variable:X_ROOT_PATH
func GetXRootPath() string {
	rtPath := os.Getenv(XEnvVarRootPath)
	if rtPath != "" && utils.FileIsExist(rtPath) {
		return rtPath
	}

	return ""
}

func GetCurRootDir() string {
	curExecDir := utils.GetCurExecDir()
	return filepath.Dir(curExecDir)
}
