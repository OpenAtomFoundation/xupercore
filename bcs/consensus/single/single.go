package single

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

/* 本次single改造支持single的升级，即Miner地址可变
 */

func init() {
	consensus.Register("single", NewSingleConsensus)
}

// SingleConsensus single为单点出块的共识逻辑
type SingleConsensus struct {
	ctx    cctx.ConsensusCtx
	status *SingleStatus
	config *SingleConfig
}

type SingleConfig struct {
	Miner   string `json:"miner"`
	Period  int64  `json:"period"`
	Version int64  `json:"version"`
}

type SingleStatus struct {
	beginHeight int64
	mutex       sync.RWMutex
	newHeight   int64
	index       int
	config      *SingleConfig
}

type MinerInfo struct {
	Miner string `json:"miner"`
}

// GetVersion 返回pow所在共识version
func (s *SingleStatus) GetVersion() int64 {
	return s.config.Version
}

// GetConsensusBeginInfo 返回该实例初始高度
func (s *SingleStatus) GetConsensusBeginInfo() int64 {
	return s.beginHeight
}

// GetStepConsensusIndex 获取共识item所在consensus slice中的index
func (s *SingleStatus) GetStepConsensusIndex() int {
	return s.index
}

// GetConsensusName 获取共识类型
func (s *SingleStatus) GetConsensusName() string {
	return "single"
}

// GetCurrentTerm 获取当前状态机term
func (s *SingleStatus) GetCurrentTerm() int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.newHeight
}

// GetCurrentValidatorsInfo 获取当前矿工信息
func (s *SingleStatus) GetCurrentValidatorsInfo() []byte {
	miner := MinerInfo{
		Miner: s.config.Miner,
	}
	m, _ := json.Marshal(miner)
	return m
}

// NewSingleConsensus 初始化实例
func NewSingleConsensus(cCtx context.ConsensusCtx, cCfg context.ConsensusConfig) base.ConsensusImplInterface {
	// 解析config中需要的字段
	if cCtx.BCtx.XLog == nil {
		return nil
	}
	// TODO:cCtx.BcName需要注册表吗？
	if cCtx.CryptoClient == nil {
		cCtx.BCtx.XLog.Error("Single::NewSingleConsensus::CryptoClient in context is nil")
		return nil
	}
	if cCtx.Ledger == nil {
		cCtx.BCtx.XLog.Error("Single::NewSingleConsensus::Ledger in context is nil")
		return nil
	}
	if cCfg.ConsensusName != "single" {
		cCtx.BCtx.XLog.Error("Single::NewSingleConsensus::consensus name in config is wrong", "name", cCfg.ConsensusName)
		return nil
	}
	config := &SingleConfig{}
	err := json.Unmarshal([]byte(cCfg.Config), config)
	if err != nil {
		cCtx.BCtx.XLog.Error("Single::NewSingleConsensus::pow struct unmarshal error", "error", err)
		return nil
	}
	// newHeight取上一共识的最高值，因为此时BeginHeight也许并为生产出来
	status := &SingleStatus{
		beginHeight: cCfg.BeginHeight,
		newHeight:   cCfg.BeginHeight - 1,
		index:       cCfg.Index,
		config:      config,
	}
	single := &SingleConsensus{
		ctx:    cCtx,
		config: config,
		status: status,
	}
	return single
}

/* CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
 * 该函数返回两个bool，第一个表示是否当前应当出块，第二个表示是否当前需要向其他节点同步区块
 */
func (s *SingleConsensus) CompeteMaster(height int64) (bool, bool, error) {
	t := time.Now().UnixNano() / 1e6
	if t%s.config.Period != 0 {
		return false, false, nil
	}
	if s.ctx.P2p.GetLocalAddress() == s.config.Miner {
		// single共识确定miner后只能通过共识升级改变miner，因此在单个single实例中miner是不可更改的
		// 此时一个miner从始至终都是自己在挖矿，故不需要向其他节点同步区块
		return true, false, nil
	}
	return false, false, nil
}

// CheckMinerMatch 查看block是否合法
func (s *SingleConsensus) CheckMinerMatch(ctx xcontext.BaseCtx, block cctx.BlockInterface) (bool, error) {
	// 检查区块的区块头是否hash正确
	if !bytes.Equal(block.MakeBlockId(), block.GetBlockid()) {
		ctx.XLog.Warn("Single::CheckMinerMatch::equal blockid error", "logid", ctx.XLog.GetLogId())
		return false, nil
	}
	// 检查矿工地址是否合法
	if block.GetProposer() != s.config.Miner {
		ctx.XLog.Warn("Single::CheckMinerMatch::miner check error", "logid", ctx.XLog.GetLogId(),
			"blockid", block.GetBlockid(), "proposer", block.GetProposer(), "local proposer", s.config.Miner)
		return false, nil
	}
	// 验证merkel根是否计算正确
	if err := s.ctx.Ledger.VerifyMerkle(block); err != nil {
		ctx.XLog.Warn("Single::CheckMinerMatch::VerifyMerkle error", "logid", ctx.XLog.GetLogId(), "error", err)
		return false, nil
	}
	//验证签名
	//1 验证一下签名和公钥是不是能对上
	k, err := s.ctx.CryptoClient.GetEcdsaPublicKeyFromJSON(block.GetPubkey())
	if err != nil {
		ctx.XLog.Warn("Single::CheckMinerMatch::get ecdsa from block error", "logid", ctx.XLog.GetLogId(), "error", err)
		return false, nil
	}
	//Todo 跟address比较
	chkResult, _ := s.ctx.CryptoClient.VerifyAddressUsingPublicKey(string(block.GetProposer()), k)
	if chkResult == false {
		ctx.XLog.Warn("Single::CheckMinerMatch::address is not match publickey", "logid", ctx.XLog.GetLogId())
		return false, nil
	}
	//2 验证一下签名是否正确
	valid, err := s.ctx.CryptoClient.VerifyECDSA(k, block.GetSign(), block.GetBlockid())
	if err != nil {
		ctx.XLog.Warn("Single::CheckMinerMatch::verifyECDSA error", "logid", ctx.XLog.GetLogId(), "error", err)
	}
	return valid, nil
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回是否需要truncate, 返回写consensusStorage, 返回err
func (s *SingleConsensus) ProcessBeforeMiner(timestamp int64) (bool, []byte, error) {
	return false, nil, nil
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
func (s *SingleConsensus) CalculateBlock(block cctx.BlockInterface) error {
	return nil
}

// ProcessConfirmBlock 用于确认块后进行相应的处理
func (s *SingleConsensus) ProcessConfirmBlock(block cctx.BlockInterface) error {
	return nil
}

// GetStatus 获取区块链共识信息
func (s *SingleConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	return s.status, nil
}

// 共识实例的挂起逻辑, 另: 若共识实例发现绑定block结构有误，会直接停掉当前共识实例并panic
func (s *SingleConsensus) Stop() error {
	return nil
}

// 共识实例的重启逻辑, 用于共识回滚
func (s *SingleConsensus) Start() error {
	return nil
}

/* ParseConsensusStorage 共识占用blockinterface的专有存储，特定共识需要提供parse接口，在此作为接口高亮
 * Single共识没有用到区块存储信息, 故返回空
 */
func (s *SingleConsensus) ParseConsensusStorage(block cctx.BlockInterface) (interface{}, error) {
	return nil, nil
}
