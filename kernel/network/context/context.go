// 明确定义该模块需要的上下文信息，方便代码阅读和理解
package context

import (
	"fmt"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	ncfg "github.com/xuperchain/xupercore/kernel/network/config"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/timer"
)

// 考虑到有些对象是有状态的，需要单例实现
// 有些上下文是领域级别的，有些是操作级别的
// 所以对领域级别的上下文和操作级别的上下文分别定义

// 领域级上下文
type NetCtx struct {
	xctx.BaseCtx
	P2PConf      *ncfg.P2PConfig
	MetricSwicth bool
}

// 必须设置的在参数直接指定，可选的通过对应的Set方法设置
func CreateNetCtx(confPath string) (*NetCtx, error) {
	if confPath == "" {
		return nil, fmt.Errorf("create network context failed because some param are missing")
	}

	// 加载配置
	cfg, err := ncfg.LoadP2PConf(confPath)
	if err != nil {
		return nil, fmt.Errorf("create object context failed because config load fail.err:%v", err)
	}

	ctx := new(NetCtx)
	ctx.XLog, _ = logs.NewLogger("", "network")
	ctx.Timer = timer.NewXTimer()
	ctx.P2PConf = cfg
	// 可选参数设置默认值
	ctx.MetricSwicth = false

	return ctx, nil
}
