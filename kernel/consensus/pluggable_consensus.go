package consensus

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	"github.com/xuperchain/xupercore/kernel/contract"
)

const (
	// pluggable_consensus需向三代合约注册的合约方法名, 共识使用三代合约存储作为自己的存储
	// contractUpdateMethod 为更新共识注册，用于在提案-投票成功后，触发共识由原A转换成B
	contractUpdateMethod = "updateConsensus"
	// pluggable_consensus使用的三代kernel合约存储bucket名
	// 共识的key value设计如下
	// key, value直接通过index拿取历史可插拔slice的长度index([0,len-1])，通过index为自增变量为key，对应如下:
	// <"PluggableConfig", configJson> 其中configJson为一个map[int]consensusJson格式，key为自增index，value为对应共识config
	// <index, consensusJson<STRING>> 每个index对应的共识属性，eg. <"1", "{"name":"pow", "config":"{}", "beginHeight":"100"}">
	// consensusJson 格式如下
	// <name, consensusName<STRING>>
	// <config, configJson<STRING>>
	// <beginHeight, height>
	contractBucket = "$consensus"
	consensusKey   = "PluggableConfig"
)

var (
	EmptyConsensusListErr = errors.New("Consensus list of PluggableConsensus is empty.")
	EmptyConsensusName    = errors.New("Consensus name can not be empty")
	EmptyConfig           = errors.New("Config name can not be empty")
	UpdateTriggerError    = errors.New("Update trigger height invalid")
	BeginBlockIdErr       = errors.New("Consensus begin blockid err")
	BuildConsensusError   = errors.New("Build consensus Error")
	ConsensusNotRegister  = errors.New("Consensus hasn't been register. Please use consensus.Register({NAME},{FUNCTION_POINTER}) to register in consensusMap")
	ContractMngErr        = errors.New("Contract manager is empty.")
)

// PluggableConsensus 实现了consensus_interface接口
type PluggableConsensus struct {
	ctx           cctx.ConsensusCtx
	stepConsensus *stepConsensus
}

// NewPluggableConsensus 初次创建PluggableConsensus实例，初始化cons列表
func NewPluggableConsensus(cCtx cctx.ConsensusCtx) (ConsensusInterface, error) {
	if cCtx.BcName == "" {
		cCtx.XLog.Error("Pluggable Consensus::NewPluggableConsensus::bcName is empty.")
	}
	pc := &PluggableConsensus{
		ctx: cCtx,
		stepConsensus: &stepConsensus{
			cons: []ConsensusInterface{},
		},
	}
	if cCtx.Contract.GetKernRegistry() == nil {
		return nil, ContractMngErr
	}
	// 向合约注册升级方法
	cCtx.Contract.GetKernRegistry().RegisterKernMethod(contractBucket, contractUpdateMethod, pc.updateConsensus)
	xMReader, err := cCtx.Ledger.GetTipXMSnapshotReader()
	if err != nil {
		return nil, err
	}
	res, err := xMReader.Get(contractBucket, []byte(consensusKey))
	// 若合约存储不存在，则证明为第一次吊起创建实例，则直接从账本里拿到创始块配置，并且声称从未初始化过的共识实例Genesis共识实例
	if res == nil {
		consensusBuf, err := cCtx.Ledger.GetConsensusConf()
		if err != nil {
			return nil, err
		}
		// 解析提取字段生成ConsensusConfig
		cfg := def.ConsensusConfig{}
		err = json.Unmarshal(consensusBuf, &cfg)
		if err != nil {
			cCtx.XLog.Error("Pluggable Consensus::NewPluggableConsensus::parse consensus configuration error!", "conf", string(consensusBuf), "error", err.Error())
			return nil, err
		}
		cfg.StartHeight = 1
		cfg.Index = 0
		genesisConsensus, err := pc.makeConsensusItem(cCtx, cfg)
		if err != nil {
			cCtx.XLog.Error("Pluggable Consensus::NewPluggableConsensus::make first consensus item error!", "error", err.Error())
			return nil, err
		}
		pc.stepConsensus.put(genesisConsensus)
		// 启动实例
		genesisConsensus.Start()
		cCtx.XLog.Debug("Pluggable Consensus::NewPluggableConsensus::create a instance for the first time.")
		return pc, nil
	}
	// 原合约存储存在，即该链重启，重新恢复pluggable consensus
	c := map[int]def.ConsensusConfig{}
	err = json.Unmarshal(res, &c)
	if err != nil {
		// 历史consensus存储有误，装载无效，此时直接panic
		cCtx.XLog.Error("Pluggable Consensus::history consensus storage invalid, pls check function.")
		return nil, err
	}
	for i := 0; i < len(c); i++ {
		config := c[i]
		oldConsensus, err := pc.makeConsensusItem(cCtx, config)
		if err != nil {
			cCtx.XLog.Warn("Pluggable Consensus::NewPluggableConsensus::make old consensus item error!", "error", err.Error())
		}
		pc.stepConsensus.put(oldConsensus)
		// 最近一次共识实例吊起
		if i == len(c)-1 {
			oldConsensus.Start()
		}
		cCtx.XLog.Debug("Pluggable Consensus::NewPluggableConsensus::create a instance with history reader.", "StepConsensus", pc.stepConsensus)
	}
	return pc, nil
}

// makeConsensusItem 创建单个特定共识，返回特定共识接口
func (pc *PluggableConsensus) makeConsensusItem(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) (base.ConsensusImplInterface, error) {
	if cCfg.ConsensusName == "" {
		cCtx.XLog.Error("Pluggable Consensus::makeConsensusItem::consensus name is empty")
		return nil, EmptyConsensusName
	}
	// 检查version是否相等, 原postUpdateConsensusActions公共部分
	specificCon, err := NewPluginConsensus(pc.ctx, cCfg)
	if err != nil {
		cCtx.XLog.Error("Pluggable Consensus::NewPluginConsensus error", "error", err)
		return nil, err
	}
	if specificCon == nil {
		cCtx.XLog.Error("Pluggable Consensus::NewPluginConsensus::empty error", "error", BuildConsensusError)
		return nil, BuildConsensusError
	}
	cCtx.XLog.Debug("Pluggable Consensus::makeConsensusItem::create a consensus item.", "type", cCfg.ConsensusName)
	return specificCon, nil
}

func (pc *PluggableConsensus) proposalArgsUnmarshal(ctxArgs map[string][]byte) (*def.ConsensusConfig, error) {
	if _, ok := ctxArgs["height"]; !ok {
		return nil, UpdateTriggerError
	}
	consensusHeight, err := strconv.ParseInt(string(ctxArgs["height"]), 10, 64)
	if err != nil {
		pc.ctx.XLog.Error("Pluggable Consensus::updateConsensus::height value invalid.", "err", err)
		return nil, err
	}
	args := make(map[string]interface{})
	err = json.Unmarshal(ctxArgs["args"], &args)
	if err != nil {
		pc.ctx.XLog.Error("Pluggable Consensus::updateConsensus::unmarshal err.", "err", err)
		return nil, err
	}
	if _, ok := args["name"]; !ok {
		return nil, EmptyConsensusName
	}
	if _, ok := args["config"]; !ok {
		return nil, ConsensusNotRegister
	}

	consensusName, ok := args["name"].(string)
	if !ok {
		pc.ctx.XLog.Error("Pluggable Consensus::updateConsensus::name should be string.")
		return nil, EmptyConsensusName
	}
	if _, dup := consensusMap[consensusName]; !dup {
		pc.ctx.XLog.Error("Pluggable Consensus::updateConsensus::consensus's type invalid when update", "name", consensusName)
		return nil, ConsensusNotRegister
	}
	// 解析arg生成用户tx中的共识consensusConfig, 生成新共识实例
	consensusConfigMap, ok := args["config"].(map[string]interface{})
	if !ok {
		pc.ctx.XLog.Error("Pluggable Consensus::updateConsensus::config should be map.")
		return nil, EmptyConfig
	}
	consensusConfigBytes, err := json.Marshal(&consensusConfigMap)
	if err != nil {
		pc.ctx.XLog.Error("Pluggable Consensus::updateConsensus::unmarshal config err.", "err", err)
		return nil, EmptyConfig
	}
	return &def.ConsensusConfig{
		ConsensusName: consensusName,
		Config:        string(consensusConfigBytes),
		Index:         pc.stepConsensus.len(),
		StartHeight:   consensusHeight,
	}, nil
}

// updateConsensus 共识升级，更新原有共识列表，向PluggableConsensus共识列表插入新共识，并暂停原共识实例
// 该方法注册到kernel的延时调用合约中，在trigger高度时被调用，此时直接按照共识cfg新建新的共识实例
func (pc *PluggableConsensus) updateConsensus(contractCtx contract.KContext) (*contract.Response, error) {
	// 解析用户合约信息，包括待升级名称name、trigger高度height和待升级配置config
	cfg, err := pc.proposalArgsUnmarshal(contractCtx.Args())
	if err != nil {
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	consensusItem, err := pc.makeConsensusItem(pc.ctx, *cfg)
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::make consensu item error! Use old one.", "error", err.Error())
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	transCon, ok := consensusItem.(ConsensusInterface)
	if !ok {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::consensus transfer error! Use old one.")
		return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
	}
	pc.ctx.XLog.Debug("Pluggable Consensus::updateConsensus::make a new consensus item successfully during updating process.")

	// 更新合约存储, 注意, 此次更新需要检查是否是初次升级情况，此时需要把genesisConf也写进map中
	pluggableConfig, err := contractCtx.Get(contractBucket, []byte(consensusKey))
	c := map[int]def.ConsensusConfig{}
	if pluggableConfig == nil {
		// 尚未写入过任何值，此时需要先写入genesisConfig，即初始共识配置值, 此处不存在err情况
		consensusBuf, _ := pc.ctx.Ledger.GetConsensusConf()
		// 解析提取字段生成ConsensusConfig
		config := def.ConsensusConfig{}
		_ = json.Unmarshal(consensusBuf, &config)
		config.StartHeight = 1
		config.Index = 0
		c[0] = config
	} else {
		err = json.Unmarshal(pluggableConfig, &c)
		if err != nil {
			pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::unmarshal error", "error", err)
			return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
		}
	}
	// 检查新共识，仅允许共识升级为老共识实例，不允许升级为同名新共识实例，如Tdpos升级成Xpos升级成Tdpos，两个Tdpos配置必须相同
	if checkSameNameConsensus(c, cfg) {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::same name consensus.")
		return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
	}
	c[len(c)] = *cfg
	newBytes, err := json.Marshal(c)
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::marshal error", "error", err)
		return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
	}
	if err = contractCtx.Put(contractBucket, []byte(consensusKey), newBytes); err != nil {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::refresh contract storage error", "error", err)
		return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
	}

	// 获取当前的共识实例, 停止上一共识实例，主要包括注册的P2P msg等
	lastCon, ok := pc.getCurrentConsensusComponent().(base.ConsensusImplInterface)
	if !ok {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::last consensus transfer error! Stop.")
		return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
	}
	lastCon.Stop()

	// 最后一步再put item到slice
	err = pc.stepConsensus.put(transCon)
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::put item into stepConsensus failed", "error", err)
		return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
	}
	// 此时再将当前待升级的共识实例start起来
	consensusItem.Start()
	pc.ctx.XLog.Debug("Pluggable Consensus::updateConsensus::key has been modified.", "ConsensusMap", c)
	return common.NewContractOKResponse([]byte("ok")), nil
}

// rollbackConsensus
// TODO: 共识回滚，更新原有共识列表，遍历PluggableConsensus共识列表并删除目标高度以上的共识实例，并启动原共识实例
func (pc *PluggableConsensus) rollbackConsensus(contractCtx contract.KContext) error {
	return nil
}

// 获取目前PluggableConsensus共识列表共识句柄
func (pc *PluggableConsensus) getCurrentConsensusComponent() ConsensusInterface {
	return pc.stepConsensus.tail()
}

// CompeteMaster 矿工检查当前自己是否需要挖矿
// param: height仅为打印需要的标示，实际还是需要账本当前最高 的高度作为输入
func (pc *PluggableConsensus) CompeteMaster(height int64) (bool, bool, error) {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::CompeteMaster::Cannot get consensus Instance.")
		return false, false, EmptyConsensusListErr
	}
	return con.CompeteMaster(height)
}

// CheckMinerMatch 调用具体实例的CheckMinerMatch()
func (pc *PluggableConsensus) CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error) {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::CheckMinerMatch::tail consensus item is empty", "err", EmptyConsensusListErr)
		return false, EmptyConsensusListErr
	}
	return con.CheckMinerMatch(ctx, block)
}

// ProcessBeforeMinerm调用具体实例的ProcessBeforeMiner()
func (pc *PluggableConsensus) ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error) {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::ProcessBeforeMiner::tail consensus item is empty", "err", EmptyConsensusListErr)
		return nil, nil, EmptyConsensusListErr
	}
	return con.ProcessBeforeMiner(timestamp)
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
func (pc *PluggableConsensus) CalculateBlock(block cctx.BlockInterface) error {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::CalculateBlock::tail consensus item is empty", "err", EmptyConsensusListErr)
		return EmptyConsensusListErr
	}
	return con.CalculateBlock(block)
}

// ProcessConfirmBlock 调用具体实例的ProcessConfirmBlock()
func (pc *PluggableConsensus) ProcessConfirmBlock(block cctx.BlockInterface) error {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::ProcessConfirmBlock::tail consensus item is empty", "err", EmptyConsensusListErr)
		return EmptyConsensusListErr
	}
	return con.ProcessConfirmBlock(block)
}

// GetConsensusStatus 调用具体实例的GetConsensusStatus()，返回接口
func (pc *PluggableConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	con := pc.getCurrentConsensusComponent()
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::GetConsensusStatus::tail consensus item is empty", "err", EmptyConsensusListErr)
		return nil, EmptyConsensusListErr
	}
	return con.GetConsensusStatus()
}

/////////////////// stepConsensus //////////////////

// stepConsensus 封装了可插拔共识需要的共识数组
type stepConsensus struct {
	cons []ConsensusInterface
	// mutex保护StepConsensus数据结构cons的读写操作
	mutex sync.RWMutex
}

// 向可插拔共识数组put item
func (sc *stepConsensus) put(con ConsensusInterface) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.cons = append(sc.cons, con)
	return nil
}

// 获取最新的共识实例
func (sc *stepConsensus) tail() ConsensusInterface {
	//getCurrentConsensusComponent
	sc.mutex.RLock()
	sc.mutex.RUnlock()
	if len(sc.cons) == 0 {
		return nil
	}
	return sc.cons[len(sc.cons)-1]
}

func (sc *stepConsensus) len() int {
	sc.mutex.RLock()
	sc.mutex.RUnlock()
	return len(sc.cons)
}

var consensusMap = make(map[string]NewStepConsensus)

type NewStepConsensus func(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) base.ConsensusImplInterface

// Register 不同类型的共识需要提前完成注册
func Register(name string, f NewStepConsensus) error {
	if f == nil {
		panic("Pluggable Consensus::Register::new function is nil")
	}
	if _, dup := consensusMap[name]; dup {
		panic("Pluggable Consensus::Register::called twice for func " + name)
	}
	consensusMap[name] = f
	return nil
}

// NewPluginConsensus 新建可插拔共识实例
func NewPluginConsensus(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) (base.ConsensusImplInterface, error) {
	if cCfg.ConsensusName == "" {
		return nil, EmptyConsensusName
	}
	if cCfg.StartHeight < 0 {
		return nil, BeginBlockIdErr
	}
	if f, ok := consensusMap[cCfg.ConsensusName]; ok {
		return f(cCtx, cCfg), nil
	}
	return nil, ConsensusNotRegister
}

// checkSameNameConsensus 不允许同名但配置文件不同的共识新组件
func checkSameNameConsensus(hisMap map[int]def.ConsensusConfig, cfg *def.ConsensusConfig) bool {
	for k, v := range hisMap {
		if v.ConsensusName != cfg.ConsensusName {
			continue
		}
		if v.Config == cfg.Config {
			if k != len(hisMap)-1 { // 允许回到历史共识实例
				return false
			}
			return true // 不允许相同共识配置的升级，无意义
		}
		// 比对Version和bft组件是否一致即可
		type tempStruct struct {
			EnableBFT map[string]bool `json:"bft_config,omitempty"`
		}
		var newConf tempStruct
		if err := json.Unmarshal([]byte(cfg.Config), &newConf); err != nil {
			return true
		}
		var oldConf tempStruct
		if err := json.Unmarshal([]byte(v.Config), &oldConf); err != nil {
			return true
		}
		// 共识名称相同，注意: xpos和tdpos在name上都称为tdpos，但xpos的enableBFT!=nil
		if (newConf.EnableBFT != nil && oldConf.EnableBFT != nil) || (newConf.EnableBFT == nil && oldConf.EnableBFT == nil) {
			return true // 不允许同一共识名称的升级，如Tdpos不可以做配置升级，只能做配置回滚，如升级到xpos再升级回原来的tdpos
		}
	}
	return false
}
