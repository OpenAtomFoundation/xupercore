package logs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	log "github.com/xuperchain/log15"
	lconf "github.com/xuperchain/xupercore/lib/logs/config"
)

// 日志实例采用单例模式
var logHandle LogDriver
var logConf *lconf.LogConf
var once sync.Once
var lock sync.RWMutex

// 基础初始化操作，出现错误会panic抛出异常
func InitLog(cfgFile, logDir string) {
	lock.Lock()
	defer lock.Unlock()

	// 避免重复调用时重复初始化
	once.Do(func() {
		// 加载日志配置
		cfg, err := lconf.LoadLogConf(cfgFile)
		if err != nil {
			panic(fmt.Sprintf("Load log config fail.path:%s err:%s", cfgFile, err))
		}
		logConf = cfg

		// 创建日志实例
		lg, err := OpenLog(logConf, logDir)
		if err != nil {
			panic(fmt.Sprintf("Open log fail.dir:%s err:%s", logDir, err))
		}
		logHandle = lg
	})
}

// OpenLog create and open log stream using LogConfig
func OpenLog(lc *lconf.LogConf, logDir string) (LogDriver, error) {
	infoFile := filepath.Join(logDir, lc.Filename+".log")
	wfFile := filepath.Join(logDir, lc.Filename+".log.wf")
	os.MkdirAll(logDir, os.ModePerm)

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
	var nmHandler, wfHandler log.Handler
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
		nmHandler = log.BufferedHandler(lc.BufSize, nmHandler)
		wfHandler = log.BufferedHandler(lc.BufSize, wfHandler)
	}

	// prints log level between `lvLevel` to Info to common log
	nmfileh := log.BoundLvlFilterHandler(lvLevel, log.LvlInfo, nmHandler)
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
