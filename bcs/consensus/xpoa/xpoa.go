package xpoa

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus"
	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	chainedBftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	quorumcert "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/storage"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

func init() {
	consensus.Register("xpoa", NewXpoaConsensus)
}

type xpoaConsensus struct {
	cCtx          cctx.ConsensusCtx
	bcName        string
	election      *xpoaSchedule
	smr           *chainedBft.Smr
	isProduce     map[int64]bool
	config        *xpoaConfig
	initTimestamp int64
	status        *XpoaStatus
	contract      contract.Manager
	kMethod       map[string]contract.KernMethod
	log           logs.Logger
}

// NewXpoaConsensus 初始化实例
func NewXpoaConsensus(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) consensus.ConsensusImplInterface {
	// 解析config中需要的字段
	if cCtx.XLog == nil {
		return nil
	}
	// TODO:cCtx.BcName需要注册表吗？
	if cCtx.Crypto == nil || cCtx.Address == nil {
		cCtx.XLog.Error("consensus:xpoa:NewXpoaConsensus: CryptoClient in context is nil")
		return nil
	}
	if cCtx.Ledger == nil {
		cCtx.XLog.Error("consensus:xpoa:NewXpoaConsensus: Ledger in context is nil")
		return nil
	}
	if cCfg.ConsensusName != "xpoa" {
		cCtx.XLog.Error("consensus:xpoa:NewXpoaConsensus: consensus name in config is wrong", "name", cCfg.ConsensusName)
		return nil
	}

	// 创建smr实例过程
	// 解析xpoaconfig
	xconfig := &xpoaConfig{}
	err := json.Unmarshal([]byte(cCfg.Config), xconfig)
	if err != nil {
		cCtx.XLog.Error("consensus:xpoa:NewXpoaConsensus: xpoa struct unmarshal error", "error", err)
		return nil
	}

	// 校验初始候选人节点列表
	if len(xconfig.InitProposer.Address) <= 0 {
		cCtx.XLog.Error("consensus:xpoa:NewXpoaConsensus: config init_proposer.address is required")
		return nil
	}

	version, err := ParseVersion(cCfg.Config)
	if err != nil {
		cCtx.XLog.Error("consensus:xpoa:NewXpoaConsensus: version error", "error", err)
		return nil
	}
	// create xpoaSchedule
	schedule := NewXpoaSchedule(xconfig, cCtx, cCfg.StartHeight, version)
	if schedule == nil {
		cCtx.XLog.Error("consensus:xpoa:NewXpoaSchedule error")
		return nil
	}
	// 创建status实例
	status := &XpoaStatus{
		Name:        "poa",
		Version:     version,
		StartHeight: cCfg.StartHeight,
		Index:       cCfg.Index,
		election:    schedule,
	}
	if schedule.enableBFT {
		status.Name = "xpoa"
	}
	// create xpoaConsensus实例
	xpoa := &xpoaConsensus{
		cCtx:          cCtx,
		bcName:        cCtx.BcName,
		election:      schedule,
		isProduce:     make(map[int64]bool),
		config:        xconfig,
		initTimestamp: time.Now().UnixNano(),
		status:        status,
		contract:      cCtx.Contract,
		log:           cCtx.XLog,
	}

	xpoaKMethods := map[string]contract.KernMethod{
		contractEditValidate: xpoa.methodEditValidates,
		contractGetValidates: xpoa.methodGetValidates,
	}

	xpoa.kMethod = xpoaKMethods

	// 凡属于共识升级的逻辑，新建的Xpoa实例将直接将当前值置为true，原因是上一共识模块已经在当前值生成了高度为trigger height的区块，新的实例会再生成一边
	timeKey := time.Now().Sub(time.Unix(0, 0)).Milliseconds() / xpoa.config.Period
	xpoa.isProduce[timeKey] = true

	cCtx.XLog.Debug("consensus:xpoa:NewXpoaConsensus: create a poa instance successfully!")
	return xpoa
}

func (x *xpoaConsensus) initBFT() error {
	// create smr/ chained-bft实例, 需要新建CBFTCrypto、pacemaker和saftyrules实例
	cryptoClient := cCrypto.NewCBFTCrypto(x.cCtx.Address, x.cCtx.Crypto)
	qcTree := quorumcert.InitQCTree(x.status.StartHeight, x.cCtx.Ledger, x.cCtx.XLog)
	if qcTree == nil {
		x.log.Error("consensus:xpoa:NewXpoaConsensus: init QCTree err", "startHeight", x.status.StartHeight)
		return nil
	}
	pacemaker := &chainedBft.DefaultPaceMaker{
		CurrentView: x.status.StartHeight,
	}
	// 重启状态检查1，pacemaker需要重置
	tipHeight := x.cCtx.Ledger.QueryTipBlockHeader().GetHeight()
	if !bytes.Equal(qcTree.GetGenesisQC().In.GetProposalId(), qcTree.GetRootQC().In.GetProposalId()) {
		pacemaker.CurrentView = tipHeight - 1
	}
	saftyrules := &chainedBft.DefaultSaftyRules{
		Crypto: cryptoClient,
		QcTree: qcTree,
		Log:    x.cCtx.XLog,
	}
	smr := chainedBft.NewSmr(x.cCtx.BcName, x.election.address, x.log, x.cCtx.Network, cryptoClient, pacemaker, saftyrules, x.election, qcTree)
	// 重启状态检查2，重做tipBlock，此时需重装载justify签名
	if !bytes.Equal(qcTree.GetGenesisQC().In.GetProposalId(), qcTree.GetRootQC().In.GetProposalId()) {
		for i := int64(0); i < 3; i++ {
			b, err := x.cCtx.Ledger.QueryBlockHeaderByHeight(tipHeight - i)
			if err != nil {
				break
			}
			smr.LoadVotes(b.GetPreHash(), x.GetJustifySigns(b))
		}
	}
	x.smr = smr
	x.smr.Start()
	return nil
}

// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
func (x *xpoaConsensus) CompeteMaster(height int64) (bool, bool, error) {
Again:
	t := time.Now().UnixNano() / int64(time.Millisecond)
	key := t / x.election.period
	sleep := x.election.period - t%x.election.period
	if sleep > MAXSLEEPTIME {
		sleep = MAXSLEEPTIME
	}
	v, ok := x.isProduce[key]
	if !ok || !v {
		x.isProduce[key] = true
	} else {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		// 定期清理isProduce
		common.CleanProduceMap(x.isProduce, x.election.period)
		goto Again
	}

	// update validates
	tipBlock := x.election.ledger.GetTipBlock()
	if x.election.UpdateValidator(tipBlock.GetHeight()) {
		x.log.Debug("consensus:xpoa:CompeteMaster: change validators", "valisators", x.election.validators)
	}
	_, pos, blockPos := x.election.minerScheduling(time.Now().UnixNano(), len(x.election.validators))
	if blockPos > x.election.blockNum || pos >= int64(len(x.election.validators)) {
		x.log.Debug("consensus:xpoa:CompeteMaster: minerScheduling err", "pos", pos, "blockPos", blockPos)
		goto Again
	}
	x.election.miner = x.election.validators[pos]
	if x.election.miner == x.election.address {
		x.log.Debug("consensus:xpoa:CompeteMaster", "isMiner", true, "height", tipBlock.GetHeight())
		needSync := tipBlock.GetHeight() == 0 || string(tipBlock.GetProposer()) != x.election.miner
		return true, needSync, nil
	}
	x.log.Debug("consensus:xpoa:CompeteMaster", "isMiner", false, "height", tipBlock.GetHeight())
	return false, false, nil
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
func (x *xpoaConsensus) CalculateBlock(block cctx.BlockInterface) error {
	return nil
}

// CheckMinerMatch 查看block是否合法
func (x *xpoaConsensus) CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error) {
	// 获取block中共识专有存储, 检查justify是否符合要求
	conStoreBytes, _ := block.GetConsensusStorage()
	// 验证矿工身份
	proposer := x.election.GetLocalLeader(block.GetTimestamp(), block.GetHeight(), conStoreBytes)
	if proposer != string(block.GetProposer()) {
		ctx.GetLog().Error("consensus:xpoa:CheckMinerMatch: calculate proposer error", "logid", ctx.GetLog().GetLogId(), "want", proposer,
			"have", string(block.GetProposer()), "blockId", utils.F(block.GetBlockid()))
		return false, MinerSelectErr
	}
	if !x.election.enableBFT {
		return true, nil
	}
	// 验证BFT时，需要除开初始化后的第一个block验证，此时没有justify值
	if block.GetHeight() <= x.status.StartHeight {
		return true, nil
	}
	// 兼容老的结构
	justify, err := common.OldQCToNew(conStoreBytes)
	if err != nil {
		ctx.GetLog().Error("consensus:xpoa:CheckMinerMatch: OldQCToNew error.", "logid", ctx.GetLog().GetLogId(), "err", err,
			"blockId", utils.F(block.GetBlockid()))
		return false, err
	}
	preBlock, _ := x.election.ledger.QueryBlockHeader(block.GetPreHash())
	preConStoreBytes, _ := preBlock.GetConsensusStorage()
	validators, _ := x.election.GetLocalValidates(preBlock.GetTimestamp(), justify.GetProposalView(), preConStoreBytes)

	// 包装成统一入口访问smr
	err = x.smr.CheckProposal(block, justify, validators)
	if err != nil {
		x.log.Error("consensus:xpoa:CheckMinerMatch: bft IsQuorumCertValidate failed", "proposalQC:[height]", block.GetHeight(),
			"proposalQC:[id]", utils.F(block.GetBlockid()), "justifyQC:[height]", justify.GetProposalView(),
			"justifyQC:[id]", utils.F(justify.GetProposalId()), "error", err)
		return false, err
	}
	return true, nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回truncate目标(如需裁剪), 返回写consensusStorage, 返回err
func (x *xpoaConsensus) ProcessBeforeMiner(height, timestamp int64) ([]byte, []byte, error) {
	if !x.election.enableBFT {
		return nil, nil, nil
	}
	// 即本地smr的HightQC和账本TipId不相等，tipId尚未收集到足够签名，回滚到本地HighQC，重做区块
	tipBlock := x.election.ledger.GetTipBlock()
	// smr返回一个裁剪目标，供miner模块直接回滚并出块
	truncate, qc, err := x.smr.ResetProposerStatus(tipBlock, x.election.ledger.QueryBlockHeader, x.election.validators)
	if err != nil {
		return nil, nil, err
	}
	// 候选人组仅一个时无需操作
	if qc == nil {
		return nil, nil, nil
	}

	qcQuorumCert, _ := qc.(*quorumcert.QuorumCert)
	oldQC, _ := common.NewToOldQC(qcQuorumCert)
	storage := common.ConsensusStorage{
		Justify: oldQC,
	}
	// 重做时还需要装载标定节点TipHeight，复用TargetBits作为回滚记录，便于追块时获取准确快照高度
	if truncate {
		x.log.Warn("consensus:xpoa:ProcessBeforeMiner: last block not confirmed, walk to previous block",
			"target", utils.F(qc.GetProposalId()), "ledger", tipBlock.GetHeight())
		storage.TargetBits = int32(tipBlock.GetHeight())
		bytes, _ := json.Marshal(storage)
		return qc.GetProposalId(), bytes, nil
	}
	bytes, _ := json.Marshal(storage)
	return nil, bytes, nil
}

// ProcessConfirmBlock 用于确认块后进行相应的处理
func (x *xpoaConsensus) ProcessConfirmBlock(block cctx.BlockInterface) error {
	if !x.election.enableBFT {
		return nil
	}
	// confirm的第一步：不管是否为当前Leader，都需要更新本地voteQC树，保证当前block的justify votes被写入本地账本
	// 获取block中共识专有存储, 检查justify是否符合要求
	justifyBytes, err := block.GetConsensusStorage()
	if err != nil && block.GetHeight() != x.status.StartHeight {
		x.log.Warn("consensus:xpoa:CheckMinerMatch: parse storage error", "err", err, "blockId", utils.F(block.GetBlockid()))
		return err
	}
	var justify quorumcert.QuorumCertInterface
	if justifyBytes != nil && block.GetHeight() > x.status.StartHeight {
		justify, err = common.OldQCToNew(justifyBytes)
		if err != nil {
			x.log.Error("consensus:xpoa:ProcessConfirmBlock: OldQCToNew error", "err", err, "blockId", utils.F(block.GetBlockid()))
			return err
		}
	}

	// 查看本地是否是最新round的生产者
	_, pos, blockPos := x.election.minerScheduling(block.GetTimestamp(), len(x.election.validators))
	if blockPos > x.election.blockNum || pos >= int64(len(x.election.validators)) {
		x.log.Debug("consensus:xpoa:smr::ProcessConfirmBlock: minerScheduling overflow.")
		return scheduleErr
	}

	var minerValidator []string
	// 如果是当前矿工，则发送Proposal消息
	if x.election.validators[pos] == x.election.address && string(block.GetProposer()) == x.election.address {
		minerValidator = x.election.GetValidators(block.GetHeight() + 1)
	}

	// 包装成统一入口访问smr
	if err := x.smr.KeepUpWithBlock(block, justify, minerValidator); err != nil {
		x.log.Warn("consensus:xpoa:ProcessConfirmBlock: update smr error.", "error", err)
		return err
	}
	return nil
}

// 共识实例的启动逻辑
func (x *xpoaConsensus) Start() error {
	// 注册合约方法
	for method, f := range x.kMethod {
		// 若有历史句柄，删除老句柄
		x.contract.GetKernRegistry().UnregisterKernMethod(x.election.bindContractBucket, method)
		x.contract.GetKernRegistry().RegisterKernMethod(x.election.bindContractBucket, method, f)
	}
	if x.election.enableBFT {
		err := x.initBFT()
		if err != nil {
			x.log.Warn("xpoaConsensus start init bft error", "err", err.Error())
			return err
		}
	}
	return nil
}

// 共识实例的挂起逻辑, 另: 若共识实例发现绑定block结构有误，会直接停掉当前共识实例并panic
func (x *xpoaConsensus) Stop() error {
	// 注销合约方法
	for method := range x.kMethod {
		// 若有历史句柄，删除老句柄
		x.contract.GetKernRegistry().UnregisterKernMethod(x.election.bindContractBucket, method)
	}
	if x.election.enableBFT {
		x.smr.Stop()
	}
	return nil
}

// 共识占用blockinterface的专有存储，特定共识需要提供parse接口，在此作为接口高亮
func (x *xpoaConsensus) ParseConsensusStorage(block cctx.BlockInterface) (interface{}, error) {
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil, err
	}
	justify, err := common.ParseOldQCStorage(b)
	if err != nil {
		return nil, err
	}
	return justify, nil
}

func (x *xpoaConsensus) GetConsensusStatus() (consensus.ConsensusStatus, error) {
	return x.status, nil
}

func (x *xpoaConsensus) GetJustifySigns(block cctx.BlockInterface) []*chainedBftPb.QuorumCertSign {
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil
	}
	signs := common.OldSignToNew(b)
	return signs
}
