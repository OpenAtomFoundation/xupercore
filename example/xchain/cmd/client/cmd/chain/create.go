package chain

import (
	"fmt"
	"log"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/utils"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"
	"github.com/xuperchain/xupercore/kernel/common/xconfig"
	"github.com/xuperchain/xupercore/lib/logs"
	_ "github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
	xutils "github.com/xuperchain/xupercore/lib/utils"

	"github.com/spf13/cobra"
)

type CreateChainCmd struct {
	global.BaseCmd
	// 创世块配置文件
	GenesisConf string
	// 环境配置文件
	EnvConf string
}

func GetCreateChainCmd() *CreateChainCmd {
	createChainCmdIns := new(CreateChainCmd)

	subCmd := &cobra.Command{
		Use:           "create",
		Short:         "create chain.",
		Example:       xdef.CmdLineName + " chain create",
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return createChainCmdIns.createChain()
		},
	}
	createChainCmdIns.SetCmd(subCmd)

	// 设置命令行参数并绑定变量
	subCmd.Flags().StringVarP(&createChainCmdIns.GenesisConf,
		"genesis_conf", "g", "./data/genesis/single.json", "genesis config file path")
	subCmd.Flags().StringVarP(&createChainCmdIns.EnvConf,
		"env_conf", "e", "./conf/env.yaml", "env config file path")

	return createChainCmdIns
}

func (t *CreateChainCmd) createChain() error {
	log.Printf("start create chain.bc_name:%s genesis_conf:%s env_conf:%s\n",
		global.GFlagBCName, t.GenesisConf, t.EnvConf)

	if !xutils.FileIsExist(t.GenesisConf) || !xutils.FileIsExist(t.EnvConf) {
		log.Printf("config file not exist.genesis_conf:%s env_conf:%s\n", t.GenesisConf, t.EnvConf)
		return fmt.Errorf("config file not exist")
	}

	econf, err := xconfig.LoadEnvConf(t.EnvConf)
	if err != nil {
		log.Printf("load env config failed.env_conf:%s err:%v\n", t.EnvConf, err)
		return fmt.Errorf("load env config failed")
	}

	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	err = utils.CreateLedger(global.GFlagBCName, t.GenesisConf, econf)
	if err != nil {
		log.Printf("create ledger failed.err:%v\n", err)
		return fmt.Errorf("create ledger failed")
	}

	log.Printf("create ledger succ.bc_name:%s\n", global.GFlagBCName)
	return nil
}
