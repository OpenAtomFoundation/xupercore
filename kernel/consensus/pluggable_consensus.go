package consensus

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

var (
	EmptyConsensusListErr = errors.New("Consensus list of PluggableConsensus is empty.")
	EmptyConsensusName    = errors.New("Consensus name can not be empty")
	BeginHeightErr        = errors.New("Consensus begin height <= 0")
	BeginBlockIdErr       = errors.New("Consensus begin blockid err")
	BuildConsensusError   = errors.New("Build consensus Error")
	UpdateTriggerError    = errors.New("Update trigger height invalid")
	ConsensusNotRegister  = errors.New("consensus hasn't been register. Please use consensus.Register({NAME},{FUNCTION_POINTER}) to register in consensusMap")
)

// StepConsensus封装了可插拔共识需要的共识数组
type StepConsensus struct {
	cons      []ConsensusInterface
	stepIndex int64        // 共识指针
	mutex     sync.RWMutex // mutex保护StepConsensus数据结构cons的读写操作
}

// 向可插拔共识数组put item
func (sc *StepConsensus) put(con ConsensusInterface) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.cons = append(sc.cons, con)
	sc.stepIndex++
	return nil
}

// 获取最新的共识实例
func (sc *StepConsensus) get() ConsensusInterface {
	//getCurrentConsensusComponent
	sc.mutex.RLock()
	sc.mutex.RUnlock()
	if len(sc.cons) == 0 {
		return nil
	}
	return sc.cons[sc.stepIndex-1]
}

/* PluggableConsensus 实现了consensus_interface接口
 */
type PluggableConsensus struct {
	ctx           cctx.ConsensusCtx
	stepConsensus *StepConsensus
	// nextHeight的写操作在CompeteMaster，读操作在updateConsensus中，外层调用并无并发，故无需锁
	nextHeight int64
}

// 获取目前PluggableConsensus共识列表共识句柄
func (pc PluggableConsensus) getCurrentConsensusComponent() ConsensusInterface {
	return pc.stepConsensus.get()
}

/* CompeteMaster 矿工检查当前自己是否需要挖矿
 * param: height仅为打印需要的标示，实际还是需要账本当前最高的高度作为输入
 */
func (pc PluggableConsensus) CompeteMaster(height int64) (bool, bool, error) {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		return false, false, EmptyConsensusListErr
	}
	pc.nextHeight = height
	return con.CompeteMaster(height)
}

// 调用具体实例的CheckMinerMatch()
func (pc PluggableConsensus) CheckMinerMatch(ctx xcontext.BaseCtx, block cctx.BlockInterface) (bool, error) {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		return false, EmptyConsensusListErr
	}
	return con.CheckMinerMatch(ctx, block)
}

// 调用具体实例的ProcessBeforeMiner()
func (pc PluggableConsensus) ProcessBeforeMiner(timestamp int64) (map[string]interface{}, bool, error) {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		return nil, false, EmptyConsensusListErr
	}
	return con.ProcessBeforeMiner(timestamp)
}

// 调用具体实例的ProcessConfirmBlock()
func (pc PluggableConsensus) ProcessConfirmBlock(block cctx.BlockInterface) error {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		return EmptyConsensusListErr
	}
	return con.ProcessConfirmBlock(block)
}

// 调用具体实例的GetConsensusStatus()
func (pc PluggableConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		return nil, EmptyConsensusListErr
	}
	return con.GetConsensusStatus()
}

/* updateConsensus
 * 共识升级，更新原有共识列表，向PluggableConsensus共识列表插入新共识，并暂停原共识实例
 * 该方法注册到kernel的延时调用合约中，在trigger高度时被调用，此时直接按照共识cfg新建新的共识实例
 */
func (pc PluggableConsensus) updateConsensus(contractCtx cctx.FakeKContext, height int64) error {
	if height > pc.nextHeight {
		pc.ctx.BCtx.XLog.Warn("Pluggable Consensus::updateConsensus::trigger height error! Use old one.", "pluggable height", pc.nextHeight, "trigger height", height)
		return UpdateTriggerError
	}
	args := contractCtx.Arg()
	// 解析arg生成用户tx中的共识consensusConfig
	cfg := cctx.ConsensusConfig{}
	err := json.Unmarshal(args, &cfg)
	if err != nil {
		pc.ctx.BCtx.XLog.Warn("Pluggable Consensus::updateConsensus::parse consensus configuration error! Use old one.", "error", err.Error())
		return err
	}
	cfg.BeginBlockid = pc.ctx.Ledger.QueryBlockByHeight(height).GetBlockid()
	consensusItem, err := pc.makeConsensusItem(pc.ctx, cfg)
	if err != nil {
		pc.ctx.BCtx.XLog.Warn("Pluggable Consensus::updateConsensus::make consensu item error! Use old one.", "error", err.Error())
		return err
	}
	transCon, ok := consensusItem.(ConsensusInterface)
	if !ok {
		pc.ctx.BCtx.XLog.Warn("Pluggable Consensus::updateConsensus::consensus transfer error! Use old one.")
		return BuildConsensusError
	}
	return pc.stepConsensus.put(transCon)
}

// 初次创建PluggableConsensus实例，初始化cons列表
func NewPluggableConsensus(cCtx cctx.ConsensusCtx) (ConsensusInterface, error) {
	pc := PluggableConsensus{
		ctx: cCtx,
		stepConsensus: &StepConsensus{
			cons:      []ConsensusInterface{},
			stepIndex: int64(0),
		},
	}
	consensusBuf := cCtx.Ledger.GetConsensusConf()
	// 解析提取字段生成ConsensusConfig
	cfg := cctx.ConsensusConfig{}
	err := json.Unmarshal(consensusBuf, &cfg)
	if err != nil {
		pc.ctx.BCtx.XLog.Error("Pluggable Consensus::NewPluggableConsensus::parse consensus configuration error!", "error", err.Error())
		return nil, err
	}
	cfg.BeginBlockid = pc.ctx.Ledger.GetGenesisBlock().GetBlockid()
	consensusItem, err := pc.makeConsensusItem(cCtx, cfg)
	if err != nil {
		pc.ctx.BCtx.XLog.Error("Pluggable Consensus::NewPluggableConsensus::make first consensus item error!", "error", err.Error())
		return nil, err
	}
	pc.stepConsensus.put(consensusItem)
	return pc, nil
}

// 创建单个特定共识，返回特定共识接口
func (pc PluggableConsensus) makeConsensusItem(cCtx cctx.ConsensusCtx, cCfg cctx.ConsensusConfig) (base.ConsensusImplInterface, error) {
	if cCfg.ConsensusName == "" {
		cCtx.BCtx.XLog.Error("Pluggable Consensus::makeConsensusItem::consensus name is empty")
		return nil, EmptyConsensusName
	}
	// 检查version是否相等, 原postUpdateConsensusActions公共部分
	specificCon, err := NewPluginConsensus(pc.ctx, cCfg)
	if err != nil {
		cCtx.BCtx.XLog.Error("Pluggable Consensus::NewPluginConsensus error", "error", err)
		return nil, err
	}
	return specificCon, nil
}

var consensusMap = make(map[string]NewStepConsensus)

type NewStepConsensus func(cCtx cctx.ConsensusCtx, cCfg cctx.ConsensusConfig) base.ConsensusImplInterface

// 不同类型的共识需要提前完成注册
func Register(name string, f NewStepConsensus) {
	if f == nil {
		panic("Pluggable Consensus::Register::new function is nil")
	}
	if _, dup := consensusMap[name]; dup {
		panic("Pluggable Consensus::Register::called twice for func " + name)
	}
	consensusMap[name] = f
}

// 新建可插拔共识实例
func NewPluginConsensus(cCtx cctx.ConsensusCtx, cCfg cctx.ConsensusConfig) (base.ConsensusImplInterface, error) {
	if cCfg.ConsensusName == "" {
		return nil, EmptyConsensusName
	}
	if cCfg.BeginHeight <= 0 {
		return nil, BeginHeightErr
	}
	if cCfg.BeginBlockid == nil {
		return nil, BeginBlockIdErr
	}
	if f, ok := consensusMap[cCfg.ConsensusName]; ok {
		return f(cCtx, cCfg), nil
	}
	return nil, ConsensusNotRegister
}
