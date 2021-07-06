package tdpos

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/lib/logs"
)

// tdposSchedule 实现了ProposerElectionInterface接口，接口定义了proposers操作
// tdposSchedule 是tdpos的主要结构，其能通过合约调用来变更smr的候选人信息，并且向smr提供对应round的候选人信息
type tdposSchedule struct {
	address string
	// 出块间隔, 单位为毫秒
	period int64
	// 每轮每个候选人最多出多少块
	blockNum int64
	// 每轮选出的候选人个数
	proposerNum int64
	// 更换候选人时间间隔
	alternateInterval int64
	// 更换轮时间间隔
	termInterval int64
	// 起始时间
	initTimestamp int64
	// 是否开启chained-bft
	enableChainedBFT bool

	// 当前validators的address
	validators         []string
	initValidators     []string
	curTerm            int64
	miner              string
	startHeight        int64
	consensusName      string
	consensusVersion   int64
	bindContractBucket string

	log    logs.Logger
	ledger cctx.LedgerRely
}

// NewSchedule 新建schedule实例
func NewSchedule(xconfig *tdposConfig, log logs.Logger, ledger cctx.LedgerRely, startHeight int64) *tdposSchedule {
	schedule := &tdposSchedule{
		period:             xconfig.Period,
		blockNum:           xconfig.BlockNum,
		proposerNum:        xconfig.ProposerNum,
		alternateInterval:  xconfig.AlternateInterval,
		termInterval:       xconfig.TermInterval,
		initTimestamp:      xconfig.InitTimestamp,
		validators:         (xconfig.InitProposer)["1"],
		startHeight:        startHeight,
		consensusName:      "tdpos",
		consensusVersion:   xconfig.Version,
		bindContractBucket: tdposBucket,
		log:                log,
		ledger:             ledger,
	}
	index := 0
	for index < len(schedule.validators) {
		key := schedule.validators[index]
		schedule.initValidators = append(schedule.initValidators, key)
		index++
	}
	// 重启时需要使用最新的validator数据，而不是initValidators数据
	tipBlock := schedule.ledger.GetTipBlock()
	s, err := tipBlock.GetConsensusStorage()
	if err != nil {
		return nil
	}
	refresh, err := schedule.CalOldProposers(tipBlock.GetHeight(), tipBlock.GetTimestamp(), s)
	if err != nil && err != heightTooLow {
		schedule.log.Error("Tdpos::NewSchedule error", "err", err)
		return nil
	}
	if !common.AddressEqual(schedule.validators, refresh) && len(refresh) != 0 {
		schedule.validators = refresh
	}
	if xconfig.EnableBFT != nil {
		schedule.enableChainedBFT = true
		schedule.consensusName = "xpos"
		schedule.bindContractBucket = xposBucket
	}
	return schedule
}

// miner 调度算法, 依据时间进行矿工节点调度
// s.alternateInterval >= s.period  &&  s.termInterval >= s.alternateInterval
func (s *tdposSchedule) minerScheduling(timestamp int64) (term int64, pos int64, blockPos int64) {
	// timstamp单位为unixnano, 配置文件中均为毫秒
	if timestamp < s.initTimestamp {
		return
	}
	T := timestamp / int64(time.Millisecond)
	initT := s.initTimestamp / int64(time.Millisecond)
	// 每一轮的时间
	// |<-termInterval->|<-(blockNum - 1) * period->|<-alternateInterval->|
	// |................|NODE1......................|.....................|NODE2.....
	termTime := s.termInterval + (s.blockNum-1)*s.proposerNum*s.period + (s.proposerNum-1)*s.alternateInterval
	// 每个矿工轮值时间
	term = (T-initT)/termTime + 1
	termBegin := initT + (term-1)*termTime + s.termInterval - s.alternateInterval
	if termBegin >= T {
		return term, 0, -1
	}
	posTime := s.alternateInterval + s.period*(s.blockNum-1)
	pos = (T - termBegin) / posTime
	proposerBegin := termBegin + pos*posTime + s.alternateInterval - s.period
	if proposerBegin >= T {
		return term, pos, -1
	}
	blockPos = (T - proposerBegin) / s.period
	return
}

// getSnapshotKey 获取当前tip高度对应key的快照
func (s *tdposSchedule) getSnapshotKey(height int64, bucket string, key []byte) ([]byte, error) {
	if height <= 0 {
		return nil, heightTooLow
	}
	block, err := s.ledger.QueryBlockByHeight(height)
	if err != nil {
		s.log.Debug("tdpos::getSnapshotKey::QueryBlockByHeight err.", "err", err)
		return nil, err
	}
	reader, err := s.ledger.CreateSnapshot(block.GetBlockid())
	if err != nil {
		s.log.Error("tdpos::getSnapshotKey::CreateSnapshot err.", "err", err)
		return nil, err
	}
	versionData, err := reader.Get(bucket, key)
	if err != nil {
		s.log.Debug("tdpos::getSnapshotKey::reader.Get err.", "err", err)
		return nil, err
	}
	if versionData == nil || versionData.PureData == nil {
		return nil, nil
	}
	return versionData.PureData.Value, nil
}

// GetLeader 根据输入的round，计算应有的proposer，实现election接口
// 该方法主要为了支撑smr扭转和矿工挖矿，在handleReceivedProposal阶段会调用该方法
// 由于主逻辑包含回滚逻辑，因此回滚逻辑必须在ProcessProposal进行
// ATTENTION: tipBlock是一个隐式依赖状态
func (s *tdposSchedule) GetLeader(round int64) string {
	// 若该round已经落盘，则直接返回历史信息，eg. 矿工在当前round的情况
	if b, err := s.ledger.QueryBlockByHeight(round); err == nil {
		return string(b.GetProposer())
	}
	proposers := s.GetValidators(round)
	if proposers == nil {
		return ""
	}
	addTime := s.calAddTime(round, s.ledger.GetTipBlock().GetHeight())
	_, pos, _ := s.minerScheduling(time.Now().UnixNano() + addTime)
	if pos >= s.proposerNum {
		return ""
	}
	return proposers[pos]
}

func (s *tdposSchedule) calAddTime(round int64, tipHeight int64) int64 {
	_, nowPos, nowBlockPos := s.minerScheduling(time.Now().UnixNano())
	if round <= tipHeight {
		return 0
	}
	if nowPos == s.proposerNum-1 && nowBlockPos == s.blockNum-1 { // 下一轮换term
		return s.termInterval * int64(time.Millisecond)
	}
	if nowBlockPos == s.blockNum-1 { // 下一轮换proposer
		return s.alternateInterval * int64(time.Millisecond)
	}
	return s.period * int64(time.Millisecond)
}

// GetValidators election接口实现，获取指定round的候选人节点Address
func (s *tdposSchedule) GetValidators(round int64) []string {
	if round < s.startHeight+3 {
		return s.initValidators
	}
	var calErr error
	var proposers []string
	block, err := s.ledger.QueryBlockByHeight(round)
	if err != nil {
		proposers, calErr = s.CalculateProposers(round)
	} else {
		storage, _ := block.GetConsensusStorage()
		proposers, calErr = s.CalOldProposers(round, block.GetTimestamp(), storage)
	}
	if calErr != nil {
		s.log.Debug("tdpos::GetValidators::CalculateProposers err", "err", calErr)
		return nil
	}
	return proposers
}

// GetIntAddress election接口实现，获取候选人地址到网络地址的映射，for unit test
func (s *tdposSchedule) GetIntAddress(address string) string {
	return ""
}

// updateProposers 根据各合约存储计算当前proposers
func (s *tdposSchedule) UpdateProposers(height int64) bool {
	if height < s.startHeight+3 {
		return false
	}
	nextProposers, err := s.CalculateProposers(height)
	if err != nil || len(nextProposers) == 0 {
		return false
	}
	if !common.AddressEqual(nextProposers, s.validators) {
		s.log.Debug("tdpos::UpdateProposers", "origin", s.validators, "proposers", nextProposers)
		s.validators = nextProposers
		return true
	}
	return false
}

// CalculateProposers 用于计算下一个Round候选人的值，仅在updateProposers和smr接口时使用
func (s *tdposSchedule) CalculateProposers(height int64) ([]string, error) {
	if height < s.startHeight+3 {
		return s.initValidators, nil
	}
	addTime := s.calAddTime(height, s.ledger.GetTipBlock().GetHeight())
	inputTerm, _, _ := s.minerScheduling(time.Now().UnixNano() + addTime)
	if s.curTerm == inputTerm {
		return s.validators, nil
	}
	// 更新候选人时需要读取快照，并计算投票的top K
	p, err := s.calTopKNominator(height)
	if err != nil {
		s.log.Error("tdpos::CalculateProposers::calculateTopK err.", "err", err)
		return nil, err
	}
	return p, nil
}

// calTopKNominator 计算最新的投票的topK候选人并返回
func (s *tdposSchedule) calTopKNominator(height int64) ([]string, error) {
	if height < s.startHeight+3 {
		return s.initValidators, nil
	}
	// 获取nominate信息
	nKey := fmt.Sprintf("%s_%d_%s", s.consensusName, s.consensusVersion, nominateKey)
	res, err := s.getSnapshotKey(height-3, s.bindContractBucket, []byte(nKey))
	if err != nil {
		s.log.Error("tdpos::calculateTopK::getSnapshotKey err.", "err", err)
		return nil, err
	}
	// 未读到值时直接返回初始化值
	if res == nil {
		return s.initValidators, nil
	}
	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		s.log.Error("tdpos::calculateTopK::load nominate read set err.")
		return nil, err
	}
	var termBallotSli termBallotsSlice
	for candidate, _ := range nominateValue {
		candidateBallot := &termBallots{
			Address: candidate,
		}
		// 根据候选人信息获取vote选票信息
		key := fmt.Sprintf("%s_%d_%s%s", s.consensusName, s.consensusVersion, voteKeyPrefix, candidate)
		res, err := s.getSnapshotKey(height-3, s.bindContractBucket, []byte(key))
		if err != nil {
			s.log.Error("tdpos::calculateTopK::load vote read set err.")
			return nil, err
		}
		// 未有vote信息则跳过
		if res == nil {
			continue
		}
		voteValue := NewvoteValue()
		if err := json.Unmarshal(res, &voteValue); err != nil {
			return nil, err
		}
		for _, ballot := range voteValue {
			candidateBallot.Ballots += ballot
		}
		if candidateBallot.Ballots <= 0 {
			continue
		}
		termBallotSli = append(termBallotSli, candidateBallot)
	}
	if int64(termBallotSli.Len()) < s.proposerNum {
		s.log.Debug("tdpos::calculateTopK::Term publish proposer num less than config", "termVotes", termBallotSli)
		return s.initValidators, nil
	}
	// 计算topK候选人
	sort.Stable(termBallotSli)
	var proposers []string
	for i := int64(0); i < s.proposerNum; i++ {
		proposers = append(proposers, termBallotSli[i].Address)
	}
	return proposers, nil
}

// CalHisProposers 主要用于追块时、计算历史高度所对应的候选人值
func (s *tdposSchedule) CalOldProposers(height int64, timestamp int64, storage []byte) ([]string, error) {
	if height < s.startHeight+3 {
		return s.initValidators, nil
	}
	tipHeight := s.ledger.GetTipBlock().GetHeight()
	if tipHeight > height {
		// 情况一：读取历史值，height对应区块存在于账本中，此时分成历史Key读取和计算差值两部分
		return s.calHisValidators(height)
	}
	// 情况二：height对应区块不存在于账本中，即当前是一个节点追块逻辑，追一个新块
	tipTerm, err := s.getTerm(tipHeight)
	if err != nil {
		return nil, err
	}
	inputTerm, _, _ := s.minerScheduling(timestamp)
	// 最高高度的term仍然有效，则获取tipTerm开始的第一个高度的快照，然后获取历史候选人节点
	if tipTerm == inputTerm {
		return s.calHisValidators(tipHeight)
	}
	if tipTerm > inputTerm {
		return nil, invalidTermErr
	}
	targetHeight := tipHeight
	if s.enableChainedBFT && storage != nil {
		// 获取该block的ConsensusStorage
		justify, err := common.ParseOldQCStorage(storage)
		if err != nil {
			return nil, err
		}
		if justify.TargetBits != 0 {
			s.log.Debug("tdpos::CalOldProposers::use rollback target.")
			targetHeight = int64(justify.TargetBits)
		}
	}
	// 否则按照当前高度计算投票的top K
	p, err := s.calTopKNominator(targetHeight)
	if err != nil {
		s.log.Error("tdpos::CalOldProposers::calculateTopK err.", "err", err)
		return nil, err
	}
	s.log.Debug("tdpos::CalOldProposers::calTopKNominator", "p", p, "target height", targetHeight)
	return p, nil
}

// calHisValidators 根据term计算历史候选人信息
func (s *tdposSchedule) calHisValidators(height int64) ([]string, error) {
	// 设最近一次bucket vote变更的高度为H，所在term为T，候选人仅会在term+1的第一个高度H+M，试图变更候选人(不一定vote会引起变更)
	// 因此在输入一个height时，拿到当前term，查询当前term的第一个区块高度height-Z，通过height-Z获取TopK即可
	block, err := s.ledger.QueryBlockByHeight(height)
	if err != nil {
		s.log.Error("tdpos::CalculateProposers::QueryBlockByHeight err.", "err", err)
		return nil, err
	}
	term, pos, blockPos := s.minerScheduling(block.GetTimestamp())
	// 往前回溯的最远距离为internal，即该轮term之前最多生产过多少个区块
	internal := pos*s.blockNum + blockPos
	begin := block.GetHeight() - internal
	if begin <= s.startHeight {
		begin = s.startHeight
	}
	// 二分法实现快速查找
	targetHeight, err := s.binarySearch(begin, block.GetHeight(), term)
	if err != nil {
		return nil, err
	}
	s.log.Debug("tdpos::CalculateProposers::target height.", "height", height, "targetHeight", targetHeight, "term", term)
	return s.calTopKNominator(targetHeight)
}

// binarySearch 二分法快速查找
func (s *tdposSchedule) binarySearch(begin int64, end int64, term int64) (int64, error) {
	for begin < end {
		mid := begin + (end-begin)/2
		midTerm, err := s.getTerm(mid)
		if err != nil {
			return -1, err
		}
		nextMidTerm, err := s.getTerm(mid + 1)
		if err != nil {
			return -1, err
		}
		if midTerm < term && nextMidTerm == term {
			return mid + 1, nil
		}
		if midTerm < term {
			begin = mid + 1
		} else {
			end = mid
		}
	}
	return begin, nil
}

func (s *tdposSchedule) getTerm(pos int64) (int64, error) {
	b, err := s.ledger.QueryBlockByHeight(pos)
	if err != nil {
		return -1, err
	}
	in, err := ParseConsensusStorage(b)
	if err != nil {
		return -1, err
	}
	storage, ok := in.(*common.ConsensusStorage)
	if !ok {
		return -1, notFoundErr
	}
	return storage.CurTerm, nil
}
