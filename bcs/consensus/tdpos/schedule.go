package tdpos

import (
	"encoding/json"
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
	validators     []string
	initValidators []string
	curTerm        int64
	miner          string

	log    logs.Logger
	ledger cctx.LedgerRely
}

// NewSchedule 新建schedule实例
func NewSchedule(xconfig *tdposConfig, log logs.Logger, ledger cctx.LedgerRely) *tdposSchedule {
	schedule := &tdposSchedule{
		period:            xconfig.Period,
		blockNum:          xconfig.BlockNum,
		proposerNum:       xconfig.ProposerNum,
		alternateInterval: xconfig.AlternateInterval,
		termInterval:      xconfig.TermInterval,
		initTimestamp:     xconfig.InitTimestamp,
		validators:        (xconfig.InitProposer)["1"],
		log:               log,
		ledger:            ledger,
	}
	index := 0
	for index < len(schedule.validators) {
		key := schedule.validators[index]
		schedule.initValidators = append(schedule.initValidators, key)
		index++
	}

	// 重启时需要使用最新的validator数据，而不是initValidators数据
	tipHeight := schedule.ledger.GetTipBlock().GetHeight()
	refresh, err := schedule.calculateProposers(tipHeight)
	if err != nil && err != heightTooLow {
		schedule.log.Error("Tdpos::NewSchedule error", "err", err)
		return nil
	}

	if !common.AddressEqual(schedule.validators, refresh) && len(refresh) != 0 {
		schedule.validators = refresh
	}
	if xconfig.EnableBFT != nil {
		schedule.enableChainedBFT = true
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
	if versionData == nil {
		return nil, nil
	}
	return versionData.PureData.Value, nil
}

// GetLeader 根据输入的round，计算应有的proposer，实现election接口
// 该方法主要为了支撑smr扭转和矿工挖矿，在handleReceivedProposal阶段会调用该方法
// 由于xpoa主逻辑包含回滚逻辑，因此回滚逻辑必须在ProcessProposal进行
// ATTENTION: tipBlock是一个隐式依赖状态
// ATTENTION: 由于GetLeader()永远在GetIntAddress()之前，故在GetLeader时更新schedule的addrToNet Map，可以保证能及时提供Addr到NetUrl的映射
func (s *tdposSchedule) GetLeader(round int64) string {
	// 若该round已经落盘，则直接返回历史信息，eg. 矿工在当前round的情况
	if b, err := s.ledger.QueryBlockByHeight(round); err == nil {
		return string(b.GetProposer())
	}
	tipBlock := s.ledger.GetTipBlock()
	tipHeight := tipBlock.GetHeight()
	proposers := s.GetValidators(round)
	if proposers == nil {
		return ""
	}
	nTime := time.Now().UnixNano()
	if round > tipHeight {
		// s.period为毫秒单位
		nTime += s.period * int64(time.Millisecond)
	}
	_, pos, _ := s.minerScheduling(nTime)
	if pos >= s.proposerNum {
		return ""
	}
	return proposers[pos]
}

// GetValidators election接口实现，获取指定round的候选人节点Address
func (s *tdposSchedule) GetValidators(round int64) []string {
	if round <= 3 {
		return s.initValidators
	}
	proposers, err := s.calculateProposers(round)
	if err != nil {
		return nil
	}
	return proposers
}

// GetIntAddress election接口实现，获取候选人地址到网络地址的映射
func (s *tdposSchedule) GetIntAddress(address string) string {
	return ""
}

// updateProposers 根据各合约存储计算当前proposers
func (s *tdposSchedule) UpdateProposers(height int64) bool {
	if height <= 3 {
		return false
	}
	nextProposers, err := s.calculateProposers(height)
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

// calculateProposers 根据vote选票信息计算最新的topK proposer, 调用方需检查height > 3
// 若提案投票人数总和小于指定数目，则统一仍使用初始值
// 或者根据历史termKey获取历史候选人集合
func (s *tdposSchedule) calculateProposers(height int64) ([]string, error) {
	if height <= 3 {
		return s.initValidators, nil
	}
	nowTerm, _, _ := s.minerScheduling(time.Now().UnixNano())
	// 情况一：目标height尚未产生，计算新一轮值
	if s.ledger.GetTipBlock().GetHeight() < height {
		if s.curTerm == nowTerm {
			// 当前并不需要更新候选人
			s.log.Debug("tdpos::calculateProposers::s.curTerm == nowTerm")
			return s.validators, nil
		}
		// 更新候选人时需要读取快照，并计算投票的top K
		p, err := s.calTopKNominator(height)
		if err != nil {
			s.log.Error("tdpos::calculateProposers::calculateTopK err.", "err", err)
			return nil, err
		}
		s.log.Debug("tdpos::calculateProposers::calTopKNominator", "p", p)
		return p, nil
	}
	// 情况二：读取历史值，此时分成历史Key读取和计算差值两部分
	b, err := s.ledger.QueryBlockByHeight(height)
	if err != nil {
		s.log.Error("tdpos::calculateProposers::QueryBlockByHeight err.", "err", err)
		return nil, err
	}
	term, _, _ := s.minerScheduling(b.GetTimestamp())
	return s.calHisValidators(height, term)
}

// calTopKNominator 计算最新的投票的topK候选人并返回
func (s *tdposSchedule) calTopKNominator(height int64) ([]string, error) {
	// 获取nominate信息
	res, err := s.getSnapshotKey(height-3, contractBucket, []byte(nominateKey))
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
		key := voteKeyPrefix + candidate
		res, err := s.getSnapshotKey(height-3, contractBucket, []byte(key))
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

// calHisValidators 根据termKey计算历史候选人信息
// ...｜<----termK---->｜...｜<----termK+N---->｜...|<----termK+M---->|<---termK+M+1--->
// ...|..VoteA....VoteB|....|......VoteC......|....|.....VoteD.......|................
// ...|B1|B2|..........|Bn..|Bm|Bm+1|.........|Bk|.|Bv|Bv+1|.........|Bx|Bx+1|Bx+2|...
// ..........term1.....|term2|.....term3......|term4|.....term5......|......term6+....
func (s *tdposSchedule) calHisValidators(height int64, inputTerm int64) ([]string, error) {
	// 获取term信息
	res, err := s.getSnapshotKey(height, contractBucket, []byte(termKey))
	if err != nil {
		s.log.Error("tdpos::calHisValidators::getSnapshotKey err.", "err", err)
		return nil, err
	}
	if res == nil {
		return s.initValidators, nil
	}
	termV := NewTermValue()
	if err := json.Unmarshal(res, &termV); err != nil {
		s.log.Error("tdpos::calHisValidators::Unmarshal err.", "err", err)
		return nil, err
	}
	// 如果slice最后一个值仍比当前值小，则需查找计算
	if termV[len(termV)-1].Term < inputTerm {
		// 此时根据最后一次记录term的height向后查找，然后计算候选人值
		beginH := termV[len(termV)-1].Height
		for {
			beginH = beginH + 1
			b, err := s.ledger.QueryBlockByHeight(beginH)
			// 未找到更大的term，此时应该返回当前值
			if err != nil {
				s.log.Warn("tdpos::calHisValidators::QueryBlockByHeight err.", "err", err)
				return termV[len(termV)-1].Validators, nil
			}
			bv, _ := b.GetConsensusStorage()
			tdposStorage, err := common.ParseOldQCStorage(bv)
			if err != nil {
				s.log.Error("tdpos::calHisValidators::ParseOldQCStorage err.", "err", err)
				return nil, err
			}
			if tdposStorage.CurTerm > termV[len(termV)-1].Term {
				return s.calTopKNominator(b.GetHeight())
			}
		}
	}
	// 如果在历史值区间，则可直接计算结果
	index := 0
	for i, item := range termV {
		if item.Term <= inputTerm {
			index = i
			continue
		}
		break
	}
	return termV[index].Validators, nil
}
