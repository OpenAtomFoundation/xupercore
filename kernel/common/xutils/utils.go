package xutils

import (
	"os"

	"github.com/xuperchain/xupercore/lib/utils"
)

// Set environment variable:X_ROOT_PATH
func GetXRootPath() string {
	rtPath := os.Getenv(XEnvVarRootPath)
	if rtPath != "" && utils.FileIsExist(rtPath) {
		return rtPath
	}

	return utils.GetCurExecDir()
}
