package tdpos

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
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/lib/utils"

	"github.com/xuperchain/xupercore/kernel/consensus/def"
	"github.com/xuperchain/xupercore/lib/logs"
)

func init() {
	consensus.Register("tdpos", NewTdposConsensus)
}

type tdposConsensus struct {
	config    *tdposConfig
	isProduce map[int64]bool
	election  *tdposSchedule
	status    *TdposStatus
	smr       *chainedBft.Smr
	log       logs.Logger
}

func NewTdposConsensus(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) base.ConsensusImplInterface {
	// 解析config中需要的字段
	if cCtx.XLog == nil {
		return nil
	}
	if cCtx.Crypto == nil || cCtx.Address == nil {
		cCtx.XLog.Error("Tdpos::NewTdposConsensus::CryptoClient in context is nil")
		return nil
	}
	if cCtx.Ledger == nil {
		cCtx.XLog.Error("Tdpos::NewTdposConsensus::Ledger in context is nil")
		return nil
	}
	if cCfg.ConsensusName != "tdpos" {
		cCtx.XLog.Error("Tdpos::NewTdposConsensus::consensus name in config is wrong", "name", cCfg.ConsensusName)
		return nil
	}
	xconfig, err := unmarshalTdposConfig([]byte(cCfg.Config))
	if err != nil {
		cCtx.XLog.Error("Tdpos::NewTdposConsensus::tdpos struct unmarshal error", "error", err)
		return nil
	}
	// 新建schedule实例，该实例包含smr中election的接口实现
	schedule := NewSchedule(xconfig, cCtx.XLog, cCtx.Ledger, cCfg.StartHeight)
	if schedule == nil {
		cCtx.XLog.Error("Tdpos::NewTdposConsensus::new schedule err.")
		return nil
	}
	schedule.address = cCtx.Network.PeerInfo().Account

	status := &TdposStatus{
		Version:     xconfig.Version,
		StartHeight: cCfg.StartHeight,
		Index:       cCfg.Index,
		election:    schedule,
		Name:        "tdpos",
	}
	if schedule.enableChainedBFT {
		status.Name = "xpos"
	}
	tdpos := &tdposConsensus{
		config:    xconfig,
		isProduce: make(map[int64]bool),
		election:  schedule,
		status:    status,
		log:       cCtx.XLog,
	}
	// 注册合约方法
	tdposKMethods := map[string]contract.KernMethod{
		contractNominateCandidate: tdpos.runNominateCandidate,
		contractRevokeCandidate:   tdpos.runRevokeCandidate,
		contractVote:              tdpos.runVote,
		contractRevokeVote:        tdpos.runRevokeVote,
	}
	for method, f := range tdposKMethods {
		if _, err := cCtx.Contract.GetKernRegistry().GetKernMethod(schedule.bindContractBucket, method); err != nil {
			cCtx.Contract.GetKernRegistry().RegisterKernMethod(schedule.bindContractBucket, method, f)
		}
	}

	// 凡属于共识升级的逻辑，新建的Tdpos实例将直接将当前值置为true，原因是上一共识模块已经在当前值生成了高度为trigger height的区块，新的实例会再生成一边
	timeKey := time.Now().Sub(time.Unix(0, 0)).Milliseconds() / tdpos.config.Period
	tdpos.isProduce[timeKey] = true
	if !schedule.enableChainedBFT {
		cCtx.XLog.Debug("Tdpos::NewTdposConsensus::create a tdpos instance successfully.")
		return tdpos
	}

	// create smr/ chained-bft实例, 需要新建CBFTCrypto、pacemaker和saftyrules实例
	cryptoClient := cCrypto.NewCBFTCrypto(cCtx.Address, cCtx.Crypto)
	qcTree := common.InitQCTree(cCfg.StartHeight, cCtx.Ledger, cCtx.XLog)
	if qcTree == nil {
		cCtx.XLog.Error("Tdpos::NewTdposConsensus::init QCTree err", "startHeight", cCfg.StartHeight)
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
			smr.LoadVotes(b.GetPreHash(), tdpos.GetJustifySigns(b))
		}
	}
	go smr.Start()
	tdpos.smr = smr
	cCtx.XLog.Debug("Tdpos::NewTdposConsensus::load chained-bft successfully.")
	return tdpos
}

// CompeteMaster is the specific implementation of ConsensusInterface
func (tp *tdposConsensus) CompeteMaster(height int64) (bool, bool, error) {
Again:
	t := time.Now().UnixNano() / int64(time.Millisecond)
	key := t / tp.config.Period
	sleep := tp.config.Period - t%tp.config.Period
	if sleep > MAXSLEEPTIME {
		sleep = MAXSLEEPTIME
	}
	_, ok := tp.isProduce[key]
	if !ok {
		tp.isProduce[key] = true
	} else {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		// 定期清理isProduce
		common.CleanProduceMap(tp.isProduce, tp.config.Period)
		goto Again
	}

	// 查当前时间的term 和 pos
	term, pos, blockPos := tp.election.minerScheduling(time.Now().UnixNano())
	if blockPos < 0 || blockPos >= tp.election.blockNum || pos >= tp.election.proposerNum {
		tp.log.Debug("Tdpos::CompeteMaster::minerScheduling err", "term", term, "pos", pos, "blockPos", blockPos)
		goto Again
	}
	// 即现在有可能发生候选人变更，此时需要拿tipHeight-3=H高度的稳定高度当作快照，故input时的高度一定是TipHeight
	if term > tp.election.curTerm {
		tp.election.UpdateProposers(tp.election.ledger.GetTipBlock().GetHeight())
	}
	// 查当前term 和 pos是否是自己
	tp.election.curTerm = term
	tp.election.miner = tp.election.validators[pos]
	// master check
	if tp.election.validators[pos] == tp.election.address {
		tp.log.Debug("Tdpos::CompeteMaster::now xterm infos", "term", term, "pos", pos, "blockPos", blockPos, "master", true, "height", tp.election.ledger.GetTipBlock().GetHeight())
		s := tp.needSync()
		return true, s, nil
	}
	tp.log.Debug("Tdpos::CompeteMaster::now xterm infos", "term", term, "pos", pos, "blockPos", blockPos, "master", false, "height", tp.election.ledger.GetTipBlock().GetHeight())
	return false, false, nil
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
func (tp *tdposConsensus) CalculateBlock(block cctx.BlockInterface) error {
	return nil
}

// CheckMinerMatch 查看block是否合法
func (tp *tdposConsensus) CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error) {
	// 获取当前共识存储
	bv, err := block.GetConsensusStorage()
	if err != nil {
		tp.log.Warn("Tdpos::CheckMinerMatch::GetConsensusStorage error", "err", err)
		return false, err
	}
	tdposStorage, err := common.ParseOldQCStorage(bv)
	if err != nil {
		tp.log.Warn("Tdpos::CheckMinerMatch::ParseOldQCStorage error.", "err", err)
		return false, err
	}
	tp.log.Debug("Tdpos::CheckMinerMatch", "blockid", utils.F(block.GetBlockid()), "height", block.GetHeight())

	// 1 判断当前区块生产者是否合法
	term, pos, blockPos := tp.election.minerScheduling(block.GetTimestamp())
	if blockPos < 0 || blockPos >= tp.election.blockNum || pos >= tp.election.proposerNum {
		tp.log.Warn("Tdpos::CheckMinerMatch::minerScheduling overflow.")
		return false, scheduleErr
	}
	var wantProposers []string
	storage, _ := block.GetConsensusStorage()
	wantProposers, err = tp.election.CalOldProposers(block.GetHeight(), block.GetTimestamp(), storage)
	if err != nil {
		tp.log.Debug("Tdpos::CheckMinerMatch::CalculateProposers error", "err", err)
		return false, err
	}
	if wantProposers[pos] != string(block.GetProposer()) {
		tp.log.Debug("Tdpos::CheckMinerMatch::invalid proposer", "want", wantProposers[pos], "have", string(block.GetProposer()))
		return false, invalidProposerErr
	}

	// 2 验证轮数信息, 判断curTerm是否合法
	if tdposStorage.CurTerm > 0 && tdposStorage.CurTerm != term {
		tp.log.Warn("Tdpos::CheckMinerMatch::check failed, invalid term.", "want", term, "have", tdposStorage.CurTerm)
		return false, invalidTermErr
	}
	if !tp.election.enableChainedBFT {
		return true, nil
	}
	// 验证BFT时，需要除开初始化后的第一个block验证，此时没有justify值
	if block.GetHeight() <= tp.status.StartHeight {
		return true, nil
	}
	// 兼容老的结构
	justify, err := common.OldQCToNew(bv)
	if err != nil {
		tp.log.Warn("Tdpos::CheckMinerMatch::OldQCToNew error.", "logid", ctx.GetLog().GetLogId(), "err", err, "blockId", utils.F(block.GetBlockid()))
		return false, err
	}
	pNode := tp.smr.BlockToProposalNode(block)
	preBlock, _ := tp.election.ledger.QueryBlock(block.GetPreHash())
	prestorage, _ := preBlock.GetConsensusStorage()
	validators, err := tp.election.CalOldProposers(preBlock.GetHeight(), preBlock.GetTimestamp(), prestorage)
	if err != nil {
		return false, err
	}
	err = tp.smr.GetSaftyRules().CheckProposal(pNode.In, justify, validators)
	if err != nil {
		tp.log.Debug("Tdpos::CheckMinerMatch::bft IsQuorumCertValidate failed", "proposalQC:[height]", pNode.In.GetProposalView(),
			"proposalQC:[id]", utils.F(pNode.In.GetProposalId()), "justifyQC:[height]", justify.GetProposalView(),
			"justifyQC:[id]", utils.F(justify.GetProposalId()), "error", err)
		return false, err
	}
	return true, nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回是否需要truncate, 返回写consensusStorage, 返回err
func (tp *tdposConsensus) ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error) {
	term, pos, blockPos := tp.election.minerScheduling(timestamp)
	if blockPos < 0 || term != tp.election.curTerm || blockPos >= tp.election.blockNum || pos >= tp.election.proposerNum {
		tp.log.Warn("Tdpos::ProcessBeforeMiner::timeoutBlockErr", "term", term, "tp.election.curTerm", tp.election.curTerm,
			"blockPos", blockPos, "tp.election.blockNum", tp.election.blockNum, "pos", pos, "tp.election.proposerNum", tp.election.proposerNum)
		return nil, nil, timeoutBlockErr
	}
	if tp.election.validators[pos] != tp.election.address {
		return nil, nil, timeoutBlockErr
	}

	storage := common.ConsensusStorage{
		CurTerm:     tp.election.curTerm,
		CurBlockNum: blockPos,
	}
	if !tp.election.enableChainedBFT {
		storageBytes, err := json.Marshal(storage)
		if err != nil {
			return nil, nil, err
		}
		return nil, storageBytes, nil
	}

	tipBlock := tp.election.ledger.GetTipBlock()
	// 根据BFT配置判断是否需要加入Chained-BFT相关存储，及变更smr状态
	var truncateT []byte
	var err error
	// 即本地smr的HightQC和账本TipId不相等，tipId尚未收集到足够签名，回滚到本地HighQC，重做区块
	if !bytes.Equal(tp.smr.GetHighQC().GetProposalId(), tipBlock.GetBlockid()) {
		// 单个节点不存在投票验证的hotstuff流程，因此返回true
		if len(tp.election.validators) == 1 {
			return nil, nil, nil
		}
		truncateT, err = func() ([]byte, error) {
			// 1. 比对HighQC与ledger高度
			b, err := tp.election.ledger.QueryBlock(tp.smr.GetHighQC().GetProposalId())
			if err != nil || b.GetHeight() > tipBlock.GetHeight() {
				// 不存在时需要把本地HighQC回滚到ledger; HighQC高度高于账本高度，本地HighQC回滚到ledger
				if err := tp.smr.EnforceUpdateHighQC(tipBlock.GetBlockid()); err != nil {
					// 本地HighQC回滚错误直接退出
					return nil, err
				}
				return nil, nil
			}
			// 高度相等时，应统一回滚到上一高度，此时genericQC一定存在
			if b.GetHeight() == tipBlock.GetHeight() {
				if err := tp.smr.EnforceUpdateHighQC(tp.smr.GetGenericQC().GetProposalId()); err != nil {
					// 本地HighQC回滚错误直接退出
					return nil, err
				}
				return tp.smr.GetHighQC().GetProposalId(), nil
			}
			// 2. 账本高度更高时，裁剪账本
			return tp.smr.GetHighQC().GetProposalId(), nil
		}()
		if err != nil {
			return nil, nil, err
		}
	}
	qc := tp.smr.GetCompleteHighQC()
	qcQuorumCert, ok := qc.(*chainedBft.QuorumCert)
	if !ok {
		tp.log.Error("Tdpos::ProcessBeforeMiner::qc transfer err", "qc", qc)
		return nil, nil, InvalidQC
	}
	oldQC, err := common.NewToOldQC(qcQuorumCert)
	if err != nil {
		tp.log.Error("Tdpos::ProcessBeforeMiner::NewToOldQC error", "error", err)
		return nil, nil, err
	}
	storage.Justify = oldQC
	// 重做时还需要装载标定节点TipHeight，复用TargetBits作为回滚记录，便于追块时获取准确快照高度
	if truncateT != nil {
		tp.log.Debug("smr::ProcessBeforeMiner::last block not confirmed, walk to previous block", "target", utils.F(truncateT),
			"ledger", tipBlock.GetHeight(), "HighQC", tp.smr.GetHighQC().GetProposalView())
		storage.TargetBits = int32(tipBlock.GetHeight())
	}
	storageBytes, err := json.Marshal(storage)
	if err != nil {
		return nil, nil, err
	}
	tp.log.Debug("Tdpos::ProcessBeforeMiner", "res", storage)
	return truncateT, storageBytes, nil
}

// ProcessConfirmBlock 用于确认块后进行相应的处理
func (tp *tdposConsensus) ProcessConfirmBlock(block cctx.BlockInterface) error {
	if !tp.election.enableChainedBFT {
		return nil
	}
	// confirm的第一步：不管是否为当前Leader，都需要更新本地voteQC树，保证当前block的justify votes被写入本地账本
	// 获取block中共识专有存储, 检查justify是否符合要求
	bv, err := block.GetConsensusStorage()
	if err != nil && block.GetHeight() != tp.status.StartHeight {
		tp.log.Warn("Tdpos::CheckMinerMatch::parse storage error", "err", err)
		return err
	}
	if bv != nil && block.GetHeight() > tp.status.StartHeight {
		justify, err := common.OldQCToNew(bv)
		if err != nil {
			tp.log.Error("Tdpos::ProcessConfirmBlock::OldQCToNew error", "err", err, "blockId", utils.F(block.GetBlockid()))
			return err
		}
		tp.smr.UpdateJustifyQcStatus(justify)
	}
	// 查看本地是否是最新round的生产者
	_, pos, blockPos := tp.election.minerScheduling(block.GetTimestamp())
	if blockPos < 0 || blockPos >= tp.election.blockNum || pos >= tp.election.proposerNum {
		tp.log.Debug("Tdpos::smr::ProcessConfirmBlock::minerScheduling overflow.")
		return scheduleErr
	}
	if tp.election.validators[pos] == tp.election.address && string(block.GetProposer()) == tp.election.address {
		// 如果是当前矿工，检测到下一轮需变更validates，且下一轮proposer并不在节点列表中，此时需在广播列表中新加入节点
		validators := tp.election.GetValidators(block.GetHeight() + 1)
		if err := tp.smr.ProcessProposal(block.GetHeight(), block.GetBlockid(), validators); err != nil {
			tp.log.Warn("Tdpos::smr::ProcessConfirmBlock::bft next proposal failed", "error", err)
			return err
		}
		tp.log.Debug("Tdpos::smr::ProcessConfirmBlock::miner confirm finish", "ledger:[height]", tp.election.ledger.GetTipBlock().GetHeight(), "viewNum", tp.smr.GetCurrentView())
	}
	// 在不在候选人节点中，都直接调用smr生成新的qc树，矿工调用避免了proposal消息后于vote消息
	pNode := tp.smr.BlockToProposalNode(block)
	err = tp.smr.UpdateQcStatus(pNode)
	tp.log.Debug("Tdpos::ProcessConfirmBlock::Now HighQC", "highQC", utils.F(tp.smr.GetHighQC().GetProposalId()), "err", err, "blockId", utils.F(block.GetBlockid()))
	return nil
}

// 共识实例的挂起逻辑, 另: 若共识实例发现绑定block结构有误，会直接停掉当前共识实例并panic
func (tp *tdposConsensus) Stop() error {
	if tp.election.enableChainedBFT {
		tp.smr.Stop()
	}
	return nil
}

// 共识实例的启动逻辑
func (tp *tdposConsensus) Start() error {
	if tp.election.enableChainedBFT {
		tp.smr.Start()
	}
	return nil
}

// 共识占用blockinterface的专有存储，特定共识需要提供parse接口，在此作为接口高亮
func (tp *tdposConsensus) ParseConsensusStorage(block cctx.BlockInterface) (interface{}, error) {
	return ParseConsensusStorage(block)
}

func (tp *tdposConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	return tp.status, nil
}

func (tp *tdposConsensus) GetJustifySigns(block cctx.BlockInterface) []*chainedBftPb.QuorumCertSign {
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil
	}
	signs := common.OldSignToNew(b)
	tp.log.Debug("Tdpos::GetJustifySigns", "signs", signs)
	return signs
}
