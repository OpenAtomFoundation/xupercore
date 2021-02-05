// 明确定义该模块需要的上下文信息，方便代码阅读和理解
package context

import (
	"fmt"

	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	nconf "github.com/xuperchain/xupercore/kernel/network/config"
	"github.com/xuperchain/xupercore/kernel/network/def"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// 网络组件运行上下文环境
type NetCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *xconf.EnvConf
	// 网络组件配置
	P2PConf *nconf.NetConf
}

func NewNetCtx(envCfg *xconf.EnvConf) (*NetCtx, error) {
	if envCfg == nil {
		return nil, fmt.Errorf("create net context failed because env conf is nil")
	}

	// 加载配置
	cfg, err := nconf.LoadP2PConf(envCfg.GenConfFilePath(envCfg.NetConf))
	if err != nil {
		return nil, fmt.Errorf("create net context failed because config load fail.err:%v", err)
	}

	// 配置路径转为绝对路径
	cfg.KeyPath = envCfg.GenDataAbsPath(cfg.KeyPath)

	log, err := logs.NewLogger("", def.SubModName)
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because new logger error. err:%v", err)
	}

	ctx := new(NetCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.EnvCfg = envCfg
	ctx.P2PConf = cfg

	return ctx, nil
}
