package tdpos

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	chainedBftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
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

	// 记录某一轮内某个候选人出块是否大于系统限制, 以此避免矿工恶意出块, 切轮时进行初始化 map[term_num]map[proposer]map[blockid]bool
	curTermProposerProduceNumCache map[int64]map[string]map[string]bool
	mutex                          sync.Mutex
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
	if len((xconfig.InitProposer)["1"]) != len((xconfig.InitProposerNeturl)["1"]) {
		cCtx.XLog.Error("Tdpos::NewTdposConsensus::initProposer should be mapped into initProposerNeturl", "error", InitProposerNeturlErr)
		return nil
	}
	// 新建schedule实例，该实例包含smr中election的接口实现
	schedule := NewSchedule(xconfig, cCtx.XLog, cCtx.Ledger)
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
	}
	tdpos := &tdposConsensus{
		config:                         xconfig,
		isProduce:                      make(map[int64]bool),
		election:                       schedule,
		status:                         status,
		log:                            cCtx.XLog,
		curTermProposerProduceNumCache: make(map[int64]map[string]map[string]bool),
	}
	if schedule.enableChainedBFT {
		// create smr/ chained-bft实例, 需要新建CBFTCrypto、pacemaker和saftyrules实例
		cryptoClient := cCrypto.NewCBFTCrypto(cCtx.Address, cCtx.Crypto)
		qcTree := common.InitQCTree(cCfg.StartHeight, cCtx.Ledger, cCtx.XLog)
		if qcTree == nil {
			cCtx.XLog.Error("Tdpos::NewTdposConsensus::init QCTree err", "startHeight", cCfg.StartHeight)
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
			justifySigns = tdpos.GetJustifySigns(cCtx.Ledger.GetTipBlock())
		}
		smr := chainedBft.NewSmr(cCtx.BcName, schedule.address, cCtx.XLog, cCtx.Network, cryptoClient, pacemaker, saftyrules, schedule, qcTree, justifySigns)
		go smr.Start()
		tdpos.smr = smr
		cCtx.XLog.Debug("Tdpos::NewTdposConsensus::load chained-bft successfully.")
	}
	cCtx.XLog.Debug("Tdpos::NewTdposConsensus::create a tdpos instance successfully.", "tdpos", tdpos)

	// 注册合约方法
	cCtx.Contract.GetKernRegistry().RegisterKernMethod(contractBucket, contractNominateCandidate, tdpos.runNominateCandidate)
	cCtx.Contract.GetKernRegistry().RegisterKernMethod(contractBucket, contractRevokeCandidata, tdpos.runRevokeCandidate)
	cCtx.Contract.GetKernRegistry().RegisterKernMethod(contractBucket, contractVote, tdpos.runVote)
	cCtx.Contract.GetKernRegistry().RegisterKernMethod(contractBucket, contractRevokeVote, tdpos.runRevokeVote)
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
	v, ok := tp.isProduce[key]
	if !ok || v == false {
		tp.isProduce[key] = true
	} else {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		// 定期清理isProduce
		cleanProduceMap(tp.isProduce, tp.config.Period)
		goto Again
	}

	// 查当前时间的term 和 pos
	term, pos, blockPos := tp.election.minerScheduling(time.Now().UnixNano())
	proposerChangedFlag := false
	// 根据term更新当前validators
	if term > tp.election.curTerm {
		proposerChangedFlag = tp.election.UpdateProposers(height) && height > 3
	}
	// 查当前term 和 pos是否是自己
	tp.election.curTerm = term
	if blockPos > tp.election.blockNum || pos >= tp.election.proposerNum {
		tp.log.Warn("Tdpos::CompeteMaster::minerScheduling err", "term", term, "pos", pos, "blockPos", blockPos)
		goto Again
	}
	// 在smr层面更新候选人信息
	if proposerChangedFlag {
		err := tp.election.notifyTermChanged(height)
		if err != nil {
			tp.log.Warn("Tdpos::CompeteMaster::proposer or term change, bft Update Validators failed", "error", err)
		}
	}
	// master check
	if tp.election.proposers[pos] == tp.election.address {
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
// ATTENTION: TODO: 上层需要先检查VerifyBlock(block)
func (tp *tdposConsensus) CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error) {
	// 获取当前共识存储
	bv, err := block.GetConsensusStorage()
	if err != nil {
		tp.log.Warn("Tdpos::CheckMinerMatch::GetConsensusStorage error", "err", err)
		return false, err
	}
	tdposStorage, err := common.ParseOldQCStorage(bv)
	if err != nil {
		tp.log.Error("Tdpos::CheckMinerMatch::ParseOldQCStorage error.", "err", err)
		return false, err
	}
	tp.log.Debug("Tdpos::CheckMinerMatch", "tdposStorage", tdposStorage)

	// 1 判断当前区块生产者是否合法
	term, pos, _ := tp.election.minerScheduling(block.GetTimestamp())
	curHeight := block.GetHeight()
	var wantProposers []string
	wantProposers, err = tp.election.calculateProposers(curHeight)
	if err != nil {
		tp.log.Warn("Tdpos::CheckMinerMatch::calculateProposers error", "err", err)
		return false, err
	}
	if wantProposers[pos] != string(block.GetProposer()) {
		tp.log.Warn("Tdpos::CheckMinerMatch::invalid proposer", "want", wantProposers[pos], "have", block.GetProposer())
		return false, invalidProposerErr
	}

	// 2 验证轮数信息, 判断curTerm是否合法
	if tdposStorage.CurTerm > 0 {
		// 获取上一区块共识存储
		preBlock, err := tp.election.ledger.QueryBlock(block.GetPreHash())
		if err != nil {
			tp.log.Warn("Tdpos::CheckMinerMatch::check failed, get preblock error")
			return false, err
		}
		pv, err := preBlock.GetConsensusStorage()
		if err != nil {
			tp.log.Warn("Tdpos::CheckMinerMatch::parse pre-storage error", "err", err)
			return false, err
		}
		preTdposStorage, err := common.ParseOldQCStorage(pv)
		if err != nil {
			tp.log.Error("Tdpos::CheckMinerMatch::ParseOldQCStorage pre-storage transfer error", "err", err)
			return false, err
		}
		if tdposStorage.CurTerm != term {
			tp.log.Warn("Tdpos::CheckMinerMatch::check failed, invalid term.", "want", term, "have", tdposStorage.CurTerm)
			return false, invalidTermErr
		}
		// 减少矿工50%概率恶意地输入时间
		if preTdposStorage.CurTerm > term {
			tp.log.Warn("Tdpos::CheckMinerMatch::check failed, preBlock.CurTerm is bigger than the new received.",
				"preBlock", preTdposStorage.CurTerm, "have", term)
			return false, invalidTermErr
		}
	}

	// 3 验证bft相关信息, 除开初始化后的第一个block验证
	if tp.election.enableChainedBFT && block.GetHeight() > tp.status.StartHeight {
		// 兼容老的结构
		justify, err := common.OldQCToNew(bv)
		if err != nil {
			tp.log.Warn("Tdpos::CheckMinerMatch::OldQCToNew error.", "logid", ctx.GetLog().GetLogId(), "err", err, "blockId", utils.F(block.GetBlockid()))
			return false, err
		}
		pNode := tp.smr.BlockToProposalNode(block)
		err = tp.smr.GetSaftyRules().CheckProposal(pNode.In, justify, tp.election.GetValidators(block.GetHeight()))
		if err != nil {
			tp.log.Warn("Tdpos::CheckMinerMatch::bft IsQuorumCertValidate failed", "proposalQC:[height]", pNode.In.GetProposalView(),
				"proposalQC:[id]", utils.F(pNode.In.GetProposalId()), "justifyQC:[height]", justify.GetProposalView(),
				"justifyQC:[id]", utils.F(justify.GetProposalId()), "error", err)
			return false, err
		}
	}

	// 4 根据term信息，以及历史信息，判断该矿工是否多生产了区块，这种行为为恶意出块行为
	tp.mutex.Lock()
	defer tp.mutex.Unlock()
	// 判断某个矿工是否恶意出块
	if _, ok := tp.curTermProposerProduceNumCache[tdposStorage.CurTerm]; !ok {
		tp.curTermProposerProduceNumCache[tdposStorage.CurTerm] = make(map[string]map[string]bool)
	}
	if _, ok := tp.curTermProposerProduceNumCache[tdposStorage.CurTerm][string(block.GetProposer())]; !ok {
		tp.curTermProposerProduceNumCache[tdposStorage.CurTerm][string(block.GetProposer())] = make(map[string]bool)
	}
	tp.curTermProposerProduceNumCache[tdposStorage.CurTerm][string(block.GetProposer())][utils.F(block.GetBlockid())] = true
	if int64(len(tp.curTermProposerProduceNumCache[tdposStorage.CurTerm][string(block.GetProposer())])) >= tp.election.blockNum+1 {
		tp.log.Warn("Tdpos::CheckMinerMatch::check failed, proposer produce more than config blockNum.", "blockNum",
			len(tp.curTermProposerProduceNumCache[tdposStorage.CurTerm][string(block.GetProposer())]))
		return false, proposeBlockMoreThanConfigErr
	}

	return true, nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回是否需要truncate, 返回写consensusStorage, 返回err
func (tp *tdposConsensus) ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error) {
	term, pos, blockPos := tp.election.minerScheduling(timestamp)
	if term != tp.election.curTerm || blockPos > tp.election.blockNum || pos >= tp.election.proposerNum {
		return nil, nil, timeoutBlockErr
	}
	if tp.election.proposers[pos] != tp.election.address {
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

	// 根据BFT配置判断是否需要加入Chained-BFT相关存储，及变更smr状态
	var truncateT []byte
	var err error
	// 即本地smr的HightQC和账本TipId不相等，tipId尚未收集到足够签名，回滚到本地HighQC，重做区块
	if !bytes.Equal(tp.smr.GetHighQC().GetProposalId(), tp.election.ledger.GetTipBlock().GetBlockid()) {
		// 单个节点不存在投票验证的hotstuff流程，因此返回true
		if len(tp.election.proposers) == 1 {
			return nil, nil, nil
		}
		truncateT, err = func() ([]byte, error) {
			// 1. 比对HighQC与ledger高度
			b, err := tp.election.ledger.QueryBlock(tp.smr.GetHighQC().GetProposalId())
			if err != nil || b.GetHeight() > tp.election.ledger.GetTipBlock().GetHeight() {
				// 不存在时需要把本地HighQC回滚到ledger; HighQC高度高于账本高度，本地HighQC回滚到ledger
				if err := tp.smr.EnforceUpdateHighQC(tp.election.ledger.GetTipBlock().GetBlockid()); err != nil {
					// 本地HighQC回滚错误直接退出
					return nil, err
				}
				return nil, nil
			}
			// 高度相等时，应统一回滚到上一高度，此时genericQC一定存在
			if b.GetHeight() == tp.election.ledger.GetTipBlock().GetHeight() {
				if err := tp.smr.EnforceUpdateHighQC(tp.smr.GetGenericQC().GetProposalId()); err != nil {
					// 本地HighQC回滚错误直接退出
					return nil, err
				}
				return tp.smr.GetGenericQC().GetProposalId(), nil
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
		tp.log.Warn("Tdpos::ProcessBeforeMiner::qc transfer err", "qc", qc)
		return nil, nil, InvalidQC
	}
	oldQC, err := common.NewToOldQC(qcQuorumCert)
	if err != nil {
		tp.log.Warn("Tdpos::ProcessBeforeMiner::NewToOldQC error", "error", err)
		return nil, nil, err
	}
	storage.Justify = oldQC
	storageBytes, err := json.Marshal(storage)
	if err != nil {
		return nil, nil, err
	}
	tp.log.Debug("Tdpos::ProcessBeforeMiner", "res", storage)
	if truncateT != nil {
		tp.log.Debug("smr::ProcessBeforeMiner::last block not confirmed, walk to previous block", "target", utils.F(truncateT),
			"ledger", tp.election.ledger.GetTipBlock().GetHeight(), "HighQC", tp.smr.GetHighQC().GetProposalView())
	}
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
			tp.log.Warn("Tdpos::ProcessConfirmBlock::OldQCToNew error", "err", err, "blockId", utils.F(block.GetBlockid()))
			return err
		}
		tp.smr.UpdateJustifyQcStatus(justify)
	}
	// 查看本地是否是最新round的生产者
	_, pos, _ := tp.election.minerScheduling(block.GetTimestamp())
	// 如果是当前矿工，检测到下一轮需变更validates，且下一轮proposer并不在节点列表中，此时需在广播列表中新加入节点
	if tp.election.proposers[pos] == tp.election.address && string(block.GetProposer()) == tp.election.address {
		validators := tp.election.GetValidators(block.GetHeight())
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
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil, err
	}
	justify, err := common.ParseOldQCStorage(b)
	if err != nil {
		tp.log.Error("Tdpos::ParseConsensusStorage invalid consensus storage", "err", err)
		return nil, err
	}
	return justify, nil
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
