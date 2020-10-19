package logs

import (
	"fmt"
	"os"

	"github.com/xuperchain/xupercore/kernel/common/utils"

	log "github.com/xuperchain/log15"
)

// LogBufSize define log buffer channel size
const LogBufSize = 102400

// OpenLog create and open log stream using LogConfig
func OpenLog(lc *LogConfig) (LogDriver, error) {
	infoFile := lc.Filepath + "/" + lc.Filename + ".log"
	wfFile := lc.Filepath + "/" + lc.Filename + ".log.wf"
	os.MkdirAll(lc.Filepath, os.ModePerm)

	lfmt := log.LogfmtFormat()
	switch lc.Fmt {
	case "json":
		lfmt = log.JsonFormat()
	}

	xlog := log.New("module", lc.Module)
	lvLevel, err := log.LvlFromString(lc.Level)
	if err != nil {
		return nil, fmt.Errorf("log level error.err:%v", err)
	}
	// set lowest level as level limit, this may improve performance
	xlog.SetLevelLimit(lvLevel)

	// init normal and warn/fault log file handler, RotateFileHandler
	// only valid if `RotateInterval` and `RotateBackups` greater than 0
	var (
		nmHandler log.Handler
		wfHandler log.Handler
	)
	if lc.RotateInterval > 0 && lc.RotateBackups > 0 {
		nmHandler = log.Must.RotateFileHandler(
			infoFile, lfmt, lc.RotateInterval, lc.RotateBackups)
		wfHandler = log.Must.RotateFileHandler(
			wfFile, lfmt, lc.RotateInterval, lc.RotateBackups)
	} else {
		nmHandler = log.Must.FileHandler(infoFile, lfmt)
		wfHandler = log.Must.FileHandler(wfFile, lfmt)
	}

	if lc.Async {
		nmHandler = log.BufferedHandler(LogBufSize, nmHandler)
		wfHandler = log.BufferedHandler(LogBufSize, wfHandler)
	}

	// prints log level between `lvLevel` to Info to common log
	nmfileh := log.BoundLvlFilterHandler(lvLevel, log.LvlError, nmHandler)

	// prints log level greater or equal to Warn to wf log
	wffileh := log.LvlFilterHandler(log.LvlWarn, wfHandler)

	var lhd log.Handler
	if lc.Console {
		hstd := log.StreamHandler(os.Stderr, lfmt)
		lhd = log.SyncHandler(log.MultiHandler(hstd, nmfileh, wffileh))
	} else {
		lhd = log.SyncHandler(log.MultiHandler(nmfileh, wffileh))
	}
	xlog.SetHandler(lhd)

	return xlog, err
}

// For unit testing.
const (
	DefLogConfFile = "conf/log.yaml"
)

// For unit testing.
// 设置环境变量XCHAIN_ROOT_PATH，统一从这个目录加载配置和数据
func GetConfFile() string {
	var utPath = utils.GetXchainRootPath()
	if utPath == "" {
		panic("XCHAIN_ROOT_PATH environment variable not set")
	}
	return utPath + DefLogConfFile
}

// For unit testing.
func GetLog() (LogDriver, error) {
	logCfg, err := LoadLogConf(GetConfFile())
	if err != nil {
		return nil, fmt.Errorf("load log config failed.err:%v", err)
	}

	logger, err := OpenLog(logCfg)
	if err != nil {
		return nil, fmt.Errorf("open log fail.err:%v", err)
	}

	return logger, nil
}

// For unit testing.
func GetLogFitter(logger LogDriver) (Logger, error) {
	if logger == nil {
		lg, err := GetLog()
		if err != nil {
			return nil, fmt.Errorf("open log fail.err:%v", err)
		}
		logger = lg
	}

	log, err := NewLogger(logger, GenLogId())
	if err != nil {
		return nil, fmt.Errorf("new logger fail.err:%v", err)
	}

	return log, nil
}
