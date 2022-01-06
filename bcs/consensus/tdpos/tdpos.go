package tdpos

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus"
	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	chainedBftPb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	quorumcert "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/storage"
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
	cCtx      cctx.ConsensusCtx
	bcName    string
	config    *tdposConfig
	isProduce map[int64]bool
	election  *tdposSchedule
	status    *TdposStatus
	smr       *chainedBft.Smr
	contract  contract.Manager
	kMethod   map[string]contract.KernMethod
	log       logs.Logger
}

func NewTdposConsensus(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) consensus.ConsensusImplInterface {
	// 解析config中需要的字段
	if cCtx.XLog == nil {
		return nil
	}
	if cCtx.Crypto == nil || cCtx.Address == nil {
		cCtx.XLog.Error("consensus:tdpos:NewTdposConsensus: CryptoClient in context is nil")
		return nil
	}
	if cCtx.Ledger == nil {
		cCtx.XLog.Error("consensus:tdpos:NewTdposConsensus: Ledger in context is nil")
		return nil
	}
	if cCfg.ConsensusName != "tdpos" {
		cCtx.XLog.Error("consensus:tdpos:NewTdposConsensus: consensus name in config is wrong", "name", cCfg.ConsensusName)
		return nil
	}
	xconfig, err := unmarshalTdposConfig([]byte(cCfg.Config))
	if err != nil {
		cCtx.XLog.Error("consensus:tdpos:NewTdposConsensus: tdpos struct unmarshal error", "error", err)
		return nil
	}
	// 新建schedule实例，该实例包含smr中election的接口实现
	schedule := NewSchedule(xconfig, cCtx.XLog, cCtx.Ledger, cCfg.StartHeight)
	if schedule == nil {
		cCtx.XLog.Error("consensus:tdpos:NewTdposConsensus: new schedule err.")
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
		bcName:    cCtx.BcName,
		config:    xconfig,
		isProduce: make(map[int64]bool),
		election:  schedule,
		status:    status,
		contract:  cCtx.Contract,
		log:       cCtx.XLog,
		cCtx:      cCtx,
	}

	tdposKMethods := map[string]contract.KernMethod{
		contractNominateCandidate: tdpos.runNominateCandidate,
		contractRevokeCandidate:   tdpos.runRevokeCandidate,
		contractVote:              tdpos.runVote,
		contractRevokeVote:        tdpos.runRevokeVote,
		contractGetTdposInfos:     tdpos.runGetTdposInfos,
	}

	tdpos.kMethod = tdposKMethods

	// 凡属于共识升级的逻辑，新建的Tdpos实例将直接将当前值置为true，原因是上一共识模块已经在当前值生成了高度为trigger height的区块，新的实例会再生成一边
	timeKey := time.Now().Sub(time.Unix(0, 0)).Milliseconds() / tdpos.config.Period
	tdpos.isProduce[timeKey] = true
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
		tp.log.Debug("consensus:tdpos:CompeteMaster: minerScheduling err", "term", term, "pos", pos, "blockPos", blockPos)
		goto Again
	}
	// 即现在有可能发生候选人变更，此时需要拿tipHeight-3=H高度的稳定高度当作快照，故input时的高度一定是TipHeight
	if term > tp.election.curTerm {
		tp.election.UpdateProposers(tp.election.ledger.QueryTipBlockHeader().GetHeight())
	}
	// 查当前term 和 pos是否是自己
	tp.election.curTerm = term
	tp.election.miner = tp.election.validators[pos]
	// master check
	if tp.election.validators[pos] == tp.election.address {
		tp.log.Debug("consensus:tdpos:CompeteMaster: now xterm infos", "term", term, "pos", pos, "blockPos", blockPos, "master", true, "height", tp.election.ledger.QueryTipBlockHeader().GetHeight())
		s := tp.needSync()
		return true, s, nil
	}
	tp.log.Debug("consensus:tdpos:CompeteMaster: now xterm infos", "term", term, "pos", pos, "blockPos", blockPos, "master", false, "height", tp.election.ledger.QueryTipBlockHeader().GetHeight())
	return false, false, nil
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要计算结果
func (tp *tdposConsensus) CalculateBlock(block cctx.BlockInterface) error {
	return nil
}

// CheckMinerMatch 查看block是否合法
func (tp *tdposConsensus) CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error) {
	// 获取当前共识存储
	bv, err := block.GetConsensusStorage()
	if err != nil {
		tp.log.Warn("consensus:tdpos:CheckMinerMatch: GetConsensusStorage error", "err", err)
		return false, err
	}
	tp.log.Debug("consensus:tdpos:CheckMinerMatch", "blockid", utils.F(block.GetBlockid()), "height", block.GetHeight())

	// 1 判断当前区块生产者是否合法
	_, pos, blockPos := tp.election.minerScheduling(block.GetTimestamp())
	if blockPos < 0 || blockPos >= tp.election.blockNum || pos >= tp.election.proposerNum {
		tp.log.Warn("consensus:tdpos:CheckMinerMatch: minerScheduling overflow.")
		return false, ErrValueNotFound
	}
	var wantProposers []string
	storage, _ := block.GetConsensusStorage()
	wantProposers, err = tp.election.CalOldProposers(block.GetHeight(), block.GetTimestamp(), storage)
	if err != nil {
		tp.log.Error("consensus:tdpos:CheckMinerMatch: CalculateProposers error", "err", err)
		return false, err
	}
	if wantProposers[pos] != string(block.GetProposer()) {
		tp.log.Error("consensus:tdpos:CheckMinerMatch: invalid proposer", "want", wantProposers[pos], "have", string(block.GetProposer()))
		return false, ErrInvalidProposer
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
		tp.log.Warn("consensus:tdpos:CheckMinerMatch: OldQCToNew error.", "logid", ctx.GetLog().GetLogId(), "err", err, "blockId", utils.F(block.GetBlockid()))
		return false, err
	}
	preBlock, _ := tp.election.ledger.QueryBlockHeader(block.GetPreHash())
	prestorage, _ := preBlock.GetConsensusStorage()
	validators, err := tp.election.CalOldProposers(preBlock.GetHeight(), preBlock.GetTimestamp(), prestorage)
	if err != nil {
		tp.log.Error("consensus:tdpos:CheckMinerMatch: election error", "err", err, "preBlock", utils.F(preBlock.GetBlockid()))
		return false, err
	}

	// 包装成统一入口访问smr
	err = tp.smr.CheckProposal(block, justify, validators)
	if err != nil {
		tp.log.Error("consensus:tdpos:CheckMinerMatch: bft IsQuorumCertValidate failed", "proposalQC:[height]", block.GetHeight(),
			"proposalQC:[id]", utils.F(block.GetBlockid()), "justifyQC:[height]", justify.GetProposalView(),
			"justifyQC:[id]", utils.F(justify.GetProposalId()), "error", err)
		return false, err
	}
	return true, nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回是否需要truncate, 返回写consensusStorage, 返回err
func (tp *tdposConsensus) ProcessBeforeMiner(height, timestamp int64) ([]byte, []byte, error) {
	term, pos, blockPos := tp.election.minerScheduling(timestamp)
	if blockPos < 0 || term != tp.election.curTerm || blockPos >= tp.election.blockNum || pos >= tp.election.proposerNum {
		tp.log.Warn("consensus:tdpos:ProcessBeforeMiner: timeoutBlockErr", "term", term, "tp.election.curTerm", tp.election.curTerm,
			"blockPos", blockPos, "tp.election.blockNum", tp.election.blockNum, "pos", pos, "tp.election.proposerNum", tp.election.proposerNum)
		return nil, nil, ErrTimeoutBlock
	}
	if tp.election.validators[pos] != tp.election.address {
		return nil, nil, ErrTimeoutBlock
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
	// 即本地smr的HightQC和账本TipId不相等，tipId尚未收集到足够签名，回滚到本地HighQC，重做区块
	tipBlock := tp.election.ledger.GetTipBlock()
	// smr返回一个裁剪目标，供miner模块直接回滚并出块
	truncate, qc, err := tp.smr.ResetProposerStatus(tipBlock, tp.election.ledger.QueryBlockHeader, tp.election.validators)
	if err != nil {
		return nil, nil, err
	}
	// 候选人组仅一个时无需操作
	if qc == nil {
		return nil, nil, nil
	}

	qcQuorumCert, _ := qc.(*quorumcert.QuorumCert)
	oldQC, _ := common.NewToOldQC(qcQuorumCert)
	storage.Justify = oldQC
	// 重做时还需要装载标定节点TipHeight，复用TargetBits作为回滚记录，便于追块时获取准确快照高度
	if truncate {
		tp.log.Warn("consensus:tdpos:ProcessBeforeMiner: last block not confirmed, walk to previous block",
			"target", utils.F(qc.GetProposalId()), "ledger", tipBlock.GetHeight())
		storage.TargetBits = int32(tipBlock.GetHeight())
		storageBytes, _ := json.Marshal(storage)
		return qc.GetProposalId(), storageBytes, nil
	}
	storageBytes, _ := json.Marshal(storage)
	return nil, storageBytes, nil
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
		tp.log.Warn("consensus:tdpos:ProcessConfirmBlock: parse storage error", "err", err)
		return err
	}
	var justify quorumcert.QuorumCertInterface
	if bv != nil && block.GetHeight() > tp.status.StartHeight {
		justify, err = common.OldQCToNew(bv)
		if err != nil {
			tp.log.Error("consensus:tdpos:ProcessConfirmBlock: OldQCToNew error", "err", err, "blockId", utils.F(block.GetBlockid()))
			return err
		}
	}

	// 查看本地是否是最新round的生产者
	_, pos, blockPos := tp.election.minerScheduling(block.GetTimestamp())
	if blockPos < 0 || blockPos >= tp.election.blockNum || pos >= tp.election.proposerNum {
		tp.log.Debug("consensus:tdpos:ProcessConfirmBlock: minerScheduling overflow.")
		return ErrSchedule
	}
	var nextValidators []string
	if tp.election.validators[pos] == tp.election.address && string(block.GetProposer()) == tp.election.address {
		// 如果是当前矿工，检测到下一轮需变更validates，且下一轮proposer并不在节点列表中，此时需在广播列表中新加入节点
		nextValidators = tp.election.GetValidators(block.GetHeight() + 1)
	}

	// 包装成统一入口访问smr
	if err = tp.smr.KeepUpWithBlock(block, justify, nextValidators); err != nil {
		tp.log.Warn("consensus:tdpos:ProcessConfirmBlock: update smr error.", "error", err)
		return err
	}
	return nil
}

// 共识实例的启动逻辑
func (tp *tdposConsensus) Start() error {
	// 注册合约方法
	for method, f := range tp.kMethod {
		// 若有历史句柄，删除老句柄
		tp.contract.GetKernRegistry().UnregisterKernMethod(tp.election.bindContractBucket, method)
		tp.contract.GetKernRegistry().RegisterKernMethod(tp.election.bindContractBucket, method, f)
	}
	if tp.election.enableChainedBFT {
		err := tp.initBFT()
		if err != nil {
			tp.log.Warn("tdposConsensus start init bft error", "err", err.Error())
			return err
		}
	}
	return nil
}

func (tp *tdposConsensus) initBFT() error {
	// create smr/ chained-bft实例, 需要新建CBFTCrypto、pacemaker和saftyrules实例
	cryptoClient := cCrypto.NewCBFTCrypto(tp.cCtx.Address, tp.cCtx.Crypto)
	qcTree := quorumcert.InitQCTree(tp.status.StartHeight, tp.cCtx.Ledger, tp.cCtx.XLog)
	if qcTree == nil {
		tp.log.Error("consensus:tdpos:NewTdposConsensus: init QCTree err", "startHeight", tp.status.StartHeight)
		return errors.New("init bft init qcTree error")
	}
	pacemaker := &chainedBft.DefaultPaceMaker{
		CurrentView: tp.status.StartHeight,
	}
	// 重启状态检查1，pacemaker需要重置
	tipHeight := tp.cCtx.Ledger.QueryTipBlockHeader().GetHeight()
	if !bytes.Equal(qcTree.GetGenesisQC().In.GetProposalId(), qcTree.GetRootQC().In.GetProposalId()) {
		pacemaker.CurrentView = tipHeight - 1
	}
	saftyrules := &chainedBft.DefaultSaftyRules{
		Crypto: cryptoClient,
		QcTree: qcTree,
		Log:    tp.cCtx.XLog,
	}
	smr := chainedBft.NewSmr(tp.bcName, tp.election.address, tp.log, tp.cCtx.Network, cryptoClient, pacemaker, saftyrules, tp.election, qcTree)
	// 重启状态检查2，重做tipBlock，此时需重装载justify签名
	if !bytes.Equal(qcTree.GetGenesisQC().In.GetProposalId(), qcTree.GetRootQC().In.GetProposalId()) {
		for i := int64(0); i < 3; i++ {
			b, err := tp.cCtx.Ledger.QueryBlockHeaderByHeight(tipHeight - i)
			if err != nil {
				break
			}
			smr.LoadVotes(b.GetPreHash(), tp.GetJustifySigns(b))
		}
	}
	tp.smr = smr
	tp.smr.Start()
	return nil
}

// 共识实例的挂起逻辑, 另: 若共识实例发现绑定block结构有误，会直接停掉当前共识实例并panic
func (tp *tdposConsensus) Stop() error {
	// 注销合约方法
	for method, _ := range tp.kMethod {
		// 若有历史句柄，删除老句柄
		tp.contract.GetKernRegistry().UnregisterKernMethod(tp.election.bindContractBucket, method)
	}
	if tp.election.enableChainedBFT {
		tp.smr.Stop()
	}
	return nil
}

// 共识占用blockinterface的专有存储，特定共识需要提供parse接口，在此作为接口高亮
func (tp *tdposConsensus) ParseConsensusStorage(block cctx.BlockInterface) (interface{}, error) {
	return ParseConsensusStorage(block)
}

func (tp *tdposConsensus) GetConsensusStatus() (consensus.ConsensusStatus, error) {
	return tp.status, nil
}

func (tp *tdposConsensus) GetJustifySigns(block cctx.BlockInterface) []*chainedBftPb.QuorumCertSign {
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil
	}
	signs := common.OldSignToNew(b)
	return signs
}
