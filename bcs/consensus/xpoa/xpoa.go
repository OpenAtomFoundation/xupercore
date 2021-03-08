package xpoa

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	chainedBftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	"github.com/xuperchain/xupercore/lib/utils"
)

func init() {
	consensus.Register("xpoa", NewXpoaConsensus)
}

type xpoaConsensus struct {
	bctx          xcontext.XContext
	election      *xpoaSchedule
	smr           *chainedBft.Smr
	isProduce     map[int64]bool
	config        *xpoaConfig
	initTimestamp int64
	status        *XpoaStatus
}

// NewXpoaConsensus 初始化实例
func NewXpoaConsensus(cCtx context.ConsensusCtx, cCfg def.ConsensusConfig) base.ConsensusImplInterface {
	// 解析config中需要的字段
	if cCtx.XLog == nil {
		return nil
	}
	// TODO:cCtx.BcName需要注册表吗？
	if cCtx.Crypto == nil || cCtx.Address == nil {
		cCtx.XLog.Error("Xpoa::NewXpoaConsensus::CryptoClient in context is nil")
		return nil
	}
	if cCtx.Ledger == nil {
		cCtx.XLog.Error("Xpoa::NewXpoaConsensus::Ledger in context is nil")
		return nil
	}
	if cCfg.ConsensusName != "xpoa" {
		cCtx.XLog.Error("Xpoa::NewXpoaConsensus::consensus name in config is wrong", "name", cCfg.ConsensusName)
		return nil
	}

	// 创建smr实例过程
	// 解析xpoaconfig
	xconfig := &xpoaConfig{}
	err := json.Unmarshal([]byte(cCfg.Config), xconfig)
	if err != nil {
		cCtx.XLog.Error("Xpoa::NewXpoaConsensus::xpoa struct unmarshal error", "error", err)
		return nil
	}
	// create xpoaSchedule
	schedule := &xpoaSchedule{
		address:  cCtx.Network.PeerInfo().Account,
		period:   xconfig.Period,
		blockNum: xconfig.BlockNum,
		ledger:   cCtx.Ledger,
		log:      cCtx.XLog,
	}
	if xconfig.EnableBFT != nil {
		schedule.enableBFT = true
	}
	// xpoaSchedule 实现了ProposerElectionInterface接口，接口定义了validators操作
	// 重启时需要使用最新的validator数据，而不是initValidators数据
	var validators []string
	for _, v := range xconfig.InitProposer {
		validators = append(validators, v.Address)
	}
	schedule.initValidators = validators
	reader, _ := schedule.ledger.GetTipXMSnapshotReader()
	res, err := reader.Get(contractBucket, []byte(validateKeys))
	if snapshotValidators, _ := loadValidatorsMultiInfo(res); snapshotValidators != nil {
		validators = snapshotValidators
	}
	schedule.validators = validators
	// 创建status实例
	status := &XpoaStatus{
		Version:     xconfig.Version,
		StartHeight: cCfg.StartHeight,
		Index:       cCfg.Index,
		election:    schedule,
	}

	// create xpoaConsensus实例
	xpoa := &xpoaConsensus{
		bctx:          &cCtx.BaseCtx,
		election:      schedule,
		isProduce:     make(map[int64]bool),
		config:        xconfig,
		initTimestamp: time.Now().UnixNano(),
		status:        status,
	}
	if schedule.enableBFT {
		// create smr/ chained-bft实例, 需要新建CBFTCrypto、pacemaker和saftyrules实例
		cryptoClient := cCrypto.NewCBFTCrypto(cCtx.Address, cCtx.Crypto)
		qcTree := common.InitQCTree(cCfg.StartHeight, cCtx.Ledger, cCtx.XLog)
		if qcTree == nil {
			cCtx.XLog.Error("Xpoa::NewXpoaConsensus::init QCTree err", "startHeight", cCfg.StartHeight)
			return nil
		}
		pacemaker := &chainedBft.DefaultPaceMaker{
			StartView: cCfg.StartHeight,
		}
		saftyrules := &chainedBft.DefaultSaftyRules{
			Crypto: cryptoClient,
			QcTree: qcTree,
			Log:    cCtx.XLog,
		}
		// 重启状态下需重做tipBlock，此时需重装载justify签名
		var justifySigns []*chainedBftPb.QuorumCertSign
		if !bytes.Equal(qcTree.Genesis.In.GetProposalId(), qcTree.GetRootQC().In.GetProposalId()) {
			justifySigns = xpoa.GetJustifySigns(cCtx.Ledger.GetTipBlock())
		}
		smr := chainedBft.NewSmr(cCtx.BcName, schedule.address, cCtx.XLog, cCtx.Network, cryptoClient, pacemaker, saftyrules, schedule, qcTree, justifySigns)
		go smr.Start()
		xpoa.smr = smr
		cCtx.XLog.Debug("Xpoa::NewXpoaConsensus::load chained-bft successfully.")
	}
	// 注册合约方法
	cCtx.Contract.GetKernRegistry().RegisterKernMethod(contractBucket, contractEditValidate, xpoa.methodEditValidates)
	cCtx.Contract.GetKernRegistry().RegisterKernMethod(contractBucket, contractGetValidates, xpoa.methodGetValidates)
	cCtx.XLog.Debug("Xpoa::NewXpoaConsensus::create a xpoa instance successfully!", "xpoa", xpoa)
	return xpoa
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
	if !ok || v == false {
		x.isProduce[key] = true
	} else {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		// 定期清理isProduce
		cleanProduceMap(x.isProduce, x.election.period)
		goto Again
	}

	// update validates
	if x.election.UpdateValidator(height) {
		x.bctx.GetLog().Debug("Xpoa::CompeteMaster::change validators", "valisators", x.election.validators)
	}
	leader := x.election.GetLocalLeader(time.Now().UnixNano(), height)
	if leader == x.election.address {
		x.bctx.GetLog().Debug("Xpoa::CompeteMaster", "isMiner", true, "height", height)
		// TODO: 首次切换为矿工时SyncBlcok, Bug: 可能会导致第一次出块失败
		needSync := x.election.ledger.GetTipBlock().GetHeight() == 0 || string(x.election.ledger.GetTipBlock().GetProposer()) != leader
		return true, needSync, nil
	}
	x.bctx.GetLog().Debug("Xpoa::CompeteMaster", "isMiner", false, "height", height)
	return false, false, nil
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
func (x *xpoaConsensus) CalculateBlock(block cctx.BlockInterface) error {
	return nil
}

// CheckMinerMatch 查看block是否合法
// ATTENTION: TODO: 上层需要先检查VerifyBlock(block)
func (x *xpoaConsensus) CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error) {
	// 验证矿工身份
	proposer := x.election.GetLocalLeader(block.GetTimestamp(), block.GetHeight())
	if proposer != string(block.GetProposer()) {
		ctx.GetLog().Warn("Xpoa::CheckMinerMatch::calculate proposer error", "logid", ctx.GetLog().GetLogId(), "want", proposer,
			"have", string(block.GetProposer()), "blockId", utils.F(block.GetBlockid()))
		return false, MinerSelectErr
	}
	if !x.election.enableBFT {
		return true, nil
	}
	// 获取block中共识专有存储, 检查justify是否符合要求
	justifyBytes, err := block.GetConsensusStorage()
	if err != nil && block.GetHeight() != x.status.StartHeight {
		ctx.GetLog().Warn("Xpoa::CheckMinerMatch::justify bytes nil", "logid", ctx.GetLog().GetLogId(), "blockId", utils.F(block.GetBlockid()))
		return false, err
	}
	if block.GetHeight() == x.status.StartHeight {
		return true, nil
	}
	// 兼容老的结构
	justify, err := common.OldQCToNew(justifyBytes)
	if err != nil {
		ctx.GetLog().Warn("Xpoa::CheckMinerMatch::OldQCToNew error.", "logid", ctx.GetLog().GetLogId(), "err", err,
			"blockId", utils.F(block.GetBlockid()))
		return false, err
	}
	pNode := x.smr.BlockToProposalNode(block)
	if pNode == nil {
		ctx.GetLog().Warn("Xpoa::CheckMinerMatch::BlockToProposalNode error.", "logid", ctx.GetLog().GetLogId(),
			"blockId", utils.F(block.GetBlockid()), "preBlockId", utils.F(block.GetPreHash()))
		return false, err
	}
	err = x.smr.GetSaftyRules().CheckProposal(pNode.In, justify, x.election.GetValidators(justify.GetProposalView()))
	if err != nil {
		ctx.GetLog().Warn("Xpoa::CheckMinerMatch::bft IsQuorumCertValidate failed", "logid", ctx.GetLog().GetLogId(),
			"proposalQC:[height]", pNode.In.GetProposalView(), "proposalQC:[id]", utils.F(pNode.In.GetProposalId()),
			"justifyQC:[height]", justify.GetProposalView(), "justifyQC:[id]", utils.F(justify.GetProposalId()), "error", err)
		return false, err
	}
	return true, nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回truncate目标(如需裁剪), 返回写consensusStorage, 返回err
func (x *xpoaConsensus) ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error) {
	// 再次检查目前是否是矿工，TODO: check是否有必要，因为和sync抢一把锁，按道理不会有这个问题
	_, pos, _ := x.election.minerScheduling(timestamp, len(x.election.validators))
	if x.election.validators[pos] != x.election.address {
		x.bctx.GetLog().Warn("Xpoa::ProcessBeforeMiner::timeout", "now", x.election.validators[pos])
		return nil, nil, MinerSelectErr
	}
	if !x.election.enableBFT {
		return nil, nil, nil
	}
	// 即本地smr的HightQC和账本TipId不相等，tipId尚未收集到足够签名，回滚到本地HighQC，重做区块
	var truncateT []byte
	var err error
	if !bytes.Equal(x.smr.GetHighQC().GetProposalId(), x.election.ledger.GetTipBlock().GetBlockid()) {
		// 单个节点不存在投票验证的hotstuff流程，因此返回true
		if len(x.election.validators) == 1 {
			return nil, nil, nil
		}
		truncateT, err = func() ([]byte, error) {
			// 1. 比对HighQC与ledger高度
			b, err := x.election.ledger.QueryBlock(x.smr.GetHighQC().GetProposalId())
			if err != nil || b.GetHeight() > x.election.ledger.GetTipBlock().GetHeight() {
				// 不存在时需要把本地HighQC回滚到ledger; HighQC高度高于账本高度，本地HighQC回滚到ledger
				if err := x.smr.EnforceUpdateHighQC(x.election.ledger.GetTipBlock().GetBlockid()); err != nil {
					// 本地HighQC回滚错误直接退出
					return nil, err
				}
				return nil, nil
			}
			// 高度相等时，应统一回滚到上一高度，此时genericQC一定存在
			if b.GetHeight() == x.election.ledger.GetTipBlock().GetHeight() {
				if err := x.smr.EnforceUpdateHighQC(x.smr.GetGenericQC().GetProposalId()); err != nil {
					// 本地HighQC回滚错误直接退出
					return nil, err
				}
				// 此时HighQC已经变化
				return x.smr.GetHighQC().GetProposalId(), nil
			}
			// 2. 账本高度更高时，裁剪账本
			return x.smr.GetHighQC().GetProposalId(), nil
		}()
		if err != nil {
			return nil, nil, err
		}
	}
	// 此处需要获取带签名的完整Justify, 此时HighQC已经更新
	qc := x.smr.GetCompleteHighQC()
	qcQuorumCert, ok := qc.(*chainedBft.QuorumCert)
	if !ok {
		x.bctx.GetLog().Warn("Xpoa::ProcessBeforeMiner::qc transfer err", "qc", qc)
		return nil, nil, InvalidQC
	}
	oldQC, err := common.NewToOldQC(qcQuorumCert)
	if err != nil {
		x.bctx.GetLog().Warn("Xpoa::ProcessBeforeMiner::NewToOldQC error", "error", err)
		return nil, nil, err
	}
	bytes, _ := json.Marshal(map[string]interface{}{"justify": oldQC})
	if truncateT != nil {
		x.bctx.GetLog().Debug("Xpoa::ProcessBeforeMiner::last block not confirmed, walk to previous block", "target", utils.F(truncateT),
			"ledger", x.election.ledger.GetTipBlock().GetHeight(), "HighQC", x.smr.GetHighQC().GetProposalView())
	}
	return truncateT, bytes, nil
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
		x.bctx.GetLog().Warn("Xpoa::CheckMinerMatch::parse storage error", "err", err, "blockId", utils.F(block.GetBlockid()))
		return err
	}
	if justifyBytes != nil && block.GetHeight() > x.status.StartHeight {
		// 若存在升级前后的两个共识都使用了chained-bft组件，在初始时仍不考虑上次共识的历史值
		justify, err := common.OldQCToNew(justifyBytes)
		if err != nil {
			x.bctx.GetLog().Warn("Xpoa::ProcessConfirmBlock::OldQCToNew error", "err", err, "blockId", utils.F(block.GetBlockid()))
			return err
		}
		x.smr.UpdateJustifyQcStatus(justify)
	}
	// 查看本地是否是最新round的生产者
	_, pos, _ := x.election.minerScheduling(block.GetTimestamp(), len(x.election.validators))
	// 如果是当前矿工，则发送Proposal消息
	if x.election.validators[pos] == x.election.address && string(block.GetProposer()) == x.election.address {
		validators := x.election.GetValidators(block.GetHeight())
		if err := x.smr.ProcessProposal(block.GetHeight(), block.GetBlockid(), validators); err != nil {
			x.bctx.GetLog().Warn("Xpoa::ProcessConfirmBlock::bft next proposal failed", "error", err, "blockId", utils.F(block.GetBlockid()))
			return err
		}
		x.bctx.GetLog().Debug("Xpoa::ProcessConfirmBlock::miner confirm finish", "ledger:[height]", x.election.ledger.GetTipBlock().GetHeight(), "viewNum", x.smr.GetCurrentView(), "blockId", utils.F(block.GetBlockid()))
	}
	// 在不在候选人节点中，都直接调用smr生成新的qc树，矿工调用避免了proposal消息后于vote消息
	pNode := x.smr.BlockToProposalNode(block)
	err = x.smr.UpdateQcStatus(pNode)
	x.bctx.GetLog().Debug("Xpoa::ProcessConfirmBlock::Now HighQC", "highQC", utils.F(x.smr.GetHighQC().GetProposalId()), "err", err, "blockId", utils.F(block.GetBlockid()))
	return nil
}

// 共识实例的挂起逻辑, 另: 若共识实例发现绑定block结构有误，会直接停掉当前共识实例并panic
func (x *xpoaConsensus) Stop() error {
	if x.election.enableBFT {
		x.smr.Stop()
	}
	return nil
}

// 共识实例的启动逻辑
func (x *xpoaConsensus) Start() error {
	if x.election.enableBFT {
		x.smr.Start()
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

func (x *xpoaConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	return x.status, nil
}

func (x *xpoaConsensus) GetJustifySigns(block cctx.BlockInterface) []*chainedBftPb.QuorumCertSign {
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil
	}
	signs := common.OldSignToNew(b)
	x.bctx.GetLog().Debug("Xpoa::GetJustifySigns", "signs", signs)
	return signs
}
