package mock

import (
    "fmt"
    "log"
    "os"
    "path/filepath"

    xledger "github.com/xuperchain/xupercore/bcs/ledger/xledger/utils"
    xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
    "github.com/xuperchain/xupercore/lib/logs"
    "github.com/xuperchain/xupercore/lib/utils"
)

func NewEnvConfForTest(paths ...string) (*xconf.EnvConf, error) {
	path := "conf/env.yaml"
	if len(paths) > 0 {
		path = paths[0]
	}

	dir := utils.GetCurFileDir()
	econfPath := filepath.Join(dir, path)
	econf, err := xconf.LoadEnvConf(econfPath)
	if err != nil {
	    return nil, err
    }

    econf.RootPath = utils.GetCurFileDir()
    logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))

    if len(paths) > 0 {
        RemoveLedger(econf)
        CreateLedger(econf)
    }

    return econf, nil
}

func GetNetConfPathForTest() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/network.yaml")
}

func CreateLedger(conf *xconf.EnvConf) error {
    dir := utils.GetCurFileDir()
    err := xledger.CreateLedger("xuper", filepath.Join(dir, "genesis/xuper.json"), conf)
    if err != nil {
        log.Printf("create ledger failed.err:%v\n", err)
        return fmt.Errorf("create ledger failed")
    }
    return nil
}

func RemoveLedger(conf *xconf.EnvConf) error {
    path := conf.GenDataAbsPath("blockchain")
    if err := os.RemoveAll(path); err != nil {
        log.Printf("remove ledger failed.err:%v\n", err)
        return err
    }
    return nil
}

func InitLogForTest() error {
	_, err := NewEnvConfForTest()
	if err != nil {
		return err
	}

	return nil
}
