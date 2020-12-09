package xpoa

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

// TODO: 目前时间关系，暂时将xpoa放置在三代合约，后可以收敛至xuper5的kernel合约
const (
	/* 为了避免调用utxoVM systemCall方法, 直接通过ledger读取xpoa合约存储
	 * ATTENTION: 此处xpoaBucket和xpoaKey必须和对应三代合约严格一致，并且该xpoa隐式限制只能包含xmodel机制的ledger才可调用
	 */
	xpoaBucket = "xpoa"
	xpoaKey    = "VALIDATES"
)

var (
	MinerSelectErr = errors.New("Node isn't a miner, calculate error.")
)

// XpoaStorage xpoa占用block中consensusStorage json串的格式
type XpoaStorage struct {
	Justify *chainedBft.QuorumCert `json:"Justify,omitempty"`
}

type xpoaContractJson struct {
	Address string `json:"address"`
}

type XpoaConsensus struct {
	election  XpoaSchedule
	smr       chainedBft.Smr
	isProduce map[int64]bool
}

// xpoaSchedule 实现了ProposerElectionInterface接口，接口定义了validators操作
type XpoaSchedule struct {
	address     string
	period      int64 // 出块间隔
	blockNum    int64 // 每轮每个候选人最多出多少块
	validators  []string

	ledger cctx.LedgerCtxInConsensus
}

func (s *XpoaSchedule) minerScheduling(timestamp int64, length int) (term int64, pos int64, blockPos int64) {
	// 每一轮的时间
	termTime := s.period * int64(length) * s.blockNum
	// 每个矿工轮值时间
	posTime := s.period * s.blockNum
	term = (timestamp)/termTime + 1
	//10640483 180000
	resTime := timestamp - (term-1)*termTime
	pos = resTime / posTime
	resTime = resTime - (resTime/posTime)*posTime
	blockPos = resTime/s.period + 1
	return
}

/* GetLeader 根据输入的round，计算应有的proposer，实现election接口
 * 该方法主要为了支撑smr扭转和矿工挖矿，在handleReceivedProposal阶段会调用该方法
 * 由于xpoa主逻辑包含回滚逻辑，因此回滚逻辑必须在ProcessProposal进行
 * ATTENTION: tipBlock是一个隐式依赖状态
 */
func (s *XpoaSchedule) GetLeader(round int64) string {
	// 若该round已经落盘，则直接返回历史信息，eg. 矿工在当前round的情况
	if b, err := s.ledger.QueryBlockByHeight(round); err != nil {
		return b.GetProposer()
	}
	tipBlock := s.ledger.GetTipBlock()
	// 作为当前round的replica，若刚好预测的是当前round，则直接根据timestamp进行调度
	if tipBlock.GetProposer() != s.address && tipBlock.GetHeight()+1 == round {
		_, pos, _ := s.minerScheduling(time.Now().UnixNano(), len(s.validators))
		return s.validators[pos]
	}
	// 需要计算下一轮leader的情况，包含: 1: 下一高度未有validators变更 2: 下一高度有validators变更
	// 首先查看round是否合法, 合法只包括两种情况: 1.作为当前round的Leader，计算下一个Leader 2: 作为当前round的replica，计算下一轮的proposer
	if (tipBlock.GetHeight()+1 == round && tipBlock.GetProposer() == s.address) ||
		tipBlock.GetHeight()+2 == round {
		// xpoa的validators变更在包含变更tx的block的后3个块后生效, 即当B0包含了变更tx
		b, err := s.ledger.QueryBlockByHeight(round - 3)
		if err != nil {
			nextTime := time.Now().UnixNano()
			_, pos, _ := s.minerScheduling(nextTime, len(s.validators))
			return s.validators[pos]
		}
		// 在B3时validators才正式统一变更
		nextValidators := s.getValidatesByBlockId(b.GetBlockid())
		// 计算round对应的timestamp时间
		nextTime := time.Now().UnixNano()
		_, pos, _ := s.minerScheduling(nextTime, len(nextValidators))
		return nextValidators[pos]
	}
	// 该接口仅限于判断上述情况，其余为空
	return ""
}

/* GetLocalLeader 用于收到一个新块时, 验证该块的时间戳和proposer是否能与本地计算结果匹配
 */
func (s *XpoaSchedule) GetLocalLeader(timestamp int64, round int64) string {
	// xpoa.lg.Info("ConfirmBlock Propcess update validates")
	// ATTENTION: 获取候选人信息时，时刻注意拿取的是check目的round的前三个块，候选人变更是在3个块之后生效，即round-3
	b, err := s.ledger.QueryBlockByHeight(round - 3)
	if err != nil {
		return ""
	}
	localValidators := s.getValidatesByBlockId(b.GetBlockid())
	if localValidators == nil && err == nil {
		// 使用初始变量
		return ""
	}
	_, pos, _ := s.minerScheduling(timestamp, len(localValidators))
	return localValidators[pos]
}

// AddressEqual 判断两个validators地址是否相等
func AddressEqual(a []string, b []string) bool {
	return true
}

// getValidatesByBlockId 根据当前输入blockid，用快照的方式在xmodel中寻找<=当前blockid的最新的候选人值，若无则使用xuper.json中指定的初始值
func (s *XpoaSchedule) getValidatesByBlockId(blockId []byte) ([]string, error) {
	reader, err := s.ledger.GetSnapShotWithBlock(blockId)
	if err != nil {
		// xpoa.lg.Error("Xpoa updateValidates getCurrentValidates error", "CreateSnapshot err:", err)
		return nil, err
	}
	res, err := reader.Get(xpoaBucket, []byte(xpoaKey))
	if res == nil {
		// 即合约还未被调用，未有变量更新
		return nil, nil
	}
	contractInfo := xpoaContractJson{}
	if err = json.Unmarshal(res, &contractInfo); err != nil {
		return nil, err
	}
	// validators由分号隔开
	validators := strings.Split(contractInfo.Address, ";")
	if len(validators) == 0 {
		return nil, nil
	}
	return validators, nil
}

func (s *XpoaSchedule) GetValidators(round int64) []string {
	// xpoa的validators变更在包含变更tx的block的后3个块后生效, 即当B0包含了变更tx
	b, err := s.ledger.QueryBlockByHeight(round - 3)
	if err != nil {
		// TODO: 此处返回的是初始值
		return s.validators
	}
	// 在B3时validators才正式统一变更
	validators, err := s.getValidatesByBlockId(b.GetBlockid())
	if err != nil {
		// TODO: 此处返回的是初始值
		return s.validators
	}
	return validators
}

func (s *XpoaSchedule) GetCurrentValidators() []string {
	return s.validators
}

func (s *XpoaSchedule) UpdateValidatorSet(newValidates []string) {

}

// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
func (x *XpoaConsensus) CompeteMaster(height int64) (bool, bool, error) {
Again:
	t := time.Now().UnixNano()
	key := t / x.election.period
	sleep := x.election.period - t%x.election.period
	maxsleeptime := time.Millisecond * 10
	if sleep > int64(maxsleeptime) {
		sleep = int64(maxsleeptime)
	}
	v, ok := x.isProduce[key]
	if !ok || v == false {
		x.isProduce[key] = true
	} else {
		time.Sleep(time.Duration(sleep))
		// 定期清理isProduce
		cleanProduceMap(x.isProduce)
		goto Again
	}

	// xpoa.lg.Info("Compete Master", "height", height)
	// update validates????
	leader := x.election.GetLeader(height)
	if leader == x.election.address {
		// xpoa.lg.Trace("Xpoa CompeteMaster now xterm infos", "master", true, "height", height)
		// TODO: 首次切换为矿工时SyncBlcok, Bug: 可能会导致第一次出块失败
		needSync := x.election.ledger.GetTipBlock().GetHeight() == 0 || x.election.ledger.GetTipBlock().GetProposer() != leader
		return true, needSync, nil
	}

	// xpoa.lg.Trace("Xpoa CompeteMaster now xterm infos", "master", false, "height", height)
	return false, false, nil
}

// CheckMinerMatch 查看block是否合法
func (x *XpoaConsensus) CheckMinerMatch(ctx xcontext.BaseCtx, block cctx.BlockInterface) (bool, error) {
	// TODO: 应由saftyrules模块负责check, xpoa需要组合一个defaultsaftyrules, 在saftyrules里调用ledger的verifyBlock
	if ok, err := x.election.ledger.VerifyBlock(block, ctx.XLog.GetLogId()); !ok || err != nil {
		// xpoa.lg.Info("XPoa CheckMinerMatch VerifyBlock not ok")
		return ok, err
	}
	// 验证矿工身份
	proposer := x.election.GetLocalLeader(block.GetTimestamp(), block.GetHeight())
	if proposer == "" {
		//xpoa.lg.Warn("CheckMinerMatch getProposerWithTime error", "error", err.Error())
		return false, nil
	}
	// 获取block中共识专有存储
	justifyBytes := block.GetConsensusStorage()
	justify := &XpoaStorage{}
	if err := json.Unmarshal(justifyBytes, justify); err != nil {
		return false, err
	}
	pNode := x.smr.BlockToProposalNode(block)
	ok, err := x.smr.GetSaftyRules.IsQuorumCertValidate(pNode.In, justify.Justify)
	if err != nil || !ok {
		// xpoa.lg.Warn("CheckMinerMatch bft IsQuorumCertValidate failed", "logid", header.Logid, "error", err)
		return false, nil
	}
	return proposer == block.GetProposer(), nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回是否需要truncate, 返回写consensusStorage, 返回err
func (x *XpoaConsensus) ProcessBeforeMiner(timestamp int64) (bool, []byte, error) {
	// 再次检查目前是否是矿工，TODO: check是否有必要，因为和sync抢一把锁，按道理不会有这个问题
	_, pos, _ := x.election.minerScheduling(timestamp, len(x.election.validators))
	if x.election.validators[pos] != x.election.address {
		return false, nil, MinerSelectErr
	}
	// 即本地smr的HightQC和账本TipId不相等，tipId尚未收集到足够签名，回滚到本地HighQC
	if !bytes.Equal(x.smr.GetHighQC().GetProposalId(), x.election.ledger.GetTipBlock())
		if len(xpoa.proposerInfos) == 1 {
			res["quorum_cert"] = nil
			return res, true
		}
		// xpoa.lg.Warn("ProcessBeforeMiner last block not confirmed, walk to previous block")
		// targetId := x.smr.GetHighQC().GetProposalId()
		return true, nil, nil
	}
	qc, err := x.smr.GetHighQC()
	qcQuorumCert, ok := qc.(*chainedBft.QuorumCert)
	if !ok {

	}
	s := &XpoaStorage{
		Justify: qcQuorumCert,
	}
	bytes, err := json.Marshal(s)
	return false, bytes, nil
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
func (x *XpoaConsensus) CalculateBlock(block cctx.BlockInterface) error {
	return nil
}

// ProcessConfirmBlock 用于确认块后进行相应的处理
func (x *XpoaConsensus) ProcessConfirmBlock(block cctx.BlockInterface) error {
	// 查看本地是否是最新round的生产者
	_, pos, _ := x.election.minerScheduling(timestamp, len(x.election.validators))
	if x.election.validators[pos] == x.election.address && block.GetProposer() == x.election.address {
		// 如果是当前矿工，检测到下一轮需变更validates，且下一轮proposer并不在节点列表中，此时需在广播列表中新加入节点
		validators := x.election.GetValidators()
		b, err := s.ledger.QueryBlockByHeight(block.GetHeight() - 3)
		if err == nil {
			if v, err = x.election.getValidatesByBlockId(b.GetBlockid()); err == nil {
				validators = v
			}
		}
		if err := x.smr.ProcessProposal(block.GetHeight(), block.GetBlockid(), validators); err != nil {
			// xpoa.lg.Warn("ProcessConfirmBlock: bft next proposal failed", "error", err)
			return err
		}
		// xpoa.lg.Info("Now Confirm finish", "ledger height", xpoa.ledger.GetMeta().TrunkHeight, "viewNum", xpoa.bftPaceMaker.CurrentView())
		return nil
	}
	// 若当前节点不在候选人节点中，直接调用smr的
	pNode := x.smr.BlockToProposalNode(block)
	x.smr.UpdateQcStatus(pNode)
}

/*
// GetStatus 获取区块链共识信息
func (x *XpoaConsensus) GetConsensusStatus() (cctx.ConsensusStatus, error) {
	return nil, nil
}
*/

// 共识实例的挂起逻辑, 另: 若共识实例发现绑定block结构有误，会直接停掉当前共识实例并panic
func (x *XpoaConsensus) Stop() error {
	return nil
}

// 共识实例的重启逻辑, 用于共识回滚
func (x *XpoaConsensus) Start() error {
	return nil
}

// 共识占用blockinterface的专有存储，特定共识需要提供parse接口，在此作为接口高亮
func (x *XpoaConsensus) ParseConsensusStorage(block cctx.BlockInterface) (interface{}, error) {
	return nil, nil
}

func cleanProduceMap(isProduce map[int64]bool) {

}
