package tdpos

import (
	"encoding/json"
	"sort"
	"sync"
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
	proposers []string
	netUrlMap map[string]string
	curTerm   int64

	// 增速使用的高度到proposers的映射，固定长度的slice，清理掉低height
	proposersMapping historyProposers
	mappingMutex     sync.Mutex

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
		proposers:         (xconfig.InitProposer)["1"],
		netUrlMap:         make(map[string]string),
		log:               log,
		ledger:            ledger,
	}
	index := 0
	netUrls := (xconfig.InitProposerNeturl)["1"]
	for index < len(schedule.proposers) {
		key := schedule.proposers[index]
		value := netUrls[index]
		schedule.netUrlMap[key] = value
		index++
	}

	// 重启时需要使用最新的validator数据，而不是initValidators数据
	tipHeight := schedule.ledger.GetTipBlock().GetHeight()
	refresh, err := schedule.calculateProposers(tipHeight)
	if err != nil && err != heightTooLow {
		schedule.log.Error("Tdpos::NewSchedule error", "err", err)
		return nil
	}

	if !common.AddressEqual(schedule.proposers, refresh) && len(refresh) != 0 {
		schedule.proposers = refresh
	}
	if xconfig.EnableBFT != nil {
		schedule.enableChainedBFT = true
	}
	return schedule
}

// miner 调度算法, 依据时间进行矿工节点调度
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
	posTime := s.alternateInterval + s.period*(s.blockNum-1)
	term = (T-initT)/termTime + 1
	resTime := (T - initT) - (term-1)*termTime
	pos = resTime / posTime
	resTime = resTime - (resTime/posTime)*posTime
	blockPos = resTime/s.period + 1
	return
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
	return proposers[pos]
}

// getSnapshotKey 获取当前tip高度的前三个区块高度的对应key的快照
func (s *tdposSchedule) getSnapshotKey(height int64, bucket string, key []byte) ([]byte, error) {
	if height <= 3 {
		return nil, heightTooLow
	}
	// 获取指定tipId的前三个区块
	block, err := s.ledger.QueryBlockByHeight(height - 3)
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
	// TODO: 具体合约未被调用过，初始化时需返回init参数
	if err != nil {
		s.log.Debug("tdpos::getSnapshotKey::reader.Get err.", "err", err)
		return nil, err
	}
	return versionData.PureData.Value, nil
}

// GetValidators election接口实现，获取指定round的候选人节点Address
func (s *tdposSchedule) GetValidators(round int64) []string {
	if round <= 3 {
		return s.proposers
	}
	// tdpos的validators变更在包含变更tx的block的后3个块后生效, 即当B0包含了变更tx，在B3时validators才正式统一变更
	// 查看增速映射里是否有对应的值
	s.mappingMutex.Lock()
	sort.Stable(s.proposersMapping)
	floor := -1
	for i, p := range s.proposersMapping {
		if p.height <= round {
			floor = i
			continue
		}
		break
	}
	if floor >= 0 {
		s.mappingMutex.Unlock()
		return s.proposersMapping[floor].proposers
	}
	s.mappingMutex.Unlock()

	// 否则读取快照
	proposers, err := s.calculateProposers(round)
	if err != nil {
		return nil
	}
	return proposers
}

// GetIntAddress election接口实现，获取候选人地址到网络地址的映射
func (s *tdposSchedule) GetIntAddress(address string) string {
	return s.netUrlMap[address]
}

// calculateProposers 根据vote选票信息计算最新的topK proposer, 调用方需检查height > 3
func (s *tdposSchedule) calculateProposers(height int64) ([]string, error) {
	if height <= 3 {
		return s.proposers, nil
	}
	// 获取候选人信息
	res, err := s.getSnapshotKey(height, contractBucket, []byte(nominateKey))
	if err != nil {
		s.log.Error("tdpos::calculateProposers::getSnapshotKey err.", "err", err)
		return nil, err
	}
	// 未读到值时直接返回初始化值
	if res == nil {
		return s.proposers, nil
	}
	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		s.log.Error("tdpos::calculateProposers::load nominate read set err.")
		return nil, err
	}
	var termBallotSli termBallotsSlice
	for candidate, _ := range nominateValue {
		candidateBallot := &termBallots{
			Address: candidate,
		}
		// 根据候选人信息获取vote选票信息
		key := voteKeyPrefix + candidate
		res, err := s.getSnapshotKey(height, contractBucket, []byte(key))
		if err != nil {
			s.log.Error("tdpos::calculateProposers::load vote read set err.")
			return nil, err
		}
		if res == nil {
			return s.proposers, nil
		}
		voteValue := NewvoteValue()
		if err := json.Unmarshal(res, &voteValue); err != nil {
			return nil, err
		}
		for _, ballot := range voteValue {
			candidateBallot.Ballots += ballot
		}
		termBallotSli = append(termBallotSli, candidateBallot)
	}
	if int64(termBallotSli.Len()) < s.proposerNum {
		s.log.Error("tdpos::calculateProposers::Term publish proposer num less than config", "termVotes", termBallotSli)
		return nil, proposerNotEnoughErr
	}
	// 计算topK候选人
	sort.Stable(termBallotSli)
	var proposers []string
	for i := int64(0); i < s.proposerNum; i++ {
		proposers = append(proposers, termBallotSli[i].Address)
	}
	s.mappingMutex.Lock()
	defer s.mappingMutex.Unlock()
	s.proposersMapping = append(s.proposersMapping, historyProposer{
		height:    height,
		proposers: proposers,
	})
	if len(s.proposersMapping) > MAXHISPROPOSERSSIZE {
		s.proposersMapping = s.proposersMapping[len(s.proposersMapping)-MAXHISPROPOSERSSIZE:]
	}
	return proposers, nil
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
	if !common.AddressEqual(nextProposers, s.proposers) {
		// 更新netURL
		res, err := s.getSnapshotKey(height, contractBucket, []byte(urlmapKey))
		if err != nil {
			s.log.Error("tdpos::updateProposers::getSnapshotKey error", "err", err)
			return false
		}
		if res == nil {
			return false
		}
		netURLValue := NewNetURLMap()
		if err := json.Unmarshal(res, &netURLValue); err != nil {
			s.log.Error("tdpos::updateProposers::unmarshal err.", "err", err)
			return false
		}
		for k, v := range netURLValue {
			s.netUrlMap[k] = v
		}
		s.proposers = nextProposers
		return true
	}
	return false
}

// notifyTermChanged 改变底层smr的候选人
func (s *tdposSchedule) notifyTermChanged(height int64) error {
	if !s.enableChainedBFT {
		// BFT not enabled, continue
		return nil
	}
	proposers, err := s.calculateProposers(height)
	if err != nil {
		return err
	}
	s.log.Debug("tdpos::notifyTermChanged", "s.proposers", s.proposers, "proposers", proposers)
	if !common.AddressEqual(proposers, s.proposers) {
		s.proposers = proposers
	}
	return nil
}
