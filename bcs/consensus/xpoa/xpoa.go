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
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

func init() {
	consensus.Register("xpoa", NewXpoaConsensus)
}

type xpoaConsensus struct {
	xcontext.XContext
	bcName        string
	election      *xpoaSchedule
	smr           *chainedBft.Smr
	isProduce     map[int64]bool
	config        *xpoaConfig
	initTimestamp int64
	status        *XpoaStatus

	log logs.Logger
}

// NewXpoaConsensus 初始化实例
func NewXpoaConsensus(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) base.ConsensusImplInterface {
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

	// create xpoaSchedule
	schedule := NewXpoaSchedule(xconfig, cCtx, cCfg.StartHeight)
	if schedule == nil {
		cCtx.XLog.Error("consensus:xpoa:NewXpoaSchedule error")
		return nil
	}
	// 创建status实例
	status := &XpoaStatus{
		Name:        "poa",
		Version:     xconfig.Version,
		StartHeight: cCfg.StartHeight,
		Index:       cCfg.Index,
		election:    schedule,
	}
	if schedule.enableBFT {
		status.Name = "xpoa"
	}
	// create xpoaConsensus实例
	xpoa := &xpoaConsensus{
		bcName:        cCtx.BcName,
		XContext:      &cCtx.BaseCtx,
		election:      schedule,
		isProduce:     make(map[int64]bool),
		config:        xconfig,
		initTimestamp: time.Now().UnixNano(),
		status:        status,
		log:           cCtx.XLog,
	}
	// 注册合约方法
	xpoaKMethods := map[string]contract.KernMethod{
		contractEditValidate: xpoa.methodEditValidates,
		contractGetValidates: xpoa.methodGetValidates,
	}
	for method, f := range xpoaKMethods {
		if _, err := cCtx.Contract.GetKernRegistry().GetKernMethod(schedule.bindContractBucket, method); err != nil {
			cCtx.Contract.GetKernRegistry().RegisterKernMethod(schedule.bindContractBucket, method, f)
		}
	}

	// 凡属于共识升级的逻辑，新建的Xpoa实例将直接将当前值置为true，原因是上一共识模块已经在当前值生成了高度为trigger height的区块，新的实例会再生成一边
	timeKey := time.Now().Sub(time.Unix(0, 0)).Milliseconds() / xpoa.config.Period
	xpoa.isProduce[timeKey] = true

	if !schedule.enableBFT {
		cCtx.XLog.Debug("consensus:xpoa:NewXpoaConsensus: create a poa instance successfully!")
		return xpoa
	}

	// create smr/ chained-bft实例, 需要新建CBFTCrypto、pacemaker和saftyrules实例
	cryptoClient := cCrypto.NewCBFTCrypto(cCtx.Address, cCtx.Crypto)
	qcTree := common.InitQCTree(cCfg.StartHeight, cCtx.Ledger, cCtx.XLog)
	if qcTree == nil {
		cCtx.XLog.Error("consensus:xpoa:NewXpoaConsensus: init QCTree err", "startHeight", cCfg.StartHeight)
		return nil
	}
	pacemaker := &chainedBft.DefaultPaceMaker{
		CurrentView: cCfg.StartHeight,
	}
	// 重启状态检查1，pacemaker需要重置
	tipHeight := cCtx.Ledger.GetTipBlock().GetHeight()
	if !bytes.Equal(qcTree.Genesis.In.GetProposalId(), qcTree.GetRootQC().In.GetProposalId()) {
		pacemaker.CurrentView = tipHeight - 1
	}
	saftyrules := &chainedBft.DefaultSaftyRules{
		Crypto: cryptoClient,
		QcTree: qcTree,
		Log:    cCtx.XLog,
	}
	smr := chainedBft.NewSmr(cCtx.BcName, schedule.address, cCtx.XLog, cCtx.Network, cryptoClient, pacemaker, saftyrules, schedule, qcTree)
	// 重启状态检查2，重做tipBlock，此时需重装载justify签名
	if !bytes.Equal(qcTree.Genesis.In.GetProposalId(), qcTree.GetRootQC().In.GetProposalId()) {
		for i := int64(0); i < 3; i++ {
			b, err := cCtx.Ledger.QueryBlockByHeight(tipHeight - i)
			if err != nil {
				break
			}
			smr.LoadVotes(b.GetPreHash(), xpoa.GetJustifySigns(b))
		}
	}
	go smr.Start()
	xpoa.smr = smr
	cCtx.XLog.Debug("consensus:xpoa:NewXpoaConsensus: load chained-bft successfully.")
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
		x.GetLog().Debug("consensus:xpoa:CompeteMaster: change validators", "valisators", x.election.validators)
	}
	_, pos, blockPos := x.election.minerScheduling(time.Now().UnixNano(), len(x.election.validators))
	if blockPos > x.election.blockNum || pos >= int64(len(x.election.validators)) {
		x.GetLog().Debug("consensus:xpoa:CompeteMaster: minerScheduling err", "pos", pos, "blockPos", blockPos)
		goto Again
	}
	x.election.miner = x.election.validators[pos]
	if x.election.miner == x.election.address {
		x.GetLog().Debug("consensus:xpoa:CompeteMaster", "isMiner", true, "height", tipBlock.GetHeight())
		needSync := tipBlock.GetHeight() == 0 || string(tipBlock.GetProposer()) != x.election.miner
		return true, needSync, nil
	}
	x.GetLog().Debug("consensus:xpoa:CompeteMaster", "isMiner", false, "height", tipBlock.GetHeight())
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
	pNode := x.smr.BlockToProposalNode(block)
	preBlock, _ := x.election.ledger.QueryBlock(block.GetPreHash())
	preConStoreBytes, _ := preBlock.GetConsensusStorage()
	err = x.smr.GetSaftyRules().CheckProposal(pNode.In, justify,
		x.election.GetLocalValidates(preBlock.GetTimestamp(), justify.GetProposalView(), preConStoreBytes))
	if err != nil {
		ctx.GetLog().Error("consensus:xpoa:CheckMinerMatch: bft IsQuorumCertValidate failed", "logid", ctx.GetLog().GetLogId(),
			"proposalQC:[height]", pNode.In.GetProposalView(), "proposalQC:[id]", utils.F(pNode.In.GetProposalId()),
			"justifyQC:[height]", justify.GetProposalView(), "justifyQC:[id]", utils.F(justify.GetProposalId()), "error", err)
		return false, err
	}
	return true, nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回truncate目标(如需裁剪), 返回写consensusStorage, 返回err
func (x *xpoaConsensus) ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error) {
	if !x.election.enableBFT {
		return nil, nil, nil
	}
	// 即本地smr的HightQC和账本TipId不相等，tipId尚未收集到足够签名，回滚到本地HighQC，重做区块
	tipBlock := x.election.ledger.GetTipBlock()
	shundown, truncateT, err := x.renewQCStatus(tipBlock)
	if err != nil {
		return nil, nil, err
	}
	if shundown {
		return nil, nil, nil
	}
	// 此处需要获取带签名的完整Justify, 此时HighQC已经更新
	qc := x.smr.GetCompleteHighQC()
	qcQuorumCert, _ := qc.(*chainedBft.QuorumCert)
	oldQC, _ := common.NewToOldQC(qcQuorumCert)
	storage := common.ConsensusStorage{
		Justify: oldQC,
	}
	// 重做时还需要装载标定节点TipHeight，复用TargetBits作为回滚记录，便于追块时获取准确快照高度
	if truncateT != nil {
		x.GetLog().Warn("consensus:xpoa:ProcessBeforeMiner: last block not confirmed, walk to previous block", "target", utils.F(truncateT),
			"ledger", tipBlock.GetHeight(), "HighQC", x.smr.GetHighQC().GetProposalView())
		storage.TargetBits = int32(tipBlock.GetHeight())
	}
	bytes, _ := json.Marshal(storage)
	return truncateT, bytes, nil
}

// renewQCStatus 返回一个裁剪目标，供miner模块直接回滚并出块
func (x *xpoaConsensus) renewQCStatus(tipBlock ledger.BlockHandle) (bool, []byte, error) {
	if bytes.Equal(x.smr.GetHighQC().GetProposalId(), tipBlock.GetBlockid()) {
		return false, nil, nil
	}
	// 单个节点不存在投票验证的hotstuff流程，因此返回true
	if len(x.election.validators) == 1 {
		return true, nil, nil
	}
	// 在本地状态树上找到指代TipBlock的QC，若找不到，则在状态树上找和TipBlock同一分支上的最近值
	targetHighQC, err := func() (chainedBft.QuorumCertInterface, error) {
		targetId := tipBlock.GetBlockid()
		for {
			block, err := x.election.ledger.QueryBlock(targetId)
			if err != nil {
				return nil, err
			}
			// 至多回滚到root节点
			if block.GetHeight() <= x.smr.GetRootQC().GetProposalView() {
				x.log.Warn("consensus:xpoa:renewQCStatus: set root qc.", "root", utils.F(x.smr.GetRootQC().GetProposalId()), "root height", x.smr.GetRootQC().GetProposalView(),
					"block", utils.F(block.GetBlockid()), "block height", block.GetHeight())
				return x.smr.GetRootQC(), nil
			}
			// 查找目标Id是否挂在状态树上，若否，则从target网上查找知道状态树里有
			node := x.smr.QueryNode(block.GetBlockid())
			if node == nil {
				targetId = block.GetPreHash()
				continue
			}
			// node在状态树上找到之后，以此为起点(包括当前点)，继续向上查找，知道找到符合全名数量要求的QC，该QC可强制转化为新的HighQC
			storage, _ := block.GetConsensusStorage()
			wantProposers := x.election.GetLocalValidates(block.GetTimestamp(), block.GetHeight(), storage)
			if wantProposers == nil {
				x.log.Error("consensus:xpoa:renewQCStatus: election error.", "error", err)
				return nil, EmptyValidors
			}
			if !x.smr.ValidNewHighQC(node.In.GetProposalId(), wantProposers) {
				x.log.Warn("consensus:xpoa:renewQCStatus: target not ready", "target", utils.F(node.In.GetProposalId()), "wantProposers", wantProposers, "height", node.In.GetProposalView())
				targetId = block.GetPreHash()
				continue
			}
			return node.In, nil
		}
	}()
	if err != nil {
		return false, nil, err
	}
	ok, err := x.smr.EnforceUpdateHighQC(targetHighQC.GetProposalId())
	if err != nil {
		x.log.Error("consensus:xpoa:renewQCStatus: EnforceUpdateHighQC error.", "error", err)
		return false, nil, err
	}
	if ok {
		x.log.Debug("consensus:xpoa:renewQCStatus: EnforceUpdateHighQC success.", "target", utils.F(targetHighQC.GetProposalId()), "height", targetHighQC.GetProposalView())
	}
	if bytes.Equal(tipBlock.GetBlockid(), targetHighQC.GetProposalId()) {
		return false, nil, nil
	}
	return false, targetHighQC.GetProposalId(), nil
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
		x.GetLog().Warn("consensus:xpoa:CheckMinerMatch: parse storage error", "err", err, "blockId", utils.F(block.GetBlockid()))
		return err
	}
	if justifyBytes != nil && block.GetHeight() > x.status.StartHeight {
		justify, err := common.OldQCToNew(justifyBytes)
		if err != nil {
			x.GetLog().Error("consensus:xpoa:ProcessConfirmBlock: OldQCToNew error", "err", err, "blockId", utils.F(block.GetBlockid()))
			return err
		}
		x.smr.UpdateJustifyQcStatus(justify)
	}
	// 查看本地是否是最新round的生产者
	_, pos, blockPos := x.election.minerScheduling(block.GetTimestamp(), len(x.election.validators))
	if blockPos > x.election.blockNum || pos >= int64(len(x.election.validators)) {
		x.GetLog().Debug("consensus:xpoa:smr::ProcessConfirmBlock: minerScheduling overflow.")
		return scheduleErr
	}
	// 如果是当前矿工，则发送Proposal消息
	if x.election.validators[pos] == x.election.address && string(block.GetProposer()) == x.election.address {
		validators := x.election.GetValidators(block.GetHeight() + 1)
		if err := x.smr.ProcessProposal(block.GetHeight(), block.GetBlockid(), block.GetPreHash(), validators); err != nil {
			x.GetLog().Warn("consensus:xpoa:ProcessConfirmBlock: bft next proposal failed", "error", err, "blockId", utils.F(block.GetBlockid()))
			return err
		}
		x.GetLog().Debug("consensus:xpoa:ProcessConfirmBlock: miner confirm finish", "ledger:[height]", x.election.ledger.GetTipBlock().GetHeight(), "viewNum", x.smr.GetCurrentView(), "blockId", utils.F(block.GetBlockid()))
	}
	// 在不在候选人节点中，都直接调用smr生成新的qc树，矿工调用避免了proposal消息后于vote消息
	pNode := x.smr.BlockToProposalNode(block)
	err = x.smr.UpdateQcStatus(pNode)
	x.GetLog().Debug("consensus:xpoa:ProcessConfirmBlock: Now HighQC", "highQC", utils.F(x.smr.GetHighQC().GetProposalId()), "err", err, "blockId", utils.F(block.GetBlockid()))
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
	return signs
}
