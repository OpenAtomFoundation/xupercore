package xpoa

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cCrypto "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/crypto"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

// TODO: 目前时间关系，暂时将xpoa放置在三代合约，后可以收敛至xuper5的kernel合约
const (
	/* 为了避免调用utxoVM systemCall方法, 直接通过ledger读取xpoa合约存储
	 * ATTENTION: 此处xpoaBucket和xpoaKey必须和对应三代合约严格一致，并且该xpoa隐式限制只能包含xmodel机制的ledger才可调用
	 */
	xpoaBucket = "xpoa"
	xpoaKey    = "VALIDATES"
	Keypath    = "FAKE"
)

var (
	MinerSelectErr   = errors.New("Node isn't a miner, calculate error.")
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
)

func init() {
	consensus.Register("xpoa", NewXpoaConsensus)
}

// XpoaStorage xpoa占用block中consensusStorage json串的格式
type XpoaStorage struct {
	Justify *chainedBft.QuorumCert `json:"Justify,omitempty"`
}

type ProposerInfo struct {
	Address string `json:"address"`
	Neturl  string `json:"neturl"`
}

type ValidatorsInfo struct {
	Validators []*ProposerInfo `json:"validators"`
	Miner      *ProposerInfo   `json:"miner"`
}

// XpoaStatus 实现了ConsensusStatus接口
type XpoaStatus struct {
	Version     int64 `json:"version"`
	BeginHeight int64 `json:"beginHeight"`
	Index       int   `json:"index"`
	mutex       sync.RWMutex
	election    *xpoaSchedule
}

// 获取共识版本号
func (x *XpoaStatus) GetVersion() int64 {
	return x.Version
}

// 共识起始高度
func (x *XpoaStatus) GetConsensusBeginInfo() int64 {
	return x.BeginHeight
}

// 获取共识item所在consensus slice中的index
func (x *XpoaStatus) GetStepConsensusIndex() int {
	return x.Index
}

// 获取共识类型
func (x *XpoaStatus) GetConsensusName() string {
	return "xpoa"
}

// 获取当前状态机term
func (x *XpoaStatus) GetCurrentTerm() int64 {
	term, _, _ := x.election.minerScheduling(time.Now().UnixNano(), len(x.election.validators))
	return term
}

// 获取当前矿工信息
func (x *XpoaStatus) GetCurrentValidatorsInfo() []byte {
	var v []*ProposerInfo
	for _, a := range x.election.validators {
		v = append(v, &ProposerInfo{
			Address: a,
			Neturl:  x.election.addrToNet[a],
		})
	}
	i := ValidatorsInfo{
		Miner: &ProposerInfo{
			Address: x.election.miner,
			Neturl:  x.election.addrToNet[x.election.miner],
		},
		Validators: v,
	}
	b, _ := json.Marshal(i)
	return b
}

// NewXpoaConsensus 初始化实例
func NewXpoaConsensus(cCtx context.ConsensusCtx, cCfg context.ConsensusConfig) base.ConsensusImplInterface {
	// 解析config中需要的字段
	if cCtx.XLog == nil {
		return nil
	}
	// TODO:cCtx.BcName需要注册表吗？
	if cCtx.CryptoClient == nil {
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
	// TODO: 此处一个CryptoClient对应的publicKey和privateKey应该是确定的，而不是度字符串和文件
	pkJSON, err := ioutil.ReadFile(Keypath + "/public.key")
	if err != nil {
		return nil
	}
	skJSON, err := ioutil.ReadFile(Keypath + "/private.key")
	if err != nil {
		return nil
	}

	// 解析xpoaconfig
	xconfig := &xpoaConfig{}
	err = json.Unmarshal([]byte(cCfg.Config), xconfig)
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
	reader := schedule.ledger.GetTipSnapShot()
	res, err := reader.Get(xpoaBucket, []byte(xpoaKey))
	if snapshotValidators, _ := loadValidatorsMultiInfo(res, &schedule.addrToNet); snapshotValidators != nil {
		validators = snapshotValidators
	}
	schedule.validators = validators

	// create smr/ chained-bft实例, 需要新建CBFTCrypto、pacemaker和saftyrules实例
	cryptoClient := cCrypto.NewCBFTCrypto(schedule.address, cCtx.CryptoClient, string(pkJSON), string(skJSON))
	pacemaker := &chainedBft.DefaultPaceMaker{
		StartView: cCtx.StartHeight,
	}
	saftyrules := &chainedBft.DefaultSaftyRules{
		Crypto: cryptoClient,
	}
	smr := chainedBft.NewSmr(cCtx.BcName, schedule.address, cCtx.XLog, cCtx.P2p, cryptoClient, pacemaker, saftyrules, schedule, initQCTree(cCtx.StartHeight, cCtx.Ledger))

	// create xpoaConsensus实例
	xpoa := &xpoaConsensus{
		election:      schedule,
		isProduce:     make(map[int64]bool),
		config:        xconfig,
		initTimestamp: time.Now().UnixNano(),
		smr:           smr,
	}
	return xpoa
}

func loadValidatorsMultiInfo(res []byte, addrToNet *map[string]string) ([]string, error) {
	if res == nil {
		return nil, NotValidContract
	}
	// 读取最新的validators值
	contractInfo := ProposerInfo{}
	if err := json.Unmarshal(res, &contractInfo); err != nil {
		return nil, err
	}
	validators := strings.Split(contractInfo.Address, ";") // validators由分号隔开
	if len(validators) == 0 {
		return nil, EmptyValidors
	}
	neturls := strings.Split(contractInfo.Neturl, ";") // neturls由分号隔开
	if len(neturls) != len(validators) {
		return nil, EmptyValidors
	}
	for i, v := range validators {
		(*addrToNet)[v] = neturls[i]
	}
	return validators, nil
}

// initQCTree 创建了smr需要的QC树存储，该Tree存储了目前待commit的QC信息
func initQCTree(startHeight int64, ledger cctx.LedgerCtxInConsensus) *chainedBft.QCPendingTree {
	// 初始状态，应该是start高度的前一个区块为genesisQC
	b, _ := ledger.QueryBlockByHeight(startHeight - 1)
	initQC := &chainedBft.QuorumCert{
		VoteInfo: &chainedBft.VoteInfo{
			ProposalId:   b.GetBlockid(),
			ProposalView: startHeight - 1,
		},
		LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
			CommitStateId: b.GetBlockid(),
		},
	}
	rootNode := &chainedBft.ProposalNode{
		In: initQC,
	}
	return &chainedBft.QCPendingTree{
		Genesis:  rootNode,
		Root:     rootNode,
		HighQC:   rootNode,
		CommitQC: rootNode,
	}
}

type xpoaConfig struct {
	InitProposer []ProposerInfo `json:"init_proposer"`
	BlockNum     int64          `json:"block_num"`
	Period       int64          `json:"period"`
}

type xpoaConsensus struct {
	election      *xpoaSchedule
	smr           *chainedBft.Smr
	isProduce     map[int64]bool
	config        *xpoaConfig
	initTimestamp int64
	status        *XpoaStatus
}

// xpoaSchedule 实现了ProposerElectionInterface接口，接口定义了validators操作
type xpoaSchedule struct {
	address    string
	newHeight  int64
	period     int64             // 出块间隔
	blockNum   int64             // 每轮每个候选人最多出多少块
	validators []string          // 当前validators的address
	miner      string            //当前Leader
	addrToNet  map[string]string // address到neturl的映射

	ledger cctx.LedgerCtxInConsensus
}

// minerScheduling 按照时间调度计算目标候选人轮换数term, 目标候选人index和候选人生成block的index
func (s *xpoaSchedule) minerScheduling(timestamp int64, length int) (term int64, pos int64, blockPos int64) {
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
 * 由于GetLeader()永远在GetMsgAddress()之前，故在GetLeader时更新schedule的addrToNet Map，可以保证能及时提供Addr到NetUrl的映射
 */
func (s *xpoaSchedule) GetLeader(round int64) string {
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
		nextValidators, err := s.getValidatesByBlockId(b.GetBlockid())
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
func (s *xpoaSchedule) GetLocalLeader(timestamp int64, round int64) string {
	// xpoa.lg.Info("ConfirmBlock Propcess update validates")
	// ATTENTION: 获取候选人信息时，时刻注意拿取的是check目的round的前三个块，候选人变更是在3个块之后生效，即round-3
	b, err := s.ledger.QueryBlockByHeight(round - 3)
	if err != nil {
		return ""
	}
	localValidators, err := s.getValidatesByBlockId(b.GetBlockid())
	if localValidators == nil && err == nil {
		// 使用初始变量
		return ""
	}
	_, pos, _ := s.minerScheduling(timestamp, len(localValidators))
	return localValidators[pos]
}

// getValidatesByBlockId 根据当前输入blockid，用快照的方式在xmodel中寻找<=当前blockid的最新的候选人值，若无则使用xuper.json中指定的初始值
func (s *xpoaSchedule) getValidatesByBlockId(blockId []byte) ([]string, error) {
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
	validators, err := loadValidatorsMultiInfo(res, &s.addrToNet)
	if err != nil {
		return nil, err
	}
	return validators, nil
}

func (s *xpoaSchedule) GetValidators(round int64) []string {
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

func (s *xpoaSchedule) GetMsgAddress(addr string) string {
	return s.addrToNet[addr]
}

func (s *xpoaSchedule) GetValidatorsMsgAddr() []string {
	var urls []string
	for _, v := range s.validators {
		urls = append(urls, s.addrToNet[v])
	}
	return urls
}

func (s *xpoaSchedule) UpdateValidatorSet(newValidates []string) {

}

// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
func (x *xpoaConsensus) CompeteMaster(height int64) (bool, bool, error) {
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
// ATTENTION: TODO: 上层需要先检查VerifyBlock(block)
func (x *xpoaConsensus) CheckMinerMatch(ctx xcontext.BaseCtx, block cctx.BlockInterface) (bool, error) {
	// TODO: 应由saftyrules模块负责check, xpoa需要组合一个defaultsaftyrules, 在saftyrules里调用ledger的verifyBlock
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
	/*
		pNode := x.smr.BlockToProposalNode(block)
		err := x.smr.GetSaftyRules().IsQuorumCertValidate(pNode.In, justify.Justify)
		if err != nil {
			// xpoa.lg.Warn("CheckMinerMatch bft IsQuorumCertValidate failed", "logid", header.Logid, "error", err)
			return false, nil
		}
	*/
	return proposer == block.GetProposer(), nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回是否需要truncate, 返回写consensusStorage, 返回err
func (x *xpoaConsensus) ProcessBeforeMiner(timestamp int64) (bool, []byte, error) {
	// 再次检查目前是否是矿工，TODO: check是否有必要，因为和sync抢一把锁，按道理不会有这个问题
	_, pos, _ := x.election.minerScheduling(timestamp, len(x.election.validators))
	if x.election.validators[pos] != x.election.address {
		return false, nil, MinerSelectErr
	}
	// 即本地smr的HightQC和账本TipId不相等，tipId尚未收集到足够签名，回滚到本地HighQC
	if !bytes.Equal(x.smr.GetHighQC().GetProposalId(), x.election.ledger.GetTipBlock().GetBlockid()) {
		/*
			if len(xpoa.proposerInfos) == 1 {
				res["quorum_cert"] = nil
				return res, true
			}
		*/
		// xpoa.lg.Warn("ProcessBeforeMiner last block not confirmed, walk to previous block")
		// targetId := x.smr.GetHighQC().GetProposalId()
		return true, nil, nil
	}
	qc := x.smr.GetHighQC()
	qcQuorumCert, ok := qc.(*chainedBft.QuorumCert)
	if !ok {

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
	x.status.mutex.Lock()
	defer x.status.mutex.Unlock()
	if block.GetHeight() > x.election.newHeight {
		x.election.newHeight = block.GetHeight()
	}

	// 查看本地是否是最新round的生产者
	_, pos, _ := x.election.minerScheduling(block.GetTimestamp(), len(x.election.validators))
	if x.election.validators[pos] == x.election.address && block.GetProposer() == x.election.address {
		// 如果是当前矿工，检测到下一轮需变更validates，且下一轮proposer并不在节点列表中，此时需在广播列表中新加入节点
		validators := x.election.GetValidators(block.GetHeight() + 1)
		b, err := x.election.ledger.QueryBlockByHeight(block.GetHeight() - 3)
		if err == nil {
			if v, err := x.election.getValidatesByBlockId(b.GetBlockid()); err == nil {
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
	return nil
}

// 共识实例的挂起逻辑, 另: 若共识实例发现绑定block结构有误，会直接停掉当前共识实例并panic
func (x *xpoaConsensus) Stop() error {
	return nil
}

// 共识实例的重启逻辑, 用于共识回滚
func (x *xpoaConsensus) Start() error {
	return nil
}

// 共识占用blockinterface的专有存储，特定共识需要提供parse接口，在此作为接口高亮
func (x *xpoaConsensus) ParseConsensusStorage(block cctx.BlockInterface) (interface{}, error) {
	return nil, nil
}

func (x *xpoaConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	return x.status, nil
}

func cleanProduceMap(isProduce map[int64]bool) {

}

// AddressEqual 判断两个validators地址是否相等
func AddressEqual(a []string, b []string) bool {
	return true
}
