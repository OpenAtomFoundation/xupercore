package xpoa

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
)

// TODO: 目前时间关系，暂时将xpoa放置在三代合约，后可以收敛至xuper5的kernel合约
const (
	// 为了避免调用utxoVM systemCall方法, 直接通过ledger读取xpoa合约存储
	// ATTENTION: 此处xpoaBucket和xpoaKey必须和对应三代合约严格一致，并且该xpoa隐式限制只能包含xmodel机制的ledger才可调用
	xpoaBucket = "xpoa"
	xpoaKey    = "VALIDATES"

	maxsleeptime = time.Millisecond * 10
)

var (
	MinerSelectErr   = errors.New("Node isn't a miner, calculate error.")
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
	NotEnoughVotes   = errors.New("Cannot get enough votes of last view from replicas.")
	InvalidQC        = errors.New("QC struct is invalid.")
)

func init() {
	consensus.Register("xpoa", NewXpoaConsensus)
}

type xpoaConsensus struct {
	bctx          xcontext.BaseCtx
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
		cCtx.XLog.Error("Xpoa::NewSingleConsensus::CryptoClient in context is nil")
		return nil
	}
	if cCtx.Ledger == nil {
		cCtx.XLog.Error("Xpoa::NewSingleConsensus::Ledger in context is nil")
		return nil
	}
	if cCfg.ConsensusName != "xpoa" {
		cCtx.XLog.Error("Xpoa::NewSingleConsensus::consensus name in config is wrong", "name", cCfg.ConsensusName)
		return nil
	}

	// 创建smr实例过程
	// 解析xpoaconfig
	xconfig := &xpoaConfig{}
	err := json.Unmarshal([]byte(cCfg.Config), xconfig)
	if err != nil {
		cCtx.XLog.Error("Xpoa::NewSingleConsensus::xpoa struct unmarshal error", "error", err)
		return nil
	}
	// create xpoaSchedule
	schedule := &xpoaSchedule{
		// TODO: +Address from p2p state
		period:    xconfig.Period,
		blockNum:  xconfig.BlockNum,
		addrToNet: make(map[string]string),
		ledger:    cCtx.Ledger,
	}
	// xpoaSchedule 实现了ProposerElectionInterface接口，接口定义了validators操作
	// 重启时需要使用最新的validator数据，而不是initValidators数据
	var validators []string
	for _, v := range xconfig.InitProposer {
		validators = append(validators, v.Address)
		schedule.addrToNet[v.Address] = v.Neturl
	}
	reader, _ := schedule.ledger.GetTipXMSnapshotReader()
	res, err := reader.Get(xpoaBucket, []byte(xpoaKey))
	if snapshotValidators, _ := loadValidatorsMultiInfo(res, &schedule.addrToNet); snapshotValidators != nil {
		validators = snapshotValidators
	}
	schedule.validators = validators
	// create smr/ chained-bft实例, 需要新建CBFTCrypto、pacemaker和saftyrules实例
	cryptoClient := cCrypto.NewCBFTCrypto(cCtx.Address, cCtx.Crypto)
	qcTree := common.InitQCTree(cCfg.StartHeight, cCtx.Ledger)
	pacemaker := &chainedBft.DefaultPaceMaker{
		StartView: cCfg.StartHeight,
	}
	saftyrules := &chainedBft.DefaultSaftyRules{
		Crypto: cryptoClient,
		QcTree: qcTree,
	}
	smr := chainedBft.NewSmr(cCtx.BcName, schedule.address, cCtx.XLog, cCtx.Network, cryptoClient, pacemaker, saftyrules, schedule, qcTree)
	go smr.Start()

	// 创建status实例
	status := &XpoaStatus{
		Version:     xconfig.Version,
		StartHeight: cCfg.StartHeight,
		Index:       cCfg.Index,
		election:    schedule,
	}

	// create xpoaConsensus实例
	xpoa := &xpoaConsensus{
		bctx:          cCtx.BaseCtx,
		election:      schedule,
		isProduce:     make(map[int64]bool),
		config:        xconfig,
		initTimestamp: time.Now().UnixNano(),
		smr:           smr,
		status:        status,
	}
	return xpoa
}

// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
func (x *xpoaConsensus) CompeteMaster(height int64) (bool, bool, error) {
Again:
	t := time.Now().UnixNano() / 1e6
	key := t / x.election.period
	sleep := x.election.period - t%x.election.period
	if sleep > int64(maxsleeptime) {
		sleep = int64(maxsleeptime)
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
	x.election.UpdateValidator()
	leader := x.election.GetLeader(height)
	if leader == x.election.address {
		x.bctx.XLog.Info("Xpoa::CompeteMaster", "isMiner", true, "height", height)
		// TODO: 首次切换为矿工时SyncBlcok, Bug: 可能会导致第一次出块失败
		needSync := x.election.ledger.GetTipBlock().GetHeight() == 0 || string(x.election.ledger.GetTipBlock().GetProposer()) != leader
		return true, needSync, nil
	}
	x.bctx.XLog.Info("Xpoa::CompeteMaster", "isMiner", false, "height", height)
	return false, false, nil
}

// CheckMinerMatch 查看block是否合法
// ATTENTION: TODO: 上层需要先检查VerifyBlock(block)
func (x *xpoaConsensus) CheckMinerMatch(ctx xcontext.BaseCtx, block cctx.BlockInterface) (bool, error) {
	// 验证矿工身份
	proposer := x.election.GetLocalLeader(block.GetTimestamp(), block.GetHeight())
	if proposer != string(block.GetProposer()) {
		x.bctx.XLog.Warn("Xpoa::CheckMinerMatch::calculate proposer error", "want", proposer, "have", string(block.GetProposer()))
		return false, MinerSelectErr
	}
	// 获取block中共识专有存储, 检查justify是否符合要求
	justifyBytes, err := block.GetConsensusStorage()
	if err != nil {
		return false, err
	}
	justify := &XpoaStorage{}
	if err := json.Unmarshal(justifyBytes, justify); err != nil {
		return false, err
	}
	pNode := x.smr.BlockToProposalNode(block)
	err = x.smr.GetSaftyRules().CheckProposal(pNode.In, justify.Justify, x.election.GetValidators(block.GetHeight()))
	if err != nil {
		x.bctx.XLog.Warn("Xpoa::CheckMinerMatch::bft IsQuorumCertValidate failed", "proposalQC:[height]", pNode.In.GetProposalView(),
			"proposalQC:[id]", pNode.In.GetProposalId, "justifyQC:[height]", justify.Justify.GetProposalView(),
			"justifyQC:[id]", justify.Justify.GetProposalId(), "error", err)
		return false, err
	}
	return true, nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回是否需要truncate, 返回写consensusStorage, 返回err
func (x *xpoaConsensus) ProcessBeforeMiner(timestamp int64) (bool, []byte, error) {
	// 再次检查目前是否是矿工，TODO: check是否有必要，因为和sync抢一把锁，按道理不会有这个问题
	_, pos, _ := x.election.minerScheduling(timestamp, len(x.election.validators))
	if x.election.validators[pos] != x.election.address {
		return false, nil, MinerSelectErr
	}
	// 即本地smr的HightQC和账本TipId不相等，tipId尚未收集到足够签名，回滚到本地HighQC，重做区块
	if !bytes.Equal(x.smr.GetHighQC().GetProposalId(), x.election.ledger.GetTipBlock().GetBlockid()) {
		// 单个节点不存在投票验证的hotstuff流程，因此返回true
		if len(x.election.validators) == 1 {
			return false, nil, nil
		}
		x.bctx.XLog.Warn("smr::ProcessBeforeMiner::last block not confirmed, walk to previous block", "ledger", x.election.ledger.GetTipBlock().GetHeight(),
			"HighQC", x.smr.GetHighQC().GetProposalView())
		return true, nil, NotEnoughVotes
	}
	qc := x.smr.GetHighQC()
	qcQuorumCert, ok := qc.(*chainedBft.QuorumCert)
	if !ok {
		return true, nil, InvalidQC
	}
	s := &XpoaStorage{
		Justify: qcQuorumCert,
	}
	bytes, _ := json.Marshal(s)
	return false, bytes, nil
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
func (x *xpoaConsensus) CalculateBlock(block cctx.BlockInterface) error {
	return nil
}

// ProcessConfirmBlock 用于确认块后进行相应的处理
func (x *xpoaConsensus) ProcessConfirmBlock(block cctx.BlockInterface) error {
	// confirm的第一步：不管是否为当前Leader，都需要更新本地voteQC树，保证当前block的justify votes被写入本地账本
	// 获取block中共识专有存储, 检查justify是否符合要求
	justifyBytes, err := block.GetConsensusStorage()
	if err != nil {
		return err
	}
	justify := &XpoaStorage{}
	if err := json.Unmarshal(justifyBytes, justify); err != nil {
		return err
	}
	x.smr.UpdateJustifyQcStatus(justify.Justify)
	// 查看本地是否是最新round的生产者
	_, pos, _ := x.election.minerScheduling(block.GetTimestamp(), len(x.election.validators))
	// 如果是当前矿工，检测到下一轮需变更validates，且下一轮proposer并不在节点列表中，此时需在广播列表中新加入节点
	if x.election.validators[pos] == x.election.address && string(block.GetProposer()) == x.election.address {
		validators := x.election.GetValidators(block.GetHeight() + 1)
		if err := x.smr.ProcessProposal(block.GetHeight(), block.GetBlockid(), validators); err != nil {
			x.bctx.XLog.Warn("smr::ProcessConfirmBlock::bft next proposal failed", "error", err)
			return err
		}
		x.bctx.XLog.Info("smr::ProcessConfirmBlock::miner confirm finish", "ledger:[height]", x.election.ledger.GetTipBlock().GetHeight(), "viewNum", x.smr.GetCurrentView())
		return nil
	}
	// 若当前节点不在候选人节点中，直接调用smr生成新的qc树
	pNode := x.smr.BlockToProposalNode(block)
	x.smr.UpdateQcStatus(pNode)
	return nil
}

// 共识实例的挂起逻辑, 另: 若共识实例发现绑定block结构有误，会直接停掉当前共识实例并panic
func (x *xpoaConsensus) Stop() error {
	x.smr.Stop()
	return nil
}

// 共识实例的启动逻辑
func (x *xpoaConsensus) Start() error {
	x.smr.Start()
	return nil
}

// 共识占用blockinterface的专有存储，特定共识需要提供parse接口，在此作为接口高亮
func (x *xpoaConsensus) ParseConsensusStorage(block cctx.BlockInterface) (interface{}, error) {
	store := XpoaStorage{}
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &store)
	if err != nil {
		x.bctx.XLog.Error("Xpoa::ParseConsensusStorage invalid consensus storage", "err", err)
		return nil, err
	}
	return store, nil
}

func (x *xpoaConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	return x.status, nil
}
