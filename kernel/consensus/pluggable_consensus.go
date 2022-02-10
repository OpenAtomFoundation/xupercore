package consensus

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
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

	ErrInvalidConfig  = errors.New("config should be an empty JSON when rolling back an old one, or try an upper version")
	ErrInvalidVersion = errors.New("version should be an upper one when upgrading a new one")
)

// PluggableConsensus 实现了consensus_interface接口
type PluggableConsensus struct {
	ctx           cctx.ConsensusCtx
	stepConsensus *stepConsensus
}

// NewPluggableConsensus 初次创建PluggableConsensus实例，初始化cons列表
func NewPluggableConsensus(cCtx cctx.ConsensusCtx) (PluggableConsensusInterface, error) {
	if cCtx.BcName == "" {
		cCtx.XLog.Error("Pluggable Consensus::NewPluggableConsensus::bcName is empty.")
	}
	pc := &PluggableConsensus{
		ctx: cCtx,
		stepConsensus: &stepConsensus{
			cons: []ConsensusImplInterface{},
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
	res, _ := xMReader.Get(contractBucket, []byte(consensusKey))
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
func (pc *PluggableConsensus) makeConsensusItem(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) (ConsensusImplInterface, error) {
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
	updateHeight, err := strconv.ParseInt(string(ctxArgs["height"]), 10, 64)
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
		StartHeight:   updateHeight + 1,
	}, nil
}

// updateConsensus 共识升级，更新原有共识列表，向PluggableConsensus共识列表插入新共识，并暂停原共识实例
// 该方法注册到kernel的延时调用合约中，在trigger高度时被调用，此时直接按照共识cfg新建新的共识实例
// 共识version需要递增序列
func (pc *PluggableConsensus) updateConsensus(contractCtx contract.KContext) (*contract.Response, error) {
	// 解析用户合约信息，包括待升级名称name、trigger高度height和待升级配置config
	cfg, err := pc.proposalArgsUnmarshal(contractCtx.Args())
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::proposalArgsUnmarshal error", "error", err)
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}

	// 不允许升级为 pow 类共识
	if cfg.ConsensusName == "pow" {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus can not be pow")
		return common.NewContractErrResponse(common.StatusErr,
				"Pluggable Consensus::updateConsensus target can not be pow"),
			errors.New("updateConsensus target can not be pow")
	}

	// 当前共识如果是pow类共识，不允许升级
	if cur := pc.stepConsensus.tail(); cur != nil {
		if curStatus, err := cur.GetConsensusStatus(); err != nil || curStatus.GetConsensusName() == "pow" {
			pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus current consensus is pow, can not upgrade from pow", "err", err)
			return common.NewContractErrResponse(common.StatusErr,
					"Pluggable Consensus::updateConsensus current consensus is pow"),
				errors.New("updateConsensus can not upgrade from pow")
		}
	}

	// 更新合约存储, 注意, 此次更新需要检查是否是初次升级情况，此时需要把genesisConf也写进map中
	pluggableConfig, _ := contractCtx.Get(contractBucket, []byte(consensusKey))
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

	// 检查生效高度
	if err := pc.checkConsensusHeight(cfg); err != nil {
		pc.ctx.GetLog().Error("Pluggable Consensus::updateConsensus::check consensus height error")
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}

	// 检查新共识配置是否正确
	if err := checkConsensusVersion(c, cfg); err != nil {
		pc.ctx.XLog.Error("Pluggable Consensus::updateConsensus::wrong value, pls check your proposal file.", "error", err)
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}

	// 生成新的共识实例
	consensusItem, err := pc.makeConsensusItem(pc.ctx, c[len(c)-1])
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::make consensu item error! Use old one.", "error", err.Error())
		return common.NewContractErrResponse(common.StatusErr, err.Error()), err
	}
	pc.ctx.XLog.Debug("Pluggable Consensus::updateConsensus::make a new consensus item successfully during updating process.")

	newBytes, err := json.Marshal(c)
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::marshal error", "error", err)
		return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
	}
	if err = contractCtx.Put(contractBucket, []byte(consensusKey), newBytes); err != nil {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::refresh contract storage error", "error", err)
		return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
	}

	// 设置switchflag
	pc.stepConsensus.setSwitch(true)
	// 最后一步再put item到slice
	err = pc.stepConsensus.put(consensusItem)
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable Consensus::updateConsensus::put item into stepConsensus failed", "error", err)
		return common.NewContractErrResponse(common.StatusErr, BuildConsensusError.Error()), BuildConsensusError
	}
	pc.ctx.XLog.Debug("Pluggable Consensus::updateConsensus::key has been modified.", "ConsensusMap", c)
	return common.NewContractOKResponse([]byte("ok")), nil
}

// CheckConsensusConfig 同名配置文件检查:
// 1. 同一个链的共识版本只能增加，不能升级到旧版本
// 2. 将合法的配置写到map中
func checkConsensusVersion(hisMap map[int]def.ConsensusConfig, cfg *def.ConsensusConfig) error {
	var err error
	var newConf configFilter
	if err = json.Unmarshal([]byte(cfg.Config), &newConf); err != nil {
		return errors.New("wrong parameter config")
	}
	newConfVersion, err := strconv.ParseInt(newConf.Version, 10, 64)
	if err != nil {
		return errors.New("wrong parameter version, version should an integer in string")
	}
	// 获取历史最近共识实例，初始状态下历史共识没有version字段的，需手动添加
	var maxVersion int64
	for i := len(hisMap) - 1; i >= 0; i-- {
		configItem := hisMap[i]
		var tmpItem configFilter
		err := json.Unmarshal([]byte(configItem.Config), &tmpItem)
		if err != nil {
			return errors.New("unmarshal config error")
		}
		if tmpItem.Version == "" {
			tmpItem.Version = "0"
		}
		v, _ := strconv.ParseInt(tmpItem.Version, 10, 64)
		if maxVersion < v {
			maxVersion = v
		}
	}
	if maxVersion < newConfVersion {
		hisMap[len(hisMap)] = *cfg
		return nil
	}
	return ErrInvalidVersion
}

type configFilter struct {
	Version string `json:"version,omitempty"`
}

// 检查区块高度，距离上次升级高度 > 20
func (pc *PluggableConsensus) checkConsensusHeight(cfg *def.ConsensusConfig) error {
	con := pc.stepConsensus.tail()
	if con == nil {
		pc.ctx.GetLog().Warn("check consensus height error")
		return errors.New("check consensus height error")
	}
	conStatus, _ := con.GetConsensusStatus()
	if cfg.StartHeight-conStatus.GetConsensusBeginInfo() < 20 {
		pc.ctx.GetLog().Warn("check consensus height error, at least more than 20 block by last ")
		return errors.New("check consensus height error")
	}
	return nil
}

// CompeteMaster 矿工检查当前自己是否需要挖矿
// param: height仅为打印需要的标示，实际还是需要账本当前最高 的高度作为输入
func (pc *PluggableConsensus) CompeteMaster(height int64) (bool, bool, error) {
	con, _ := pc.getCurrentConsensusItem(height)
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::CompeteMaster::Cannot get consensus Instance.")
		return false, false, EmptyConsensusListErr
	}
	return con.CompeteMaster(height)
}

// CheckMinerMatch 调用具体实例的CheckMinerMatch()
func (pc *PluggableConsensus) CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error) {
	con, _ := pc.getCurrentConsensusItem(block.GetHeight())
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::CheckMinerMatch::tail consensus item is empty", "err", EmptyConsensusListErr)
		return false, EmptyConsensusListErr
	}
	return con.CheckMinerMatch(ctx, block)
}

// ProcessBeforeMinerm调用具体实例的ProcessBeforeMiner()
func (pc *PluggableConsensus) ProcessBeforeMiner(height, timestamp int64) ([]byte, []byte, error) {
	con, _ := pc.getCurrentConsensusItem(height)
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::ProcessBeforeMiner::tail consensus item is empty", "err", EmptyConsensusListErr)
		return nil, nil, EmptyConsensusListErr
	}
	return con.ProcessBeforeMiner(height, timestamp)
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
func (pc *PluggableConsensus) CalculateBlock(block cctx.BlockInterface) error {
	con, _ := pc.getCurrentConsensusItem(block.GetHeight())
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::CalculateBlock::tail consensus item is empty", "err", EmptyConsensusListErr)
		return EmptyConsensusListErr
	}
	return con.CalculateBlock(block)
}

// ProcessConfirmBlock 调用具体实例的ProcessConfirmBlock()
func (pc *PluggableConsensus) ProcessConfirmBlock(block cctx.BlockInterface) error {
	con, _ := pc.getCurrentConsensusItem(block.GetHeight())
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::ProcessConfirmBlock::tail consensus item is empty", "err", EmptyConsensusListErr)
		return EmptyConsensusListErr
	}
	return con.ProcessConfirmBlock(block)
}

// GetConsensusStatus 调用具体实例的GetConsensusStatus()，返回接口
func (pc *PluggableConsensus) GetConsensusStatus() (ConsensusStatus, error) {
	block := pc.ctx.Ledger.GetTipBlock()
	con, _ := pc.getCurrentConsensusItem(block.GetHeight() + 1)
	if con == nil {
		pc.ctx.XLog.Error("Pluggable Consensus::GetConsensusStatus::tail consensus item is empty", "err", EmptyConsensusListErr)
		return nil, EmptyConsensusListErr
	}
	return con.GetConsensusStatus()
}

// SwitchConsensus 用于共识升级时切换共识实例
func (pc *PluggableConsensus) SwitchConsensus(height int64) error {
	// 获取最新的共识实例
	con := pc.stepConsensus.tail()
	if con == nil {
		pc.ctx.XLog.Error("pluggable consensus SwitchConsensus stepConsensus.tail error")
		return errors.New("pluggable consensus SwitchConsensus stepConsensus.tail error")
	}

	// 获取最新实例的共识状态
	consensusStatus, err := con.GetConsensusStatus()
	if err != nil {
		pc.ctx.XLog.Error("pluggable consensus SwitchConsensus GetConsensusStatus failed", "error", err)
		return errors.New("pluggable consensus SwitchConsensus GetConsensusStatus failed")
	}
	pc.ctx.XLog.Debug("pluggable consensus SwitchConsensus", "block height", height,
		"current consensus start height", consensusStatus.GetConsensusBeginInfo(), "pc.stepConsensus.getSwitch()", pc.stepConsensus.getSwitch())

	if height >= consensusStatus.GetConsensusBeginInfo()-1 && pc.stepConsensus.getSwitch() {
		pc.ctx.XLog.Debug("pluggable consensus SwitchConsensus switch consensus is true")
		// 由于共识升级切换期间涉及到新老共识并存的问题，如果旷工已经打包更高的区块，那么可以启动新共识，关闭老共识
		preCon := pc.stepConsensus.preTail()
		if preCon != nil {
			_ = preCon.Stop()
			pc.ctx.XLog.Debug("pluggable consensus SwitchConsensus switch stop pre consensus success")
		}
		con := pc.stepConsensus.tail()
		if con == nil {
			pc.ctx.XLog.Error("pluggable consensus SwitchConsensus stepConsensus.tail error")
			return errors.New("pluggable consensus SwitchConsensus stepConsensus.tail error")
		}
		if err := con.Start(); err != nil {
			pc.ctx.XLog.Error("pluggable consensus SwitchConsensus start new consensus failed", "error", err)
			return errors.New("pluggable consensus SwitchConsensus start new consensus failed")
		}
		pc.ctx.XLog.Debug("pluggable consensus SwitchConsensus switch start new consensus success")
		// 关闭共识切换开关
		pc.stepConsensus.setSwitch(false)
	}
	return nil
}

func (pc *PluggableConsensus) getCurrentConsensusItem(height int64) (ConsensusImplInterface, error) {
	con := pc.stepConsensus.tail()
	if con == nil {
		pc.ctx.XLog.Error("pluggable consensus stepConsensus.tail error")
		return nil, errors.New("pluggable consensus stepConsensus.tail error")
	}

	// 获取最新实例的共识状态
	consensusStatus, err := con.GetConsensusStatus()
	if err != nil {
		pc.ctx.XLog.Error("pluggable consensus GetConsensusStatus failed", "error", err)
		return nil, errors.New("pluggable consensus GetConsensusStatus failed")
	}

	// 判断当前区块的高度是否>=最新实例共识起始高度
	if height >= consensusStatus.GetConsensusBeginInfo() {
		return con, nil
	}

	pc.ctx.XLog.Debug("pluggable consensus start use pre consensus", "height", height)
	preCon := pc.stepConsensus.preTail()
	if preCon == nil {
		pc.ctx.XLog.Error("pluggable consensus stepConsensus.preTail error")
		return nil, errors.New("pluggable consensus stepConsensus.preTail error")
	}
	return preCon, nil
}
